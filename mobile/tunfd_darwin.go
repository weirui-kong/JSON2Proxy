//go:build darwin || ios

package mobile

import (
	"errors"

	"golang.org/x/sys/unix"
)

const utunControlName = "com.apple.net.utun_control"

// TunnelFD returns a duplicate of the utun file descriptor that
// NEPacketTunnelProvider created for this process. Call it after
// setTunnelNetworkSettings completes and pass the result to StartEngine,
// which closes it on stop; the system keeps its own descriptor.
func TunnelFD() (int, error) {
	info := &unix.CtlInfo{}
	copy(info.Name[:], utunControlName)
	for fd := 0; fd < 1024; fd++ {
		addr, err := unix.Getpeername(fd)
		if err != nil {
			continue
		}
		ctl, ok := addr.(*unix.SockaddrCtl)
		if !ok {
			continue
		}
		if info.Id == 0 {
			if err := unix.IoctlCtlInfo(fd, info); err != nil {
				continue
			}
		}
		if ctl.ID == info.Id {
			return unix.Dup(fd)
		}
	}
	return -1, errors.New("utun file descriptor not found")
}
