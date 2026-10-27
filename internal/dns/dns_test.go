package dns

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

var testRecords = map[string][]netip.Addr{
	"example.com": {netip.MustParseAddr("127.0.0.1")},
	"dual.test":   {netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.1")},
	"target.test": {netip.MustParseAddr("198.51.100.7")},
}

// respond builds an authoritative-looking answer for query from records.
func respond(t testing.TB, query []byte, records map[string][]netip.Addr, truncated bool) []byte {
	t.Helper()
	var p dnsmessage.Parser
	h, err := p.Start(query)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	q, err := p.Question()
	if err != nil {
		t.Fatalf("parse question: %v", err)
	}
	addrs, known := records[strings.TrimSuffix(q.Name.String(), ".")]
	header := dnsmessage.Header{ID: h.ID, Response: true, RecursionAvailable: true, Truncated: truncated}
	if !known {
		header.RCode = dnsmessage.RCodeNameError
	}
	b := dnsmessage.NewBuilder(nil, header)
	b.EnableCompression()
	_ = b.StartQuestions()
	_ = b.Question(q)
	_ = b.StartAnswers()
	rh := dnsmessage.ResourceHeader{Name: q.Name, Class: dnsmessage.ClassINET, TTL: 300}
	for _, addr := range addrs {
		if truncated {
			break
		}
		switch {
		case addr.Is4() && q.Type == dnsmessage.TypeA:
			_ = b.AResource(rh, dnsmessage.AResource{A: addr.As4()})
		case addr.Is6() && q.Type == dnsmessage.TypeAAAA:
			_ = b.AAAAResource(rh, dnsmessage.AAAAResource{AAAA: addr.As16()})
		}
	}
	msg, err := b.Finish()
	if err != nil {
		t.Fatalf("build response: %v", err)
	}
	return msg
}

// startDNS serves UDP (optionally truncated) and TCP on the same local port.
func startDNS(t *testing.T, truncateUDP bool) (string, *atomic.Int32) {
	t.Helper()
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tcp.Close(); _ = udp.Close() })
	var queries atomic.Int32
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			queries.Add(1)
			_, _ = udp.WriteTo(respond(t, buf[:n], testRecords, truncateUDP), from)
		}
	}()
	go func() {
		for {
			conn, err := tcp.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				var size [2]byte
				if _, err := io.ReadFull(conn, size[:]); err != nil {
					return
				}
				query := make([]byte, binary.BigEndian.Uint16(size[:]))
				if _, err := io.ReadFull(conn, query); err != nil {
					return
				}
				queries.Add(1)
				resp := respond(t, query, testRecords, false)
				_, _ = conn.Write(append(binary.BigEndian.AppendUint16(nil, uint16(len(resp))), resp...))
			}()
		}
	}()
	return tcp.Addr().String(), &queries
}

func TestBootstrapOrdersIPv4FirstAndCaches(t *testing.T) {
	server, queries := startDNS(t, false)
	b, err := NewBootstrap(server, nil)
	if err != nil {
		t.Fatal(err)
	}
	addrs, err := b.LookupIP(context.Background(), "Dual.TEST.")
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("2001:db8::1")}
	if len(addrs) != 2 || addrs[0] != want[0] || addrs[1] != want[1] {
		t.Fatalf("addrs = %v, want %v", addrs, want)
	}
	before := queries.Load()
	if _, err := b.LookupIP(context.Background(), "dual.test"); err != nil {
		t.Fatal(err)
	}
	if queries.Load() != before {
		t.Fatalf("cached lookup sent %d new queries", queries.Load()-before)
	}
	b.ClearCache()
	if _, err := b.LookupIP(context.Background(), "dual.test"); err != nil {
		t.Fatal(err)
	}
	if queries.Load() == before {
		t.Fatal("lookup after ClearCache did not query the server")
	}
}

func TestBootstrapFallsBackToTCPWhenTruncated(t *testing.T) {
	server, _ := startDNS(t, true)
	b, err := NewBootstrap(server, nil)
	if err != nil {
		t.Fatal(err)
	}
	addrs, err := b.LookupIP(context.Background(), "target.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != netip.MustParseAddr("198.51.100.7") {
		t.Fatalf("addrs = %v", addrs)
	}
}

func TestBootstrapNXDomain(t *testing.T) {
	server, _ := startDNS(t, false)
	b, _ := NewBootstrap(server, nil)
	_, err := b.LookupIP(context.Background(), "missing.test")
	var be *BootstrapError
	var na *NoAddressError
	if !errors.As(err, &be) || !errors.As(err, &na) || na.RCode != dnsmessage.RCodeNameError {
		t.Fatalf("err = %v, want BootstrapError wrapping NXDOMAIN", err)
	}
}

func TestNewBootstrapRequiresIP(t *testing.T) {
	if _, err := NewBootstrap("dns.google", nil); err == nil {
		t.Fatal("expected an error for a host name")
	}
	b, err := NewBootstrap("8.8.8.8", nil)
	if err != nil || b.server != "8.8.8.8:53" {
		t.Fatalf("server = %v, err = %v", b, err)
	}
}

type dohServer struct {
	*httptest.Server
	mu       sync.Mutex
	failNext int
	requests []*http.Request
}

