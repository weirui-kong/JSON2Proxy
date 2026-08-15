package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

const commandUDPAssociate = 3

// EncodeUDPDatagram produces the RFC 1928 UDP request header followed by one
// complete datagram. Fragmentation is intentionally unsupported.
func EncodeUDPDatagram(host string, port uint16, data []byte) ([]byte, error) {
	if port == 0 {
		return nil, errors.New("invalid SOCKS5 UDP port")
	}
	result := []byte{0, 0, 0}
	address, err := encodeAddress(host, port)
	if err != nil {
		return nil, err
	}
	result = append(result, address...)
	result = append(result, data...)
	return result, nil
}

func ParseUDPDatagram(packet []byte) (string, uint16, []byte, error) {
	if len(packet) < 7 || packet[0] != 0 || packet[1] != 0 {
		return "", 0, nil, errors.New("invalid SOCKS5 UDP reserved field")
	}
	if packet[2] != 0 {
		return "", 0, nil, errors.New("SOCKS5 UDP fragmentation is unsupported")
	}
	host, port, consumed, err := parseAddress(packet[3:])
	if err != nil {
		return "", 0, nil, err
	}
	return host, port, packet[3+consumed:], nil
}

func encodeAddress(host string, port uint16) ([]byte, error) {
	if port == 0 {
		return nil, errors.New("invalid port")
	}
	ip := net.ParseIP(host)
	var result []byte
	switch {
	case ip != nil && ip.To4() != nil:
		result = append([]byte{addressIPv4}, ip.To4()...)
	case ip != nil && ip.To16() != nil:
		result = append([]byte{addressIPv6}, ip.To16()...)
	default:
		if len(host) == 0 || len(host) > 253 {
			return nil, errors.New("invalid domain")
		}
		result = append([]byte{addressDomain, byte(len(host))}, []byte(host)...)
	}
	var encodedPort [2]byte
	binary.BigEndian.PutUint16(encodedPort[:], port)
	return append(result, encodedPort[:]...), nil
}

func parseAddress(data []byte) (string, uint16, int, error) {
	if len(data) < 1 {
		return "", 0, 0, errors.New("missing SOCKS5 address")
	}
	var host string
	var offset int
	switch data[0] {
	case addressIPv4:
		if len(data) < 7 {
			return "", 0, 0, errors.New("short SOCKS5 IPv4 address")
		}
		host, offset = net.IP(data[1:5]).String(), 5
	case addressIPv6:
		if len(data) < 19 {
			return "", 0, 0, errors.New("short SOCKS5 IPv6 address")
		}
		host, offset = net.IP(data[1:17]).String(), 17
	case addressDomain:
		if len(data) < 2 || data[1] == 0 || data[1] > 253 || len(data) < int(data[1])+4 {
			return "", 0, 0, errors.New("invalid SOCKS5 domain address")
		}
		host, offset = string(data[2:2+int(data[1])]), 2+int(data[1])
	default:
		return "", 0, 0, errors.New("unsupported SOCKS5 address type")
	}
	port := binary.BigEndian.Uint16(data[offset : offset+2])
	if port == 0 {
		return "", 0, 0, errors.New("invalid SOCKS5 port")
	}
	return host, port, offset + 2, nil
}

type UDPAssociation struct {
	control   net.Conn
	packet    *net.UDPConn
	relay     *net.UDPAddr
	done      chan struct{}
	doneOnce  sync.Once
	closeOnce sync.Once
}

