package socks5

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func pipeClient(clientConn net.Conn, username, password string) *Client {
	return &Client{
		address: "proxy.test:1080", username: username, password: password,
		connect: func(context.Context, string, string) (net.Conn, error) {
			return clientConn, nil
		},
	}
}

func TestDialContextNoAuthIPv4(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		if err := expectBytes(serverConn, []byte{5, 1, 0}); err != nil {
			serverErr <- err
			return
		}
		if _, err := serverConn.Write([]byte{5, 0}); err != nil {
			serverErr <- err
			return
		}
		if err := expectBytes(serverConn, []byte{5, 1, 0, 1, 203, 0, 113, 7, 1, 187}); err != nil {
			serverErr <- err
			return
		}
		_, err := serverConn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
		serverErr <- err
	}()

	got, err := pipeClient(clientConn, "", "").DialContext(context.Background(), "tcp", "203.0.113.7:443")
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	if got != clientConn {
		t.Fatal("SOCKS5 dialer did not return the original transport connection")
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestDialContextUsernamePasswordIPv6(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		if err := expectBytes(serverConn, []byte{5, 1, 2}); err != nil {
			serverErr <- err
			return
		}
		if _, err := serverConn.Write([]byte{5, 2}); err != nil {
			serverErr <- err
			return
		}
		if err := expectBytes(serverConn, []byte{1, 4, 'u', 's', 'e', 'r', 4, 'p', 'a', 's', 's'}); err != nil {
			serverErr <- err
			return
		}
		if _, err := serverConn.Write([]byte{1, 0}); err != nil {
			serverErr <- err
			return
		}
		wantRequest := append([]byte{5, 1, 0, 4}, net.ParseIP("2001:db8::7").To16()...)
		wantRequest = append(wantRequest, 0, 53)
		if err := expectBytes(serverConn, wantRequest); err != nil {
			serverErr <- err
			return
		}
		reply := append([]byte{5, 0, 0, 3, 5}, []byte("proxy")...)
		reply = append(reply, 0, 1)
		_, err := serverConn.Write(reply)
		serverErr <- err
	}()

	got, err := pipeClient(clientConn, "user", "pass").DialContext(context.Background(), "tcp", "[2001:db8::7]:53")
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestDialContextReportsConnectRejection(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	go func() {
		defer serverConn.Close()
		greeting := make([]byte, 3)
		_, _ = io.ReadFull(serverConn, greeting)
		_, _ = serverConn.Write([]byte{5, 0})
		request := make([]byte, 10)
		_, _ = io.ReadFull(serverConn, request)
		_, _ = serverConn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
	}()

	_, err := pipeClient(clientConn, "", "").DialContext(context.Background(), "tcp", "198.51.100.1:80")
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("unexpected rejection error: %v", err)
	}
}

func TestDialContextHonorsCancellationDuringHandshake(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	go func() {
		greeting := make([]byte, 3)
		_, _ = io.ReadFull(serverConn, greeting)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := pipeClient(clientConn, "", "").DialContext(ctx, "tcp", "203.0.113.1:80")
	if err == nil {
		t.Fatal("handshake ignored context timeout")
	}
}

func TestConnectRequestSupportsRemoteDNS(t *testing.T) {
	got, err := connectRequest("example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{5, 1, 0, 3, byte(len("example.com"))}, []byte("example.com")...)
	want = append(want, 1, 187)
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected SOCKS5 domain request: %v", got)
	}
}

func expectBytes(reader io.Reader, want []byte) error {
	got := make([]byte, len(want))
	if _, err := io.ReadFull(reader, got); err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return errors.New("unexpected SOCKS5 wire bytes")
	}
	return nil
}