func startDoH(t *testing.T) *dohServer {
	t.Helper()
	s := &dohServer{}
	s.Server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r)
		fail := s.failNext > 0
		if fail {
			s.failNext--
		}
		s.mu.Unlock()
		if fail {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		query, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", dohContentType)
		_, _ = w.Write(respond(t, query, testRecords, false))
	}))
	s.EnableHTTP2 = true
	s.StartTLS()
	t.Cleanup(s.Close)
	return s
}

func (s *dohServer) tlsConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(s.Certificate())
	return &tls.Config{RootCAs: pool}
}

func TestDoHResolvesHostThroughBootstrap(t *testing.T) {
	dnsServer, _ := startDNS(t, false)
	b, _ := NewBootstrap(dnsServer, nil)
	srv := startDoH(t)
	u, _ := url.Parse(srv.URL)
	// httptest certificates are valid for example.com, which the bootstrap
	// server maps to 127.0.0.1.
	d, err := NewDoH(DoHConfig{
		URL:       "https://example.com:" + u.Port() + "/dns-query",
		Bootstrap: b,
		TLSConfig: srv.tlsConfig(),
	})
	if err != nil {
		t.Fatal(err)
	}
	addrs, err := d.LookupIP(context.Background(), "target.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != netip.MustParseAddr("198.51.100.7") {
		t.Fatalf("addrs = %v", addrs)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	for _, r := range srv.requests {
		if r.ProtoMajor != 2 || r.Method != http.MethodPost || r.Header.Get("Content-Type") != dohContentType {
			t.Fatalf("request %s %s content-type %q", r.Proto, r.Method, r.Header.Get("Content-Type"))
		}
	}
}

func TestDoHReportsBootstrapFailure(t *testing.T) {
	dnsServer, _ := startDNS(t, false)
	b, _ := NewBootstrap(dnsServer, nil)
	var got []error
	d, _ := NewDoH(DoHConfig{
		URL:            "https://missing.test/dns-query",
		Bootstrap:      b,
		OnHealthChange: func(_ bool, err error) { got = append(got, err) },
	})
	_, err := d.LookupIP(context.Background(), "target.test")
	var be *BootstrapError
	if !errors.As(err, &be) || be.Host != "missing.test" {
		t.Fatalf("err = %v, want BootstrapError for missing.test", err)
	}
	if len(got) != 1 || !errors.As(got[0], &be) {
		t.Fatalf("health notifications = %v", got)
	}
}

func TestDoHHealthNotifiesOnlyOnTransitions(t *testing.T) {
	srv := startDoH(t)
	var mu sync.Mutex
	var states []bool
	d, err := NewDoH(DoHConfig{
		URL:       srv.URL + "/dns-query",
		TLSConfig: srv.tlsConfig(),
		OnHealthChange: func(available bool, _ error) {
			mu.Lock()
			states = append(states, available)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	srv.mu.Lock()
	srv.failNext = 1 << 20
	srv.mu.Unlock()
	for range 3 {
		d.ClearCache()
		if _, err := d.LookupIP(ctx, "target.test"); err == nil {
			t.Fatal("expected failure")
		}
	}
	if d.Available() {
		t.Fatal("DoH should be unavailable")
	}
	srv.mu.Lock()
	srv.failNext = 0
	srv.mu.Unlock()
	d.ClearCache()
	if _, err := d.LookupIP(ctx, "target.test"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(states) != 2 || states[0] || !states[1] {
		t.Fatalf("health transitions = %v, want [false true]", states)
	}
}

func TestDoHExchangePreservesQueryID(t *testing.T) {
	srv := startDoH(t)
	d, _ := NewDoH(DoHConfig{URL: srv.URL + "/dns-query", TLSConfig: srv.tlsConfig()})
	query, _ := newQuery(0xbeef, "target.test", dnsmessage.TypeA)
	resp, err := d.Exchange(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	a, err := parseAnswer(resp, 0xbeef, dnsmessage.TypeA)
	if err != nil || len(a.addrs) != 1 {
		t.Fatalf("answer = %+v, err = %v", a, err)
	}
}

func TestNewDoHValidatesURL(t *testing.T) {
	for _, raw := range []string{"http://dns.google/dns-query", "dns.google", "https://dns.google/dns-query"} {
		_, err := NewDoH(DoHConfig{URL: raw})
		if err == nil {
			t.Fatalf("NewDoH(%q) without bootstrap succeeded", raw)
		}
	}
	if _, err := NewDoH(DoHConfig{URL: "https://8.8.8.8/dns-query"}); err != nil {
		t.Fatalf("IP DoH URL: %v", err)
	}
}

func TestTTLIsClampedAndCacheExpires(t *testing.T) {
	if got := clampTTL(5 * time.Second); got != minTTL {
		t.Fatalf("clampTTL(5s) = %v", got)
	}
	if got := clampTTL(2 * time.Hour); got != maxTTL {
		t.Fatalf("clampTTL(2h) = %v", got)
	}
	c := newCache()
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }
	key := cacheKey{name: "a.test", qtype: dnsmessage.TypeA}
	c.put(key, answer{ttl: minTTL})
	if _, ok := c.get(key); !ok {
		t.Fatal("fresh entry missing")
	}
	now = now.Add(minTTL)
	if _, ok := c.get(key); ok {
		t.Fatal("expired entry returned")
	}
}
