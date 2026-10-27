package mobile

import (
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

type fakePlatform struct {
	mu        sync.Mutex
	protected int
	refuse    bool
	logs      []string
}

func (f *fakePlatform) Protect(int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.protected++
	return !f.refuse
}

func (f *fakePlatform) Log(_ int, tag, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, tag+": "+message)
}

func (f *fakePlatform) Notice(string, string) {}

func usePlatform(t *testing.T, p Platform) {
	SetPlatform(p)
	t.Cleanup(func() { SetPlatform(nil) })
}

func listen(t *testing.T) (*net.TCPListener, int) {
	t.Helper()
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	return l, l.Addr().(*net.TCPAddr).Port
}

// startDNS answers every A query with 127.0.0.1 and AAAA with nothing.
func startDNS(t *testing.T) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			var m dnsmessage.Message
			if m.Unpack(buf[:n]) != nil {
				continue
			}
			m.Response = true
			q := m.Questions[0]
			if q.Type == dnsmessage.TypeA {
				m.Answers = []dnsmessage.Resource{{
					Header: dnsmessage.ResourceHeader{Name: q.Name, Type: q.Type, Class: q.Class, TTL: 60},
					Body:   &dnsmessage.AResource{A: [4]byte{127, 0, 0, 1}},
				}}
			}
			resp, _ := m.Pack()
			_, _ = conn.WriteTo(resp, from)
		}
	}()
	return conn.LocalAddr().String()
}

func TestTCPPingThroughBootstrap(t *testing.T) {
	p := &fakePlatform{}
	usePlatform(t, p)
	_, port := listen(t)
	r := TCPPing(startDNS(t), "server.test", port, 1000, 3)
	if r.Error != "" || r.Successes != 3 {
		t.Fatalf("result = %+v", r)
	}
	if r.Address != "127.0.0.1:"+strconv.Itoa(port) {
		t.Fatalf("address = %q", r.Address)
	}
	// Bootstrap A and AAAA sockets plus three TCP connections.
	if p.protected < 5 {
		t.Fatalf("protected %d sockets, want at least 5", p.protected)
	}
}

func TestTCPPingFailures(t *testing.T) {
	usePlatform(t, &fakePlatform{})
	l, port := listen(t)
	_ = l.Close()
	if r := TCPPing("", "127.0.0.1", port, 500, 2); r.Successes != 0 || r.Error == "" {
		t.Fatalf("closed port: %+v", r)
	}
	if r := TCPPing("", "127.0.0.1", 0, 500, 1); !strings.Contains(r.Error, "invalid port") {
		t.Fatalf("port 0: %+v", r)
	}
	if r := TCPPing("not-an-ip", "server.test", 443, 500, 1); r.Error == "" {
		t.Fatalf("bad bootstrap: %+v", r)
	}
}

func TestTCPPingFailsWhenProtectRefused(t *testing.T) {
	usePlatform(t, &fakePlatform{refuse: true})
	_, port := listen(t)
	if r := TCPPing("", "127.0.0.1", port, 500, 1); r.Successes != 0 || !strings.Contains(r.Error, "protect") {
		t.Fatalf("result = %+v", r)
	}
}

func TestParseEngineConfig(t *testing.T) {
	cfg, host, err := parseEngineConfig(`{"client":{"server":"example.net:8443"},"mode":"geoip-cn","memory_limit_mb":40,"totals_file":"/data/totals.json"}`)
	if err != nil {
		t.Fatal(err)
	}
	if host.memoryLimit != 40<<20 || host.totalsFile != "/data/totals.json" {
		t.Fatalf("host options = %+v", host)
	}
	if cfg.DoHURL != aliDoH || cfg.Bootstrap != DefaultBootstrap || cfg.Client.Server != "example.net:8443" {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
	if last := cfg.Rules[len(cfg.Rules)-1].String(); last != "FINAL,PROXY" || cfg.Rules[len(cfg.Rules)-2].String() != "GEOIP,CN,DIRECT" {
		t.Fatalf("geoip-cn rules = %v", cfg.Rules)
	}
	cfg, host, err = parseEngineConfig(`{"mode":"custom","rules":"DOMAIN,a.test,DIRECT\nFINAL,REJECT","doh":"https://1.1.1.1/dns-query"}`)
	if err != nil || len(cfg.Rules) != 2 || cfg.DoHURL != "https://1.1.1.1/dns-query" || host != (hostOptions{}) {
		t.Fatalf("custom: %+v, err %v", cfg, err)
	}
	for _, bad := range []string{`{`, `{"mode":"other"}`, `{"mode":"custom","rules":"DOMAIN,a.test,DIRECT"}`, `{"memory_limit_mb":-1}`} {
		if _, _, err := parseEngineConfig(bad); err == nil {
			t.Errorf("parseEngineConfig(%s) succeeded", bad)
		}
	}
}

func TestDefaultsAndValidation(t *testing.T) {
	if DefaultDoH(ModeGlobal) != googleDoH || DefaultDoH(ModeGeoIPCN) != aliDoH {
		t.Fatal("unexpected default DoH")
	}
	if err := ValidateRules("DOMAIN-SUFFIX,cn,DIRECT\nGEOIP,CN,DIRECT\nFINAL,PROXY"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRules("DOMAIN,a.test,PROXY"); err == nil {
		t.Fatal("rules without FINAL accepted")
	}
}

func TestStatsWithoutEngine(t *testing.T) {
	if got := Stats(true); got != "" {
		t.Fatalf("Stats without an engine = %q, want empty", got)
	}
}

func TestEngineConfigRefusesTunnelNetworks(t *testing.T) {
	cfg, _, err := parseEngineConfig(`{"client":{}}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range []string{TunnelDNSAddress, TunnelIPv4Address, TunnelIPv6Address} {
		a := netip.MustParseAddr(addr)
		if !slices.ContainsFunc(cfg.TunnelPrefixes, func(p netip.Prefix) bool { return p.Contains(a) }) {
			t.Errorf("%s is not in the tunnel prefixes %v", addr, cfg.TunnelPrefixes)
		}
	}
}
