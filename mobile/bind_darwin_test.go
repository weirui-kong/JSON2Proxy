//go:build darwin || ios

package mobile

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSetPhysicalInterfaceBindsSockets(t *testing.T) {
	defer SetPhysicalInterface("")
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if err := SetPhysicalInterface("no-such-interface0"); err == nil {
		t.Fatal("unknown interface accepted")
	}
	// Bound to loopback, a loopback address is still reachable.
	if err := SetPhysicalInterface("lo0"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := dialProtected(ctx, "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial bound to lo0: %v", err)
	}
	conn.Close()
	if boundInterface.Load() == 0 {
		t.Fatal("interface index not stored")
	}
}
