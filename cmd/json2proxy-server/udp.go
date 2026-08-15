package main

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"json2proxy/internal/config"
	"json2proxy/internal/protocol"
	"json2proxy/internal/socks5"
)

type udpRateState struct {
	updated      time.Time
	packetTokens float64
	byteTokens   float64
}

type udpDNSCacheEntry struct {
	addresses []net.IP
	expires   time.Time
	negative  bool
}

type udpServerStats struct {
	packetsUp, packetsDown atomic.Uint64
	bytesUp, bytesDown     atomic.Uint64
	queueDrops             atomic.Uint64
	rateDrops              atomic.Uint64
	policyDrops            atomic.Uint64
	oversizeDrops          atomic.Uint64
}

type udpEgress interface {
	WriteDatagram(string, uint16, []byte) error
	ReadDatagram(time.Time, []byte) (string, uint16, []byte, error)
	Close() error
}

type directPacket struct {
	host string
	port uint16
	data []byte
}

type directUDPEgress struct {
	v4, v6 *net.UDPConn
	queue  chan directPacket
	done   chan struct{}
	once   sync.Once
	drops  *atomic.Uint64
}

func newDirectUDPEgress(queueSize, maxDatagram int, drops *atomic.Uint64) (*directUDPEgress, error) {
	d := &directUDPEgress{queue: make(chan directPacket, queueSize), done: make(chan struct{}), drops: drops}
	d.v4, _ = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	d.v6, _ = net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6unspecified})
	if d.v4 == nil && d.v6 == nil {
		return nil, errors.New("cannot open UDP4 or UDP6 socket")
	}
	if d.v4 != nil {
		go d.readLoop(d.v4, maxDatagram)
	}
	if d.v6 != nil {
		go d.readLoop(d.v6, maxDatagram)
	}
	return d, nil
}

func (d *directUDPEgress) readLoop(conn *net.UDPConn, maxDatagram int) {
	buffer := make([]byte, 65535)
	for {
		n, source, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		if n > maxDatagram {
			d.drops.Add(1)
			continue
		}
		packet := directPacket{
			host: source.IP.String(), port: uint16(source.Port),
			data: append([]byte(nil), buffer[:n]...),
		}
		select {
		case d.queue <- packet:
		default:
			d.drops.Add(1)
		}
	}
}

func (d *directUDPEgress) WriteDatagram(host string, port uint16, data []byte) error {
	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("direct UDP target must be a numeric IP")
	}
	address := &net.UDPAddr{IP: ip, Port: int(port)}
	conn := d.v6
	if ip.To4() != nil {
		conn = d.v4
	}
	if conn == nil {
		return errors.New("UDP address family unavailable")
	}
	_, err := conn.WriteToUDP(data, address)
	return err
}

func (d *directUDPEgress) ReadDatagram(deadline time.Time, _ []byte) (string, uint16, []byte, error) {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case packet := <-d.queue:
		return packet.host, packet.port, packet.data, nil
	case <-d.done:
		return "", 0, nil, net.ErrClosed
	case <-timer.C:
		return "", 0, nil, &net.DNSError{IsTimeout: true, Err: "UDP read timeout"}
	}
}

func (d *directUDPEgress) Close() error {
	var first error
	d.once.Do(func() {
		close(d.done)
		if d.v4 != nil {
			first = d.v4.Close()
		}
		if d.v6 != nil {
			if err := d.v6.Close(); first == nil {
				first = err
			}
		}
	})
	return first
}

