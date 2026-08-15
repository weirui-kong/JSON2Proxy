package flow

import (
	"errors"
	"io"
	"testing"
	"time"
)

func TestAdaptiveReaderGrowsAndShrinks(t *testing.T) {
	r := NewAdaptiveReader()
	if got := r.Target(); got != InitialFrame {
		t.Fatalf("initial target = %d, want %d", got, InitialFrame)
	}
	r.Observe(InitialFrame, time.Millisecond, nil)
	r.Observe(InitialFrame, time.Millisecond, nil)
	if got := r.Target(); got != 2*InitialFrame {
		t.Fatalf("grown target = %d, want %d", got, 2*InitialFrame)
	}
	r.Observe(r.Target(), 30*time.Millisecond, nil)
	if got := r.Target(); got != InitialFrame {
		t.Fatalf("shrunk target = %d, want %d", got, InitialFrame)
	}
}

func TestAdaptiveReaderBounds(t *testing.T) {
	r := NewAdaptiveReader()
	for range 20 {
		r.Observe(r.Target(), time.Millisecond, nil)
		r.Observe(r.Target(), time.Millisecond, nil)
	}
	if got := r.Target(); got != MaxFrame {
		t.Fatalf("maximum target = %d, want %d", got, MaxFrame)
	}
	for range 20 {
		r.Observe(0, time.Second, errors.New("blocked"))
	}
	if got := r.Target(); got != MinFrame {
		t.Fatalf("minimum target = %d, want %d", got, MinFrame)
	}
}

func TestThroughputModeStartsAtMaximumFrame(t *testing.T) {
	r := NewAdaptiveReader(ModeThroughput)
	if got := r.Target(); got != MaxFrame {
		t.Fatalf("throughput initial target = %d, want %d", got, MaxFrame)
	}
	for range 20 {
		r.Observe(0, time.Second, errors.New("blocked"))
	}
	if got := r.Target(); got != 16<<10 {
		t.Fatalf("throughput minimum target = %d, want %d", got, 16<<10)
	}
}

func TestIsNormalClose(t *testing.T) {
	if !IsNormalClose(nil) || !IsNormalClose(io.EOF) || !IsNormalClose(io.ErrClosedPipe) {
		t.Fatal("nil, EOF, and closed pipe must be normal closes")
	}
	if IsNormalClose(errors.New("boom")) {
		t.Fatal("arbitrary errors must not be normal closes")
	}
}
