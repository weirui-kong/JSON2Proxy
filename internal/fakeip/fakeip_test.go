package fakeip

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

func TestPoolStableAndReverse(t *testing.T) {
	p, err := NewPool(DefaultPrefix, DefaultCapacity)
	if err != nil {
		t.Fatal(err)
	}
	a := p.Lookup("Example.COM.")
	if a != netip.MustParseAddr("198.18.0.1") {
		t.Fatalf("first address = %v", a)
	}
	if b := p.Lookup("example.com"); b != a {
		t.Fatalf("same name got %v and %v", a, b)
	}
	if c := p.Lookup("other.test"); c == a || !p.Contains(c) {
		t.Fatalf("second name got %v", c)
	}
	if name, ok := p.Name(netip.MustParseAddr("::ffff:198.18.0.1")); !ok || name != "example.com" {
		t.Fatalf("Name = %q, %v", name, ok)
	}
	if _, ok := p.Name(netip.MustParseAddr("198.18.9.9")); ok {
		t.Fatal("unallocated address resolved to a name")
	}
	if p.Contains(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("Contains(8.8.8.8) = true")
	}
	p.Reset()
	if _, ok := p.Name(a); ok {
		t.Fatal("mapping survived Reset")
	}
}

func TestPoolEvictsLeastRecentlyUsed(t *testing.T) {
	p, err := NewPool(netip.MustParsePrefix("10.0.0.0/30"), 0)
	if err != nil {
		t.Fatal(err)
	}
	// A /30 has two usable addresses.
	a := p.Lookup("a.test")
	b := p.Lookup("b.test")
	p.Name(a) // a is now more recent than b
	c := p.Lookup("c.test")
	if c != b {
		t.Fatalf("c.test got %v, want b.test's evicted %v", c, b)
	}
	if _, ok := p.Name(b); !ok {
		t.Fatal("reused address has no name")
	}
	if name, _ := p.Name(b); name != "c.test" {
		t.Fatalf("Name(%v) = %q", b, name)
	}
	if got := p.Lookup("a.test"); got != a {
		t.Fatalf("a.test moved from %v to %v", a, got)
	}
	if _, err := NewPool(netip.MustParsePrefix("2001:db8::/64"), 10); err == nil {
		t.Fatal("IPv6 prefix accepted")
	}
}

func query(t *testing.T, name string, qtype dnsmessage.Type) []byte {
	t.Helper()
	msg := dnsmessage.Message{
		Header:    dnsmessage.Header{ID: 0x1234, RecursionDesired: true},
		Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName(name), Type: qtype, Class: dnsmessage.ClassINET}},
	}
	b, err := msg.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parse(t *testing.T, b []byte) dnsmessage.Message {
	t.Helper()
	var m dnsmessage.Message
	if err := m.Unpack(b); err != nil {
		t.Fatalf("unpack response: %v", err)
	}
	if m.ID != 0x1234 || !m.Response {
		t.Fatalf("header = %+v", m.Header)
	}
	return m
}

func TestHandleQuery(t *testing.T) {
	p, _ := NewPool(DefaultPrefix, DefaultCapacity)
	ctx := context.Background()
	var forwarded int
	forward := func(_ context.Context, q []byte) ([]byte, error) {
		forwarded++
		var m dnsmessage.Message
		_ = m.Unpack(q)
		m.Response = true
		return m.Pack()
	}

	m := parse(t, p.HandleQuery(ctx, query(t, "www.example.com.", dnsmessage.TypeA), forward))
	if len(m.Answers) != 1 || m.Answers[0].Header.TTL != fakeTTL {
		t.Fatalf("A answers = %+v", m.Answers)
	}
	ip := netip.AddrFrom4(m.Answers[0].Body.(*dnsmessage.AResource).A)
	if name, ok := p.Name(ip); !ok || name != "www.example.com" {
		t.Fatalf("answer %v maps to %q", ip, name)
	}

	m = parse(t, p.HandleQuery(ctx, query(t, "www.example.com.", dnsmessage.TypeAAAA), forward))
	if m.RCode != dnsmessage.RCodeSuccess || len(m.Answers) != 0 || len(m.Questions) != 1 {
		t.Fatalf("AAAA response = %+v", m)
	}

	parse(t, p.HandleQuery(ctx, query(t, "example.com.", dnsmessage.TypeTXT), forward))
	if forwarded != 1 {
		t.Fatalf("TXT forwarded %d times", forwarded)
	}

	failing := func(context.Context, []byte) ([]byte, error) { return nil, errors.New("down") }
	m = parse(t, p.HandleQuery(ctx, query(t, "example.com.", dnsmessage.TypeMX), failing))
	if m.RCode != dnsmessage.RCodeServerFailure {
		t.Fatalf("rcode = %v, want SERVFAIL", m.RCode)
	}

	if p.HandleQuery(ctx, []byte{1, 2, 3}, forward) != nil {
		t.Fatal("garbage query got a response")
	}
	noQuestion, _ := (&dnsmessage.Message{Header: dnsmessage.Header{ID: 0x1234}}).Pack()
	if m := parse(t, p.HandleQuery(ctx, noQuestion, forward)); m.RCode != dnsmessage.RCodeFormatError {
		t.Fatalf("rcode = %v, want FORMERR", m.RCode)
	}
}