func (a *app) serveUDP(tunnel net.Conn, writer *frameWriter, session *protocol.Session, user config.User, source string, started time.Time) {
	releaseUDP, ok := a.claimUDP(user.ClientID)
	if !ok {
		_ = writer.frame(protocol.Frame{Type: protocol.FrameUDPOpenResult, Payload: protocol.Result(3, "UDP association limit exceeded")})
		a.log.Printf("connection rejected source=%s user=%s session=%s udp=true reason=udp_association_limit", source, user.ClientID, session.ID())
		return
	}
	defer releaseUDP()

	stats := &udpServerStats{}
	var egress udpEgress
	var err error
	if a.outboundSOCKS != nil {
		ctx, cancel := context.WithTimeout(context.Background(), a.dialTimeout)
		egress, err = a.outboundSOCKS.AssociateUDP(ctx)
		cancel()
	} else {
		egress, err = newDirectUDPEgress(a.cfg.UDP.QueuePackets, a.cfg.UDP.MaxDatagramSize, &stats.queueDrops)
	}
	if err != nil {
		code := byte(1)
		if a.outboundSOCKS != nil {
			code = 5
		}
		_ = writer.frame(protocol.Frame{Type: protocol.FrameUDPOpenResult, Payload: protocol.Result(code, "UDP egress unavailable")})
		a.log.Printf("connection failed source=%s target=- user=%s session=%s udp=true egress=%s egress_proxy=%s reason=udp_egress_unavailable error=%q",
			source, user.ClientID, session.ID(), a.egressType(), a.egressProxy(), err)
		return
	}
	defer egress.Close()
	if err = writer.frame(protocol.Frame{Type: protocol.FrameUDPOpenResult, Payload: protocol.Result(0, "")}); err != nil {
		return
	}
	_ = tunnel.SetDeadline(time.Time{})

	recent := make(map[string]time.Time)
	var recentMu sync.Mutex
	var lastTargetMu sync.Mutex
	lastTarget := "-"
	lastActivity := atomic.Int64{}
	lastActivity.Store(time.Now().UnixNano())
	errCh := make(chan error, 3)
	done := make(chan struct{})
	var closeOnce sync.Once
	var workers sync.WaitGroup
	finish := func(err error) {
		closeOnce.Do(func() {
			close(done)
			_ = egress.Close()
			_ = tunnel.Close()
			errCh <- err
		})
	}

	a.log.Printf("connection established source=%s target=- user=%s session=%s udp=true egress=%s egress_proxy=%s",
		source, user.ClientID, session.ID(), a.egressType(), a.egressProxy())

	workers.Add(3)
	go func() {
		defer workers.Done()
		for {
			frame, readErr := readFrame(writer.wire, session)
			if readErr != nil {
				finish(readErr)
				return
			}
			switch frame.Type {
			case protocol.FrameUDPDatagram:
				host, port, data, parseErr := protocol.ParseUDPDatagram(frame.Payload)
				if parseErr != nil || len(data) > a.cfg.UDP.MaxDatagramSize {
					_ = writer.presetError(errorInvalidRequest, "invalid request")
					finish(errors.New("invalid UDP datagram"))
					return
				}
				if !a.allowUDPRate(user.ClientID, len(data)) {
					stats.rateDrops.Add(1)
					continue
				}
				address, resolved, resolveErr := a.resolveUDP(host, port, user.AllowPrivate)
				if resolveErr != nil {
					if errors.Is(resolveErr, errPrivateTarget) {
						_ = writer.presetError(errorPrivateTarget, "private target denied")
						finish(resolveErr)
						return
					}
					stats.policyDrops.Add(1)
					continue
				}
				recentMu.Lock()
				now := time.Now()
				idleTimeout := a.udpIdleTimeout()
				for endpoint, lastSeen := range recent {
					if now.Sub(lastSeen) > idleTimeout {
						delete(recent, endpoint)
					}
				}
				_, exists := recent[resolved]
				if !exists && len(recent) >= a.cfg.UDP.MaxDestinationsPerAssociation {
					recentMu.Unlock()
					stats.policyDrops.Add(1)
					continue
				}
				recentMu.Unlock()
				if writeErr := egress.WriteDatagram(address.IP.String(), uint16(address.Port), data); writeErr != nil {
					stats.policyDrops.Add(1)
					continue
				}
				now = time.Now()
				recentMu.Lock()
				recent[resolved] = now
				recentMu.Unlock()
				lastTargetMu.Lock()
				lastTarget = resolved
				lastTargetMu.Unlock()
				stats.packetsUp.Add(1)
				stats.bytesUp.Add(uint64(len(data)))
				lastActivity.Store(now.UnixNano())
			case protocol.FramePing:
				if writeErr := writer.frame(protocol.Frame{Type: protocol.FramePong, Payload: frame.Payload}); writeErr != nil {
					finish(writeErr)
					return
				}
			case protocol.FrameUDPClose:
				finish(io.EOF)
				return
			case protocol.FrameReset:
				finish(errors.New("client reset"))
				return
			default:
				_ = writer.presetError(errorInvalidRequest, "invalid request")
				finish(errors.New("invalid UDP frame"))
				return
			}
		}
	}()

	go func() {
		defer workers.Done()
		buffer := make([]byte, 65535+300)
		for {
			host, port, data, readErr := egress.ReadDatagram(time.Now().Add(time.Second), buffer)
			if readErr != nil {
				if timeout, ok := readErr.(net.Error); ok && timeout.Timeout() {
					select {
					case <-done:
						return
					default:
						continue
					}
				}
				finish(readErr)
				return
			}
			if len(data) > a.cfg.UDP.MaxDatagramSize {
				stats.oversizeDrops.Add(1)
				continue
			}
			endpoint := net.JoinHostPort(host, strconv.Itoa(int(port)))
			now := time.Now()
			recentMu.Lock()
			expiry, allowed := recent[endpoint]
			if allowed && now.Sub(expiry) > a.udpIdleTimeout() {
				delete(recent, endpoint)
				allowed = false
			}
			recentMu.Unlock()
			if !allowed {
				stats.policyDrops.Add(1)
				continue
			}
			if !a.allowUDPRate(user.ClientID, len(data)) {
				stats.rateDrops.Add(1)
				continue
			}
			payload, encodeErr := protocol.UDPDatagramPayload(host, port, data)
			if encodeErr != nil {
				stats.oversizeDrops.Add(1)
				continue
			}
			if writeErr := writer.frame(protocol.Frame{Type: protocol.FrameUDPDatagram, Payload: payload}); writeErr != nil {
				finish(writeErr)
				return
			}
			stats.packetsDown.Add(1)
			stats.bytesDown.Add(uint64(len(data)))
			lastActivity.Store(now.UnixNano())
		}
	}()

	go func() {
		defer workers.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, lastActivity.Load())) >= a.udpIdleTimeout() {
					_ = writer.frame(protocol.Frame{Type: protocol.FrameUDPClose, Payload: protocol.Result(1, "idle timeout")})
					finish(errors.New("UDP idle timeout"))
					return
				}
			}
		}
	}()

	endErr := <-errCh
	workers.Wait()
	lastTargetMu.Lock()
	target := lastTarget
	lastTargetMu.Unlock()
	recentMu.Lock()
	distinctTargets := len(recent)
	recentMu.Unlock()
	reason := "completed"
	if endErr != nil && !errors.Is(endErr, io.EOF) && !errors.Is(endErr, net.ErrClosed) {
		reason = "udp_failed"
	}
	a.log.Printf("connection closed source=%s target=%s user=%s session=%s udp=true packets_up=%d packets_down=%d bytes_up=%d bytes_down=%d queue_drops=%d rate_drops=%d policy_drops=%d oversize_drops=%d distinct_targets=%d duration=%s egress=%s egress_proxy=%s reason=%s",
		source, target, user.ClientID, session.ID(), stats.packetsUp.Load(), stats.packetsDown.Load(),
		stats.bytesUp.Load(), stats.bytesDown.Load(), stats.queueDrops.Load(), stats.rateDrops.Load(),
		stats.policyDrops.Load(), stats.oversizeDrops.Load(), distinctTargets,
		time.Since(started).Round(time.Millisecond), a.egressType(), a.egressProxy(), reason)
}

