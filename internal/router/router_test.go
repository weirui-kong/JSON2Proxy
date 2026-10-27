package router

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
)

type fakeResolver struct {
	answers map[string][]netip.Addr
	calls   int
}

func (f *fakeResolver) LookupIP(_ context.Context, host string) ([]netip.Addr, error) {
	f.calls++
	if addrs, ok := f.answers[host]; ok {
		return addrs, nil
	}
	return nil, errors.New("lookup failed")
}

func addrs(s ...string) []netip.Addr {
	var out []netip.Addr
	for _, a := range s {
		out = append(out, netip.MustParseAddr(a))
	}
	return out
}

func mustRouter(t *testing.T, rules string, res Resolver) *Router {
	t.Helper()
	parsed, err := ParseRules(rules)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(parsed, res)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDomainRules(t *testing.T) {
	r := mustRouter(t, `
		DOMAIN,exact.test,DIRECT
		DOMAIN-SUFFIX,example.com,REJECT
		DOMAIN-KEYWORD,ads,DIRECT
		FINAL,PROXY
	`, nil)
	for host, want := range map[string]Action{
		"exact.test":      Direct,
		"EXACT.TEST.":     Direct,
		"sub.exact.test":  Proxy,
		"example.com":     Reject,
		"a.b.example.com": Reject,
		"badexample.com":  Proxy, // suffix must match on a label boundary
		"myads.net":       Direct,
		"other.test":      Proxy,
	} {
		if got := r.Route(context.Background(), Target{Host: host}).Action; got != want {
			t.Errorf("Route(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestIPRulesResolveLazilyOnce(t *testing.T) {
	res := &fakeResolver{answers: map[string][]netip.Addr{
		"lan.test":  addrs("192.168.1.5"),
		"dual.test": addrs("2001:db8::1", "10.1.2.3"),
	}}
	r := mustRouter(t, `
		DOMAIN,skip.test,PROXY
		IP-CIDR,192.168.0.0/16,DIRECT
		IP-CIDR,10.0.0.0/8,REJECT
		IP-CIDR6,2001:db8::/32,DIRECT
		FINAL,PROXY
	`, res)

	d := r.Route(context.Background(), Target{Host: "lan.test"})
	if d.Action != Direct || len(d.Addrs) != 1 || res.calls != 1 {
		t.Fatalf("lan.test: %+v, calls %d", d, res.calls)
	}
	// IPv4 is preferred for matching even when IPv6 comes first.
	if d := r.Route(context.Background(), Target{Host: "dual.test"}); d.Action != Reject || res.calls != 2 {
		t.Fatalf("dual.test: %+v, calls %d", d, res.calls)
	}
	// A domain rule earlier in the list must not trigger a lookup.
	if d := r.Route(context.Background(), Target{Host: "skip.test"}); d.Action != Proxy || res.calls != 2 {
		t.Fatalf("skip.test: %+v, calls %d", d, res.calls)
	}
	// Known IPs never resolve.
	if d := r.Route(context.Background(), Target{Addr: netip.MustParseAddr("2001:db8::9")}); d.Action != Direct || res.calls != 2 {
		t.Fatalf("ip target: %+v, calls %d", d, res.calls)
	}
}

func TestResolveFailureSkipsIPRules(t *testing.T) {
	res := &fakeResolver{}
	r := mustRouter(t, "IP-CIDR,0.0.0.0/0,DIRECT\nFINAL,PROXY", res)
	d := r.Route(context.Background(), Target{Host: "broken.test"})
	if d.Action != Proxy || d.ResolveErr == nil {
		t.Fatalf("decision = %+v, want PROXY with ResolveErr", d)
	}
}

func TestNoResolveSkipsDomainTargets(t *testing.T) {
	res := &fakeResolver{answers: map[string][]netip.Addr{"lan.test": addrs("192.168.1.5")}}
	r := mustRouter(t, "IP-CIDR,192.168.0.0/16,DIRECT,no-resolve\nFINAL,PROXY", res)
	if d := r.Route(context.Background(), Target{Host: "lan.test"}); d.Action != Proxy || res.calls != 0 {
		t.Fatalf("decision = %+v, calls %d", d, res.calls)
	}
	if d := r.Route(context.Background(), Target{Host: "192.168.3.3"}); d.Action != Direct {
		t.Fatalf("IP literal host: %+v", d)
	}
}

func TestGlobalModeNeverResolves(t *testing.T) {
	rules, err := ModeRules(ModeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	res := &fakeResolver{answers: map[string][]netip.Addr{"cn.test": addrs("223.5.5.5")}}
	r, _ := New(rules, res)
	for _, tc := range []struct {
		target Target
		want   Action
	}{
		{Target{Host: "cn.test"}, Proxy},
		{Target{Host: "www.google.com"}, Proxy},
		{Target{Addr: netip.MustParseAddr("223.5.5.5")}, Proxy},
		{Target{Addr: netip.MustParseAddr("192.168.1.1")}, Direct},
		{Target{Addr: netip.MustParseAddr("::1")}, Direct},
	} {
		if got := r.Route(context.Background(), tc.target).Action; got != tc.want {
			t.Errorf("Route(%+v) = %v, want %v", tc.target, got, tc.want)
		}
	}
	if res.calls != 0 {
		t.Fatalf("global mode resolved %d times", res.calls)
	}
}

func TestGeoIPCNMode(t *testing.T) {
	rules, _ := ModeRules(ModeGeoIPCN)
	res := &fakeResolver{answers: map[string][]netip.Addr{
		"cn.test":     addrs("223.5.5.5"),
		"global.test": addrs("8.8.8.8"),
		"v6cn.test":   addrs("2400:3200::1"),
	}}
	r, _ := New(rules, res)
	for _, tc := range []struct {
		target Target
		want   Action
	}{
		{Target{Host: "cn.test"}, Direct},
		{Target{Host: "global.test"}, Proxy},
		{Target{Host: "v6cn.test"}, Direct},
		{Target{Host: "unresolvable.test"}, Proxy},
		{Target{Addr: netip.MustParseAddr("114.114.114.114")}, Direct},
		{Target{Addr: netip.MustParseAddr("1.1.1.1")}, Proxy},
		{Target{Addr: netip.MustParseAddr("10.0.0.1")}, Direct},
	} {
		d := r.Route(context.Background(), tc.target)
		if d.Action != tc.want {
			t.Errorf("Route(%+v) = %v (rule %v), want %v", tc.target, d.Action, d.Rule, tc.want)
		}
	}
	d := r.Route(context.Background(), Target{Host: "cn.test"})
	if len(d.Addrs) != 1 || d.Addrs[0] != netip.MustParseAddr("223.5.5.5") {
		t.Fatalf("DIRECT decision should carry resolved addresses, got %v", d.Addrs)
	}
	if _, err := ModeRules("bogus"); err == nil {
		t.Fatal("unknown mode accepted")
	}
}

func TestParseRules(t *testing.T) {
	r, err := ParseRule(" ip-cidr , 10.1.2.3/8 , direct , No-Resolve ")
	if err != nil {
		t.Fatal(err)
	}
	if r.String() != "IP-CIDR,10.0.0.0/8,DIRECT,no-resolve" {
		t.Fatalf("String() = %q", r.String())
	}
	for _, bad := range []string{
		"DOMAIN,example.com",
		"DOMAIN,,PROXY",
		"DOMAIN,example.com,BLOCK",
		"IP-CIDR,2001:db8::/32,DIRECT",
		"IP-CIDR6,10.0.0.0/8,DIRECT",
		"IP-CIDR,10.0.0.0,DIRECT",
		"GEOIP,US,DIRECT",
		"DOMAIN,example.com,PROXY,no-resolve",
		"PROCESS,foo,DIRECT",
		"FINAL,PROXY,extra",
	} {
		if _, err := ParseRule(bad); err == nil {
			t.Errorf("ParseRule(%q) succeeded", bad)
		}
	}
	for _, bad := range []string{"", "DOMAIN,a.test,PROXY", "FINAL,PROXY\nDOMAIN,a.test,PROXY"} {
		if _, err := ParseRules(bad); err == nil {
			t.Errorf("ParseRules(%q) succeeded", bad)
		}
	}
	_, err = ParseRules("# comment\n\nDOMAIN,a.test,PROXY\nBAD\nFINAL,DIRECT")
	if err == nil || !strings.Contains(err.Error(), "line 4") {
		t.Fatalf("error should name the line: %v", err)
	}
}
