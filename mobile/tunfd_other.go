//go:build !darwin && !ios

package mobile

import "errors"

// TunnelFD is only needed on Apple platforms; Android passes the descriptor
// from VpnService.Builder.establish to StartEngine directly.
func TunnelFD() (int, error) {
	return -1, errors.New("TunnelFD is only available on Apple platforms")
}
