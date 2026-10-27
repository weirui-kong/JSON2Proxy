//go:build !darwin && !ios

package mobile

// SetPhysicalInterface is only needed on Apple platforms. Android excludes
// the app from its own VPN, so the app's sockets already bypass the tunnel.
func SetPhysicalInterface(name string) error { return nil }

func bindPhysical(network string, fd uintptr) error { return nil }
