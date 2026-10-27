package main

import (
	"math/big"
	"net/netip"
	"strings"
	"testing"
)

const sample = `# comment
2|apnic|20261027|5|19830613|20261026|+1000
apnic|CN|asn|3460|1|20020801|allocated
apnic|CN|ipv4|1.0.1.0|256|20110414|allocated
apnic|CN|ipv4|1.0.2.0|512|20110414|allocated
apnic|CN|ipv4|1.0.8.0|768|20110414|assigned
apnic|AU|ipv4|1.1.1.0|256|20110811|assigned
apnic|CN|ipv6|2001:250::|31|20000426|allocated
apnic|CN|ipv6|2001:252::|32|20000426|allocated
apnic|CN|ipv4|1.0.20.0|256|20110414|reserved
`

func TestParseMergeAndSplit(t *testing.T) {
	v4, v6, serial, err := parse(strings.NewReader(sample), "CN")
	if err != nil || serial != "20261027" {
		t.Fatalf("serial = %q, err = %v", serial, err)
	}
	var got []string
	for _, s := range merge(v4) {
		for _, p := range prefixes(s) {
			got = append(got, p.String())
		}
	}
	// 1.0.1.0/24 and 1.0.2.0/23 merge into 1.0.1.0-1.0.3.255; 768 addresses
	// starting at 1.0.8.0 split into a /23 and a /24.
	want := []string{"1.0.1.0/24", "1.0.2.0/23", "1.0.8.0/23", "1.0.10.0/24"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("v4 = %v, want %v", got, want)
	}
	got = nil
	for _, s := range merge(v6) {
		for _, p := range prefixes(s) {
			got = append(got, p.String())
		}
	}
	// Adjacent but not a single aligned block, so two prefixes remain.
	if strings.Join(got, " ") != "2001:250::/31 2001:252::/32" {
		t.Fatalf("v6 = %v", got)
	}
}

func TestPrefixesCoverRangeExactly(t *testing.T) {
	s := span{netip.MustParseAddr("10.0.0.3"), netip.MustParseAddr("10.0.1.17")}
	var total big.Int
	next := s.lo
	for _, p := range prefixes(s) {
		if p.Addr() != next {
			t.Fatalf("gap before %v", p)
		}
		size := new(big.Int).Lsh(big.NewInt(1), uint(32-p.Bits()))
		total.Add(&total, size)
		next = lastAddr(p).Next()
	}
	if total.Int64() != 271 || next != s.hi.Next() {
		t.Fatalf("covered %v addresses ending before %v", total.String(), next)
	}
}
