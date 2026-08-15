package socks5

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestUDPDatagramRoundTrip(t *testing.T) {
	for _, host := range []string{"203.0.113.7", "2001:db8::7", "example.com"} {
		packet, err := EncodeUDPDatagram(host, 53, []byte("dns"))
		if err != nil {
			t.Fatalf("encode %s: %v", host, err)
		}
		gotHost, gotPort, gotData, err := ParseUDPDatagram(packet)
		if err != nil {
			t.Fatalf("parse %s: %v", host, err)
		}
		if gotHost != host || gotPort != 53 || !bytes.Equal(gotData, []byte("dns")) {
			t.Fatalf("unexpected datagram: %q %d %q", gotHost, gotPort, gotData)
		}
	}
}

func TestAssociateUDPRoundTrip(t *testing.T) {
	relay, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	clientControl, proxyControl := net.Pipe()
	defer proxyControl.Close()
	proxyErr := make(chan error, 1)
	go func() {
		greeting := make([]byte, 3)
		if _, readErr := io.ReadFull(proxyControl, greeting); readErr != nil {
			proxyErr <- readErr
			return
		}
		_, _ = proxyControl.Write([]byte{5, 0})
		request := make([]byte, 10)
		if _, readErr := io.ReadFull(proxyControl, request); readErr != nil {
			proxyErr <- readErr
			return
		}
		port := relay.LocalAddr().(*net.UDPAddr).Port
		reply := []byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0}
		binary.BigEndian.PutUint16(reply[8:], uint16(port))
		_, writeErr := proxyControl.Write(reply)
		proxyErr <- writeErr
	}()
	client := pipeClient(clientControl, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	association, err := client.AssociateUDP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer association.Close()
	if err = <-proxyErr; err != nil {
		t.Fatal(err)
	}
	echoErr := make(chan error, 1)
	go func() {
		buffer := make([]byte, 2048)
		n, peer, readErr := relay.ReadFromUDP(buffer)
		if readErr == nil {
			_, readErr = relay.WriteToUDP(buffer[:n], peer)
		}
		echoErr <- readErr
	}()
	if err = association.WriteDatagram("203.0.113.7", 53, []byte("query")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 2048)
	host, port, data, err := association.ReadDatagram(time.Now().Add(time.Second), buffer)
	if err != nil {
		t.Fatal(err)
	}
	if host != "203.0.113.7" || port != 53 || !bytes.Equal(data, []byte("query")) {
		t.Fatalf("unexpected proxy UDP response: %q %d %q", host, port, data)
	}
	if err = <-echoErr; err != nil {
		t.Fatal(err)
	}
}

func TestUDPDatagramRejectsFragment(t *testing.T) {
	packet, err := EncodeUDPDatagram("203.0.113.7", 53, nil)
	if err != nil {
		t.Fatal(err)
	}
	packet[2] = 1
	if _, _, _, err = ParseUDPDatagram(packet); err == nil {
		t.Fatal("fragmented UDP packet accepted")
	}
}
