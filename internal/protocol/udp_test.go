package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestUDPDatagramRoundTrip(t *testing.T) {
	for _, host := range []string{"203.0.113.7", "2001:db8::7", "example.com"} {
		payload, err := UDPDatagramPayload(host, 53, []byte("query"))
		if err != nil {
			t.Fatalf("encode %s: %v", host, err)
		}
		gotHost, gotPort, gotData, err := ParseUDPDatagram(payload)
		if err != nil {
			t.Fatalf("parse %s: %v", host, err)
		}
		if gotHost != host || gotPort != 53 || !bytes.Equal(gotData, []byte("query")) {
			t.Fatalf("unexpected round trip: %q %d %q", gotHost, gotPort, gotData)
		}
		if _, err = EncodeFrame(Frame{Type: FrameUDPDatagram, Payload: payload}); err != nil {
			t.Fatalf("frame rejected: %v", err)
		}
	}
}

func TestUDPDatagramAllowsEmptyData(t *testing.T) {
	payload, err := UDPDatagramPayload("203.0.113.7", 9, nil)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := DecodeFrame(mustEncodedFrame(t, Frame{Type: FrameUDPDatagram, Payload: payload}))
	if err != nil || frame.Type != FrameUDPDatagram {
		t.Fatalf("empty UDP datagram rejected: %v", err)
	}
}

func TestUDPDatagramLimitsAndLength(t *testing.T) {
	if _, err := UDPDatagramPayload("example.com", 53, make([]byte, MaxUDPDatagram+1)); err == nil {
		t.Fatal("oversize UDP datagram accepted")
	}
	payload, err := UDPDatagramPayload("example.com", 53, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	payload[len(payload)-2] = 9
	if _, _, _, err = ParseUDPDatagram(payload); err == nil {
		t.Fatal("invalid UDP data length accepted")
	}
	if _, err = UDPDatagramPayload(strings.Repeat("x", 254), 53, nil); err == nil {
		t.Fatal("oversize domain accepted")
	}
}

func TestUDPControlFrames(t *testing.T) {
	for _, frame := range []Frame{
		{Type: FrameUDPOpen, Payload: UDPOpenPayload()},
		{Type: FrameUDPOpenResult, Payload: Result(0, "")},
		{Type: FrameUDPClose, Payload: Result(0, "")},
	} {
		encoded := mustEncodedFrame(t, frame)
		if _, err := DecodeFrame(encoded); err != nil {
			t.Fatalf("valid UDP control frame rejected: %v", err)
		}
	}
	bad := mustEncodedFrame(t, Frame{Type: FrameUDPOpen, Payload: []byte{2, 0}})
	if _, err := DecodeFrame(bad); err == nil {
		t.Fatal("unknown UDP extension version accepted")
	}
}

func mustEncodedFrame(t *testing.T, frame Frame) []byte {
	t.Helper()
	b, err := EncodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
