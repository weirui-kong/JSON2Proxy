//go:build darwin || ios

package tun

// utun packets start with the address family in network byte order.
const (
	hasUTunHeader = true
	afInet        = 2
	afInet6       = 30
)