func (a *app) udpIdleTimeout() time.Duration {
	duration, _ := config.Duration(a.cfg.UDP.IdleTimeout)
	return duration
}

func (a *app) claimUDP(clientID string) (func(), bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.udpActive[clientID] >= a.cfg.UDP.MaxAssociationsPerUser {
		return nil, false
	}
	a.udpActive[clientID]++
	var once sync.Once
	return func() {
		once.Do(func() {
			a.mu.Lock()
			a.udpActive[clientID]--
			a.mu.Unlock()
		})
	}, true
}

func (a *app) allowUDPRate(clientID string, bytes int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	state := a.udpRates[clientID]
	packetCapacity := float64(a.cfg.UDP.PacketsPerSecondPerUser)
	byteCapacity := float64(a.cfg.UDP.BytesPerSecondPerUser)
	if state == nil {
		state = &udpRateState{updated: now, packetTokens: packetCapacity, byteTokens: byteCapacity}
		a.udpRates[clientID] = state
	} else {
		elapsed := now.Sub(state.updated).Seconds()
		state.packetTokens += elapsed * packetCapacity
		state.byteTokens += elapsed * byteCapacity
		if state.packetTokens > packetCapacity {
			state.packetTokens = packetCapacity
		}
		if state.byteTokens > byteCapacity {
			state.byteTokens = byteCapacity
		}
		state.updated = now
	}
	if state.packetTokens < 1 || state.byteTokens < float64(bytes) {
		return false
	}
	state.packetTokens--
	state.byteTokens -= float64(bytes)
	return true
}

