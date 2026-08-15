package main

import (
	"bytes"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"json2proxy/internal/config"
)

func TestDirectUDPEgressRoundTrip(t *testing.T) {
	echo, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		buffer := make([]byte, 1024)
		n, peer, readErr := echo.ReadFromUDP(buffer)
		if readErr == nil {
			_, _ = echo.WriteToUDP(buffer[:n], peer)
		}
	}()
	var drops atomic.Uint64
	egress, err := newDirectUDPEgress(8, 61440, &drops)
	if err != nil {
		t.Fatal(err)
	}
	defer egress.Close()
	target := echo.LocalAddr().(*net.UDPAddr)
	if err = egress.WriteDatagram(target.IP.String(), uint16(target.Port), []byte("hello")); err != nil {
		t.Fatal(err)
	}
	host, port, data, err := egress.ReadDatagram(time.Now().Add(time.Second), nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "127.0.0.1" || port != uint16(target.Port) || !bytes.Equal(data, []byte("hello")) {
		t.Fatalf("unexpected response: %s:%d %q", host, port, data)
	}
}

func TestUDPAssociationLimitAndRate(t *testing.T) {
	a := &app{
		cfg: config.Server{UDP: config.ServerUDP{
			MaxAssociationsPerUser:  1,
			PacketsPerSecondPerUser: 1,
			BytesPerSecondPerUser:   4,
		}},
		udpActive: make(map[string]int), udpRates: make(map[string]*udpRateState),
	}
	release, ok := a.claimUDP("user")
	if !ok || release == nil {
		t.Fatal("first UDP association rejected")
	}
	if _, ok = a.claimUDP("user"); ok {
		t.Fatal("UDP association limit ignored")
	}
	release()
	if !a.allowUDPRate("user", 4) || a.allowUDPRate("user", 1) {
		t.Fatal("UDP per-user rate limit not enforced")
	}
}

func TestUDPNeverAllowsMulticastOrUnspecified(t *testing.T) {
	for _, value := range []string{"0.0.0.0", "255.255.255.255", "224.0.0.1", "::", "ff02::1"} {
		if !udpNeverAllowed(net.ParseIP(value)) {
			t.Fatalf("unsafe UDP target accepted: %s", value)
		}
	}
	if udpNeverAllowed(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback must remain available when allow_private=true")
	}
}
