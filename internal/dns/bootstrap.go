package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const defaultBootstrapTimeout = 5 * time.Second

// BootstrapError reports that the plaintext bootstrap resolver could not
// resolve an infrastructure host (the proxy server or the DoH server).
type BootstrapError struct {
	Host string
	Err  error
}

func (e *BootstrapError) Error() string {
	return fmt.Sprintf("bootstrap DNS failed for %s: %v", e.Host, e.Err)
}

func (e *BootstrapError) Unwrap() error { return e.Err }

// Bootstrap is a plaintext DNS resolver used only for infrastructure hosts.
type Bootstrap struct {
	server  string
	dial    DialFunc
	timeout time.Duration
	cache   *cache
}

// NewBootstrap creates a resolver for server, an IP address with an optional
// port (default 53). A nil dial uses a plain net.Dialer.
func NewBootstrap(server string, dial DialFunc) (*Bootstrap, error) {
	addr, err := netip.ParseAddrPort(server)
	if err != nil {
		ip, ipErr := netip.ParseAddr(server)
		if ipErr != nil {
			return nil, fmt.Errorf("bootstrap DNS must be an IP address: %q", server)
		}
		addr = netip.AddrPortFrom(ip, 53)
	}
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	return &Bootstrap{server: addr.String(), dial: dial, timeout: defaultBootstrapTimeout, cache: newCache()}, nil
}

// LookupIP resolves host, returning IPv4 addresses before IPv6 ones. Failures
// are returned as *BootstrapError.
func (b *Bootstrap) LookupIP(ctx context.Context, host string) ([]netip.Addr, error) {
	addrs, err := lookupIP(ctx, b.cache, host, b.exchange)
	if err != nil {
		return nil, &BootstrapError{Host: host, Err: err}
	}
	return addrs, nil
}

// ClearCache drops cached answers, for example after a network change.
func (b *Bootstrap) ClearCache() { b.cache.clear() }

func (b *Bootstrap) exchange(ctx context.Context, name string, qtype dnsmessage.Type) (answer, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	id := uint16(rand.Uint32())
	query, err := newQuery(id, name, qtype)
	if err != nil {
		return answer{}, err
	}
	resp, err := b.exchangeUDP(ctx, id, query)
	if err != nil {
		return answer{}, err
	}
	a, err := parseAnswer(resp, id, qtype)
	if err != nil || !a.truncated {
		return a, err
	}
	resp, err = b.exchangeTCP(ctx, query)
	if err != nil {
		return answer{}, err
	}
	return parseAnswer(resp, id, qtype)
}

func (b *Bootstrap) exchangeUDP(ctx context.Context, id uint16, query []byte) ([]byte, error) {
	conn, err := b.dial(ctx, "udp", b.server)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	setDeadline(ctx, conn)
	if _, err := conn.Write(query); err != nil {
		return nil, err
	}
	buf := make([]byte, maxMessageSize)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, err
		}
		// Ignore stray datagrams that do not answer this query.
		if n >= 2 && binary.BigEndian.Uint16(buf) == id {
			return buf[:n], nil
		}
	}
}

func (b *Bootstrap) exchangeTCP(ctx context.Context, query []byte) ([]byte, error) {
	conn, err := b.dial(ctx, "tcp", b.server)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	setDeadline(ctx, conn)
	framed := binary.BigEndian.AppendUint16(make([]byte, 0, len(query)+2), uint16(len(query)))
	if _, err := conn.Write(append(framed, query...)); err != nil {
		return nil, err
	}
	var size [2]byte
	if _, err := io.ReadFull(conn, size[:]); err != nil {
		return nil, err
	}
	resp := make([]byte, binary.BigEndian.Uint16(size[:]))
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errors.New("empty DNS response")
	}
	return resp, nil
}

func setDeadline(ctx context.Context, conn net.Conn) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
}
