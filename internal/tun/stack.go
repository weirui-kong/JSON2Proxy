// Package tun terminates TCP and UDP from a TUN device in a user-space
// network stack (gVisor netstack) and hands each flow to a Handler.
package tun

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const (
	nicID = 1
	// maxPendingTCP bounds handshakes waiting for the handler to decide.
	maxPendingTCP = 1024
	queueSize     = 1024
)

// Handler receives flows. Both methods run on their own goroutine.
type Handler interface {
	// HandleTCP must call Accept or Reject; returning without either
	// rejects the connection.
	HandleTCP(req *TCPRequest)
	HandleUDP(src, dst netip.AddrPort, conn net.Conn)
}

// TCPRequest is a TCP handshake from an app that has not been answered yet.
type TCPRequest struct {
	Src, Dst netip.AddrPort
	r        *tcp.ForwarderRequest
	done     bool
}

// Accept completes the handshake and returns the app's connection.
func (r *TCPRequest) Accept() (net.Conn, error) {
	if r.done {
		return nil, errors.New("TCP request already answered")
	}
	r.done = true
	var wq waiter.Queue
	ep, err := r.r.CreateEndpoint(&wq)
	if err != nil {
		r.r.Complete(true)
		return nil, fmt.Errorf("create TCP endpoint: %s", err)
	}
	r.r.Complete(false)
	ep.SocketOptions().SetKeepAlive(true)
	return gonet.NewTCPConn(&wq, ep), nil
}

// Reject answers the handshake with a reset.
func (r *TCPRequest) Reject() {
	if !r.done {
		r.done = true
		r.r.Complete(true)
	}
}

type Stack struct {
	s      *stack.Stack
	ep     *channel.Endpoint
	dev    Device
	mtu    int
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// New starts a stack that reads packets from dev. It takes ownership of dev.
func New(dev Device, mtu int, h Handler) (*Stack, error) {
	if mtu < 1280 || mtu > 65535 {
		return nil, fmt.Errorf("MTU %d out of range", mtu)
	}
	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})
	if err := configure(s); err != nil {
		s.Close()
		return nil, err
	}
	ep := channel.New(queueSize, uint32(mtu), "")
	if err := s.CreateNIC(nicID, ep); err != nil {
		s.Close()
		return nil, fmt.Errorf("create NIC: %s", err)
	}
	// Accept packets for every destination and answer from any address.
	if err := s.SetPromiscuousMode(nicID, true); err != nil {
		s.Close()
		return nil, fmt.Errorf("promiscuous mode: %s", err)
	}
	if err := s.SetSpoofing(nicID, true); err != nil {
		s.Close()
		return nil, fmt.Errorf("spoofing: %s", err)
	}
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	tcpForwarder := tcp.NewForwarder(s, 0, maxPendingTCP, func(r *tcp.ForwarderRequest) {
		id := r.ID()
		req := &TCPRequest{Src: addrPort(id.RemoteAddress, id.RemotePort), Dst: addrPort(id.LocalAddress, id.LocalPort), r: r}
		h.HandleTCP(req)
		req.Reject()
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
	udpForwarder := udp.NewForwarder(s, func(r *udp.ForwarderRequest) bool {
		id := r.ID()
		var wq waiter.Queue
		ep, err := r.CreateEndpoint(&wq)
		if err != nil {
			return false
		}
		src, dst := addrPort(id.RemoteAddress, id.RemotePort), addrPort(id.LocalAddress, id.LocalPort)
		go h.HandleUDP(src, dst, gonet.NewUDPConn(&wq, ep))
		return true
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)

	ctx, cancel := context.WithCancel(context.Background())
	st := &Stack{s: s, ep: ep, dev: dev, mtu: mtu, cancel: cancel}
	st.wg.Add(2)
	go st.inbound()
	go st.outbound(ctx)
	return st, nil
}

func configure(s *stack.Stack) error {
	sack := tcpip.TCPSACKEnabled(true)
	moderate := tcpip.TCPModerateReceiveBufferOption(true)
	// Keep buffers modest: the iOS packet tunnel has a small memory limit.
	rcv := tcpip.TCPReceiveBufferSizeRangeOption{Min: 4 << 10, Default: 128 << 10, Max: 1 << 20}
	snd := tcpip.TCPSendBufferSizeRangeOption{Min: 4 << 10, Default: 128 << 10, Max: 1 << 20}
	for _, opt := range []tcpip.SettableTransportProtocolOption{&sack, &moderate, &rcv, &snd} {
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, opt); err != nil {
			return fmt.Errorf("TCP option %T: %s", opt, err)
		}
	}
	return nil
}

// inbound injects packets read from the device into the stack.
func (st *Stack) inbound() {
	defer st.wg.Done()
	buf := make([]byte, st.mtu)
	for {
		n, err := st.dev.ReadPacket(buf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}
		var proto tcpip.NetworkProtocolNumber
		switch buf[0] >> 4 {
		case 4:
			proto = ipv4.ProtocolNumber
		case 6:
			proto = ipv6.ProtocolNumber
		default:
			continue
		}
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(append([]byte(nil), buf[:n]...)),
		})
		st.ep.InjectInbound(proto, pkt)
		pkt.DecRef()
	}
}

// outbound writes packets produced by the stack to the device.
func (st *Stack) outbound(ctx context.Context) {
	defer st.wg.Done()
	for {
		pkt := st.ep.ReadContext(ctx)
		if pkt == nil {
			return
		}
		view := pkt.ToView()
		err := st.dev.WritePacket(view.AsSlice())
		view.Release()
		pkt.DecRef()
		if err != nil {
			return
		}
	}
}

// Close stops the stack and closes the device.
func (st *Stack) Close() error {
	var err error
	st.once.Do(func() {
		st.cancel()
		err = st.dev.Close()
		st.ep.Close()
		st.s.Close()
		st.wg.Wait()
	})
	return err
}

func addrPort(a tcpip.Address, port uint16) netip.AddrPort {
	addr, _ := netip.AddrFromSlice(a.AsSlice())
	return netip.AddrPortFrom(addr.Unmap(), port)
}
