//go:build !darwin && !ios

package tun

// Android's VpnService opens TUN with IFF_NO_PI, so packets carry no header.
const (
	hasUTunHeader = false
	afInet        = 0
	afInet6       = 0
)