func (a *app) resolveUDP(host string, port uint16, allowPrivate bool) (*net.UDPAddr, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.dialTimeout)
	defer cancel()
	var addresses []net.IP
	if ip := net.ParseIP(host); ip != nil {
		addresses = []net.IP{ip}
	} else {
		key := strings.ToLower(host)
		now := time.Now()
		a.mu.Lock()
		cached, found := a.udpDNS[key]
		a.mu.Unlock()
		if found && now.Before(cached.expires) {
			if cached.negative {
				return nil, "", errors.New("cached UDP DNS failure")
			}
			addresses = append([]net.IP(nil), cached.addresses...)
		} else {
			ips, lookupErr := net.DefaultResolver.LookupIP(ctx, "ip", host)
			entry := udpDNSCacheEntry{}
			if lookupErr != nil {
				entry.negative = true
				negativeTTL, _ := config.Duration(a.cfg.UDP.DNSNegativeTTL)
				entry.expires = now.Add(negativeTTL)
			} else {
				entry.addresses = append([]net.IP(nil), ips...)
				cacheTTL, _ := config.Duration(a.cfg.UDP.DNSCacheTTL)
				entry.expires = now.Add(cacheTTL)
			}
			a.storeUDPDNS(key, entry, now)
			if lookupErr != nil {
				return nil, "", lookupErr
			}
			addresses = ips
		}
	}
	for _, ip := range addresses {
		if udpNeverAllowed(ip) {
			continue
		}
		if !allowPrivate && forbidden(ip) {
			continue
		}
		address := &net.UDPAddr{IP: ip, Port: int(port)}
		return address, net.JoinHostPort(ip.String(), strconv.Itoa(int(port))), nil
	}
	return nil, "", errPrivateTarget
}

func udpNeverAllowed(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.Equal(net.IPv4bcast) || v4.Equal(net.IPv4zero)
	}
	return false
}

func (a *app) storeUDPDNS(key string, entry udpDNSCacheEntry, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.udpDNS == nil {
		a.udpDNS = make(map[string]udpDNSCacheEntry)
	}
	if len(a.udpDNS) >= 4096 {
		for cachedKey, cached := range a.udpDNS {
			if now.After(cached.expires) {
				delete(a.udpDNS, cachedKey)
			}
		}
	}
	if len(a.udpDNS) >= 4096 {
		for cachedKey := range a.udpDNS {
			delete(a.udpDNS, cachedKey)
			break
		}
	}
	a.udpDNS[key] = entry
}

var _ udpEgress = (*socks5.UDPAssociation)(nil)
