//go:build darwin || ios

package mobile

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// boundInterface is the index of the physical interface the app's own
// sockets are bound to; 0 leaves them to the routing table.
var boundInterface atomic.Int32

// SetPhysicalInterface binds the app's own sockets (TCP ping and its
// bootstrap DNS lookups) to the named Wi-Fi or cellular interface. The iOS
// app is not excluded from its own tunnel, so without this its sockets would
// enter the VPN. Pass "" to stop binding. The packet tunnel extension does
// not call it: its sockets bypass the tunnel already.
func SetPhysicalInterface(name string) error {
	if name == "" {
		boundInterface.Store(0)
		return nil
	}
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("interface %q: %w", name, err)
	}
	boundInterface.Store(int32(iface.Index))
	return nil
}

// bindPhysical applies SetPhysicalInterface to a new socket. network is the
// dialer's network, such as "tcp4" or "udp6".
func bindPhysical(network string, fd uintptr) error {
	index := int(boundInterface.Load())
	if index == 0 {
		return nil
	}
	if strings.HasSuffix(network, "6") {
		return unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, index)
	}
	return unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, index)
}
