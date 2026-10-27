package tun

import (
	"os"
	"sync"
)

// Device moves raw IP packets, without any link header, between the system
// VPN interface and the stack.
type Device interface {
	ReadPacket(buf []byte) (int, error)
	WritePacket(packet []byte) error
	Close() error
}

// fdDevice reads and writes a TUN file descriptor. Apple's utun prefixes
// every packet with a 4-byte address family; Linux/Android TUN does not.
type fdDevice struct {
	file    *os.File
	header  bool
	readBuf []byte
	writeMu sync.Mutex
	wbuf    []byte
}

const utunHeaderLen = 4

// NewFDDevice wraps a TUN file descriptor. The device takes ownership of fd
// and closes it in Close.
func NewFDDevice(fd int) Device {
	return &fdDevice{file: os.NewFile(uintptr(fd), "tun"), header: hasUTunHeader}
}

func (d *fdDevice) ReadPacket(buf []byte) (int, error) {
	if !d.header {
		return d.file.Read(buf)
	}
	if cap(d.readBuf) < len(buf)+utunHeaderLen {
		d.readBuf = make([]byte, len(buf)+utunHeaderLen)
	}
	for {
		n, err := d.file.Read(d.readBuf[:len(buf)+utunHeaderLen])
		if err != nil {
			return 0, err
		}
		if n > utunHeaderLen {
			return copy(buf, d.readBuf[utunHeaderLen:n]), nil
		}
	}
}

func (d *fdDevice) WritePacket(packet []byte) error {
	if !d.header {
		_, err := d.file.Write(packet)
		return err
	}
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	family := byte(afInet)
	if len(packet) > 0 && packet[0]>>4 == 6 {
		family = afInet6
	}
	d.wbuf = append(append(d.wbuf[:0], 0, 0, 0, family), packet...)
	_, err := d.file.Write(d.wbuf)
	return err
}

func (d *fdDevice) Close() error { return d.file.Close() }
