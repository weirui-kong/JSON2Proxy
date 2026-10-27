// Package geoip answers which country an address was allocated to. The
// embedded data only covers CN; other countries are not known.
package geoip

import (
	_ "embed"
	"net/netip"
	"slices"
	"strings"
	"sync"
)

//go:generate go run ./gen -out .

//go:embed cn_ipv4.txt
var cnIPv4 string

//go:embed cn_ipv6.txt
var cnIPv6 string

type span struct{ lo, hi netip.Addr }

var cn = sync.OnceValue(func() []span {
	spans := append(parse(cnIPv4), parse(cnIPv6)...)
	slices.SortFunc(spans, func(a, b span) int { return a.lo.Compare(b.lo) })
	return spans
})

// Supported reports whether lookups for country code can ever match.
func Supported(code string) bool { return strings.EqualFold(code, "CN") }

// Country returns "CN" when addr belongs to a CN allocation, otherwise "".
func Country(addr netip.Addr) string {
	if IsCN(addr) {
		return "CN"
	}
	return ""
}

func IsCN(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return false
	}
	spans := cn()
	// First span starting after addr; the candidate is the one before it.
	i, _ := slices.BinarySearchFunc(spans, addr, func(s span, a netip.Addr) int {
		if s.lo.Compare(a) <= 0 {
			return -1
		}
		return 1
	})
	return i > 0 && spans[i-1].hi.Compare(addr) >= 0 && spans[i-1].lo.Is4() == addr.Is4()
}

func parse(data string) []span {
	var spans []span
	for line := range strings.Lines(data) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		p, err := netip.ParsePrefix(line)
		if err != nil {
			panic("geoip: bad embedded prefix " + line)
		}
		spans = append(spans, span{p.Masked().Addr(), lastAddr(p)})
	}
	return spans
}

func lastAddr(p netip.Prefix) netip.Addr {
	b := p.Masked().Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 1 << (7 - i%8)
	}
	addr, _ := netip.AddrFromSlice(b)
	return addr
}
