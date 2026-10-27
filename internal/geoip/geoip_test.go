package geoip

import (
	"net/netip"
	"testing"
)

func TestIsCN(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want bool
	}{
		{"223.5.5.5", true},       // AliDNS
		{"114.114.114.114", true}, // 114DNS
		{"1.0.1.1", true},
		{"::ffff:223.5.5.5", true},
		{"2400:3200::1", true}, // AliDNS IPv6
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"192.168.1.1", false},
		{"2001:4860:4860::8888", false},
		{"::", false},
		{"0.0.0.0", false},
		{"255.255.255.255", false},
	} {
		if got := IsCN(netip.MustParseAddr(tc.addr)); got != tc.want {
			t.Errorf("IsCN(%s) = %v, want %v", tc.addr, got, tc.want)
		}
	}
	if IsCN(netip.Addr{}) {
		t.Error("IsCN(invalid) = true")
	}
}

func TestEmbeddedSpansAreSortedAndDisjoint(t *testing.T) {
	spans := cn()
	if len(spans) < 1000 {
		t.Fatalf("only %d spans embedded", len(spans))
	}
	for i := 1; i < len(spans); i++ {
		if spans[i-1].hi.Compare(spans[i].lo) >= 0 {
			t.Fatalf("spans overlap: %v-%v and %v-%v", spans[i-1].lo, spans[i-1].hi, spans[i].lo, spans[i].hi)
		}
	}
}

func TestCountryAndSupported(t *testing.T) {
	if Country(netip.MustParseAddr("223.5.5.5")) != "CN" || Country(netip.MustParseAddr("8.8.8.8")) != "" {
		t.Fatal("Country returned unexpected values")
	}
	if !Supported("cn") || Supported("US") {
		t.Fatal("Supported returned unexpected values")
	}
}
