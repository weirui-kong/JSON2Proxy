package tun

import (
	"bytes"
	"os"
	"testing"
)

func TestFDDeviceHeader(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	writer := &fdDevice{file: w, header: hasUTunHeader}
	reader := &fdDevice{file: r, header: hasUTunHeader}
	defer writer.Close()
	defer reader.Close()

	v6 := append([]byte{0x60}, bytes.Repeat([]byte{1}, 39)...)
	if err := writer.WritePacket(v6); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1500)
	n, err := reader.ReadPacket(buf)
	if err != nil || !bytes.Equal(buf[:n], v6) {
		t.Fatalf("round trip = %x, err %v", buf[:n], err)
	}
	if !hasUTunHeader {
		return
	}
	// On Apple platforms the wire format carries the address family.
	if err := writer.WritePacket([]byte{0x45, 2, 3}); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 16)
	n, _ = r.Read(raw)
	if !bytes.Equal(raw[:n], []byte{0, 0, 0, afInet, 0x45, 2, 3}) {
		t.Fatalf("utun frame = %x", raw[:n])
	}
}
