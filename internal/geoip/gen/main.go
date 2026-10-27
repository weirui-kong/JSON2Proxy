// Command gen rebuilds the embedded CN address tables from APNIC's delegated
// statistics file.
//
//	go run ./internal/geoip/gen -src delegated-apnic-latest -out internal/geoip
//
// -src may be a local path or an http(s) URL; it defaults to APNIC's file.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const defaultSource = "https://ftp.apnic.net/stats/apnic/delegated-apnic-latest"

type span struct{ lo, hi netip.Addr }

func main() {
	src := flag.String("src", defaultSource, "APNIC delegated statistics file (path or URL)")
	out := flag.String("out", "internal/geoip", "output directory")
	country := flag.String("country", "CN", "country code to extract")
	flag.Parse()

	r, err := open(*src)
	if err != nil {
		fail(err)
	}
	defer r.Close()
	v4, v6, serial, err := parse(r, *country)
	if err != nil {
		fail(err)
	}
	header := fmt.Sprintf("# %s address ranges from APNIC delegated statistics, serial %s.\n"+
		"# Shows where addresses were first allocated, not where they are in use today.\n"+
		"# Regenerate with: go run ./internal/geoip/gen\n", *country, serial)
	name := strings.ToLower(*country)
	for _, f := range []struct {
		file  string
		spans []span
	}{{name + "_ipv4.txt", v4}, {name + "_ipv6.txt", v6}} {
		if err := write(filepath.Join(*out, f.file), header, merge(f.spans)); err != nil {
			fail(err)
		}
	}
}

func open(src string) (io.ReadCloser, error) {
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		return os.Open(src)
	}
	resp, err := http.Get(src)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", src, resp.Status)
	}
	return resp.Body, nil
}

// parse reads records of the form registry|cc|type|start|value|date|status.
func parse(r io.Reader, country string) (v4, v6 []span, serial string, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Split(line, "|")
		if serial == "" && len(f) >= 3 && f[0] == "2" {
			serial = f[2]
			continue
		}
		if len(f) < 7 || f[1] != country || (f[2] != "ipv4" && f[2] != "ipv6") ||
			(f[6] != "allocated" && f[6] != "assigned") {
			continue
		}
		start, err := netip.ParseAddr(f[3])
		if err != nil {
			return nil, nil, "", fmt.Errorf("bad start address in %q", line)
		}
		value, err := strconv.ParseUint(f[4], 10, 64)
		if err != nil || value == 0 {
			return nil, nil, "", fmt.Errorf("bad value in %q", line)
		}
		switch f[2] {
		case "ipv4":
			// The value is an address count, not necessarily a power of two.
			v4 = append(v4, span{start, addOffset(start, new(big.Int).SetUint64(value-1))})
		case "ipv6":
			// The value is a prefix length.
			p := netip.PrefixFrom(start, int(value))
			v6 = append(v6, span{p.Masked().Addr(), lastAddr(p)})
		}
	}
	if serial == "" {
		return nil, nil, "", fmt.Errorf("missing version header")
	}
	return v4, v6, serial, s.Err()
}

func merge(spans []span) []span {
	slices.SortFunc(spans, func(a, b span) int { return a.lo.Compare(b.lo) })
	var out []span
	for _, s := range spans {
		if n := len(out); n > 0 {
			last := &out[n-1]
			if next := last.hi.Next(); next.IsValid() && s.lo.Compare(next) <= 0 {
				if s.hi.Compare(last.hi) > 0 {
					last.hi = s.hi
				}
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

// prefixes splits an inclusive range into the minimal list of CIDR blocks.
func prefixes(s span) []netip.Prefix {
	var out []netip.Prefix
	lo := s.lo
	for {
		bits := lo.BitLen()
		// Grow the block while it stays aligned and inside the range.
		for bits > 0 {
			p := netip.PrefixFrom(lo, bits-1)
			if p.Masked().Addr() != lo || lastAddr(p).Compare(s.hi) > 0 {
				break
			}
			bits--
		}
		p := netip.PrefixFrom(lo, bits)
		out = append(out, p)
		end := lastAddr(p)
		if end.Compare(s.hi) >= 0 {
			return out
		}
		lo = end.Next()
	}
}

func lastAddr(p netip.Prefix) netip.Addr {
	b := p.Masked().Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 1 << (7 - i%8)
	}
	addr, _ := netip.AddrFromSlice(b)
	return addr
}

func addOffset(addr netip.Addr, offset *big.Int) netip.Addr {
	n := new(big.Int).SetBytes(addr.AsSlice())
	n.Add(n, offset)
	b := n.FillBytes(make([]byte, len(addr.AsSlice())))
	out, _ := netip.AddrFromSlice(b)
	return out
}

func write(path, header string, spans []span) error {
	var b strings.Builder
	b.WriteString(header)
	for _, s := range spans {
		for _, p := range prefixes(s) {
			b.WriteString(p.String())
			b.WriteByte('\n')
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gen:", err)
	os.Exit(1)
}
