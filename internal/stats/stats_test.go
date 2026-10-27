package stats

import (
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestCollector(maxHosts, maxDNS, maxFlows int) (*Collector, *clock) {
	clk := &clock{t: time.UnixMilli(1_000_000)}
	return NewWithLimits(maxHosts, maxDNS, maxFlows, clk.now), clk
}

func TestCountsActiveFlowsAndBytes(t *testing.T) {
	c, _ := newTestCollector(10, 10, 10)
	tcp := c.Route(NetworkTCP, "example.com", 443, "PROXY", "FINAL,PROXY")
	tcp.Start()
	udp := c.Route(NetworkUDP, "198.51.100.7", 3478, "DIRECT", "GEOIP,CN,DIRECT")
	udp.Start()
	rejected := c.Route(NetworkTCP, "ads.test", 80, "REJECT", "DOMAIN-SUFFIX,ads.test,REJECT")
	rejected.Fail(nil)

	tcp.AddUp(100)
	tcp.AddDown(1000)
	udp.AddUp(10)
	udp.AddDown(20)

	s := c.Snapshot(false)
	if s.TCP != 1 || s.UDP != 1 || s.Up != 110 || s.Down != 1020 {
		t.Fatalf("snapshot = %+v", s)
	}
	if s.Hosts != nil || s.DNS != nil || s.Flows != nil {
		t.Fatalf("summary snapshot has detail: %+v", s)
	}

	tcp.Close()
	tcp.Close() // second close is ignored
	udp.Close()
	rejected.Close() // never started
	s = c.Snapshot(false)
	if s.TCP != 0 || s.UDP != 0 || s.Up != 110 || s.Down != 1020 {
		t.Fatalf("after close = %+v", s)
	}
}

func TestDetailListsHostsDNSAndFlows(t *testing.T) {
	c, clk := newTestCollector(10, 10, 10)
	a := c.Route(NetworkTCP, "a.test", 443, "PROXY", "FINAL,PROXY")
	a.Start()
	a.AddDown(5)
	clk.advance(time.Second)
	b := c.Route(NetworkTCP, "b.test", 80, "DIRECT", "GEOIP,CN,DIRECT")
	b.Fail(errors.New("connection refused"))
	c.RecordDNS("b.test", []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil)
	c.RecordDNS("c.test", nil, errors.New("no such host"))

	s := c.Snapshot(true)
	if len(s.Hosts) != 2 || s.Hosts[0].Host != "b.test" || s.Hosts[1].Host != "a.test" {
		t.Fatalf("hosts = %+v", s.Hosts)
	}
	if h := s.Hosts[1]; h.TCP != 1 || h.Down != 5 || h.Action != "PROXY" || h.Rule != "FINAL,PROXY" {
		t.Fatalf("host a = %+v", h)
	}
	if len(s.Flows) != 2 || s.Flows[0].Host != "b.test" || s.Flows[0].Error != "connection refused" || s.Flows[1].Host != "a.test" {
		t.Fatalf("flows = %+v", s.Flows)
	}
	if len(s.DNS) != 2 {
		t.Fatalf("dns = %+v", s.DNS)
	}
	data, err := json.Marshal(s)
	if err != nil || !strings.Contains(string(data), `"addrs":["192.0.2.1"]`) {
		t.Fatalf("json = %s, %v", data, err)
	}
}

func TestEvictsQuietIdleHosts(t *testing.T) {
	c, clk := newTestCollector(2, 10, 10)
	busy := c.Route(NetworkTCP, "busy.test", 443, "PROXY", "FINAL,PROXY")
	busy.Start()
	clk.advance(time.Second)
	c.Route(NetworkTCP, "old.test", 443, "PROXY", "FINAL,PROXY").Fail(nil)
	clk.advance(time.Second)
	c.Route(NetworkTCP, "new.test", 443, "PROXY", "FINAL,PROXY").Fail(nil)

	hosts := map[string]bool{}
	for _, h := range c.Snapshot(true).Hosts {
		hosts[h.Host] = true
	}
	// busy.test is the oldest but still has an open flow.
	if len(hosts) != 2 || !hosts["busy.test"] || !hosts["new.test"] {
		t.Fatalf("hosts = %v", hosts)
	}
}

func TestBoundsDNSAndFlows(t *testing.T) {
	c, clk := newTestCollector(10, 2, 3)
	for _, name := range []string{"a", "b", "a", "c"} {
		c.RecordDNS(name, nil, nil)
		clk.advance(time.Second)
	}
	for i := range 5 {
		c.Route(NetworkUDP, "h", uint16(i), "PROXY", "FINAL,PROXY").Fail(nil)
	}
	s := c.Snapshot(true)
	if len(s.DNS) != 2 || s.DNS[0].Name != "c" || s.DNS[1].Name != "a" {
		t.Fatalf("dns = %+v", s.DNS)
	}
	if len(s.Flows) != 3 || s.Flows[0].Port != 4 || s.Flows[2].Port != 2 {
		t.Fatalf("flows = %+v", s.Flows)
	}
}

func TestWrapCountsAppSide(t *testing.T) {
	c, _ := newTestCollector(10, 10, 10)
	f := c.Route(NetworkTCP, "pipe.test", 1, "PROXY", "FINAL,PROXY")
	f.Start()
	app, engine := net.Pipe()
	defer app.Close()
	conn := f.Wrap(engine)
	go func() {
		_, _ = app.Write([]byte("hello"))
		buf := make([]byte, 3)
		_, _ = app.Read(buf)
	}()
	buf := make([]byte, 5)
	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if s := c.Snapshot(false); s.Up != 5 || s.Down != 3 {
		t.Fatalf("snapshot = %+v", s)
	}
}