func (c *Client) AssociateUDP(ctx context.Context) (*UDPAssociation, error) {
	control, err := c.connect(ctx, "tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("connect SOCKS5 server: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = control.Close()
		}
	}()
	if deadline, ok := ctx.Deadline(); ok {
		_ = control.SetDeadline(deadline)
	}
	if err = c.negotiate(control); err != nil {
		return nil, err
	}
	request := []byte{version, commandUDPAssociate, 0, addressIPv4, 0, 0, 0, 0, 0, 0}
	if err = writeAll(control, request); err != nil {
		return nil, fmt.Errorf("write SOCKS5 UDP ASSOCIATE: %w", err)
	}
	host, port, code, err := readReply(control)
	if err != nil {
		return nil, err
	}
	if code != connectSucceeded {
		return nil, fmt.Errorf("SOCKS5 UDP ASSOCIATE failed: %s", replyDescription(code))
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		peerHost, _, splitErr := net.SplitHostPort(control.RemoteAddr().String())
		if splitErr != nil {
			return nil, errors.New("invalid SOCKS5 proxy peer address")
		}
		host = peerHost
	}
	relay, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		return nil, fmt.Errorf("resolve SOCKS5 UDP relay: %w", err)
	}
	network := "udp6"
	bind := &net.UDPAddr{IP: net.IPv6unspecified}
	if relay.IP.To4() != nil {
		network, bind = "udp4", &net.UDPAddr{IP: net.IPv4zero}
	}
	packet, err := net.ListenUDP(network, bind)
	if err != nil {
		return nil, fmt.Errorf("open SOCKS5 UDP socket: %w", err)
	}
	if err = control.SetDeadline(time.Time{}); err != nil {
		_ = packet.Close()
		return nil, err
	}
	a := &UDPAssociation{control: control, packet: packet, relay: relay, done: make(chan struct{})}
	go a.watchControl()
	success = true
	return a, nil
}

func readReply(conn net.Conn) (string, uint16, byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return "", 0, 0, fmt.Errorf("read SOCKS5 reply: %w", err)
	}
	if header[0] != version || header[2] != 0 {
		return "", 0, 0, errors.New("invalid SOCKS5 reply")
	}
	encoded := []byte{header[3]}
	switch header[3] {
	case addressIPv4:
		tail := make([]byte, 6)
		if _, err := io.ReadFull(conn, tail); err != nil {
			return "", 0, 0, fmt.Errorf("read SOCKS5 reply address: %w", err)
		}
		encoded = append(encoded, tail...)
	case addressIPv6:
		tail := make([]byte, 18)
		if _, err := io.ReadFull(conn, tail); err != nil {
			return "", 0, 0, fmt.Errorf("read SOCKS5 reply address: %w", err)
		}
		encoded = append(encoded, tail...)
	case addressDomain:
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil || length[0] == 0 {
			return "", 0, 0, errors.New("invalid SOCKS5 reply domain")
		}
		tail := make([]byte, int(length[0])+2)
		if _, err := io.ReadFull(conn, tail); err != nil {
			return "", 0, 0, fmt.Errorf("read SOCKS5 reply address: %w", err)
		}
		encoded = append(encoded, length[0])
		encoded = append(encoded, tail...)
	default:
		return "", 0, 0, errors.New("invalid SOCKS5 reply address type")
	}
	if header[1] != connectSucceeded {
		return "", 0, header[1], nil
	}
	host, port, _, err := parseAddress(encoded)
	return host, port, header[1], err
}

func (a *UDPAssociation) watchControl() {
	var one [1]byte
	_, _ = a.control.Read(one[:])
	a.doneOnce.Do(func() { close(a.done) })
	_ = a.packet.SetReadDeadline(time.Now())
}

func (a *UDPAssociation) Done() <-chan struct{} { return a.done }

func (a *UDPAssociation) WriteDatagram(host string, port uint16, data []byte) error {
	packet, err := EncodeUDPDatagram(host, port, data)
	if err != nil {
		return err
	}
	_, err = a.packet.WriteToUDP(packet, a.relay)
	return err
}

func (a *UDPAssociation) ReadDatagram(deadline time.Time, buffer []byte) (string, uint16, []byte, error) {
	select {
	case <-a.done:
		return "", 0, nil, net.ErrClosed
	default:
	}
	if err := a.packet.SetReadDeadline(deadline); err != nil {
		return "", 0, nil, err
	}
	n, source, err := a.packet.ReadFromUDP(buffer)
	if err != nil {
		select {
		case <-a.done:
			return "", 0, nil, net.ErrClosed
		default:
		}
		return "", 0, nil, err
	}
	if !source.IP.Equal(a.relay.IP) || source.Port != a.relay.Port {
		return "", 0, nil, errors.New("unexpected SOCKS5 UDP relay source")
	}
	return ParseUDPDatagram(buffer[:n])
}

func (a *UDPAssociation) Close() error {
	var err error
	a.closeOnce.Do(func() {
		a.doneOnce.Do(func() { close(a.done) })
		err = a.control.Close()
		if packetErr := a.packet.Close(); err == nil {
			err = packetErr
		}
	})
	return err
}
