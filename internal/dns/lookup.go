package dns

import (
	"context"
	"net"
	"net/netip"
	"sync"

	"golang.org/x/net/dns/dnsmessage"
)

// DialFunc opens a connection. Platform code injects one that keeps sockets
// outside the VPN tunnel (Android VpnService.protect).
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type exchangeFunc func(ctx context.Context, name string, qtype dnsmessage.Type) (answer, error)

// lookupIP resolves A and AAAA concurrently and returns IPv4 addresses first.
func lookupIP(ctx context.Context, c *cache, host string, exchange exchangeFunc) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr.Unmap()}, nil
	}
	name := Normalize(host)
	qtypes := [2]dnsmessage.Type{dnsmessage.TypeA, dnsmessage.TypeAAAA}
	var results [2]cacheEntry
	var errs [2]error
	var wg sync.WaitGroup
	for i, qtype := range qtypes {
		key := cacheKey{name: name, qtype: qtype}
		if e, ok := c.get(key); ok {
			results[i] = e
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := exchange(ctx, name, qtype)
			if err != nil {
				errs[i] = err
				return
			}
			c.put(key, a)
			results[i] = cacheEntry{addrs: a.addrs, rcode: a.rcode}
		}()
	}
	wg.Wait()

	addrs := append(append([]netip.Addr(nil), results[0].addrs...), results[1].addrs...)
	if len(addrs) > 0 {
		return addrs, nil
	}
	if errs[0] != nil {
		return nil, errs[0]
	}
	if errs[1] != nil {
		return nil, errs[1]
	}
	return nil, &NoAddressError{Name: name, RCode: results[0].rcode}
}
