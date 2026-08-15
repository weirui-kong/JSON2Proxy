package flow

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

const (
	MinFrame     = 4 << 10
	InitialFrame = 16 << 10
	MaxFrame     = 64 << 10
)

type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeThroughput  Mode = "throughput"
)

// AdaptiveReader coalesces small adjacent TCP reads for a very short period and
// adjusts its target frame size from observed downstream write latency.
//
// Reads and writes remain synchronous, so TCP backpressure still bounds memory use.
type AdaptiveReader struct {
	target      int
	minimum     int
	fullStreak  int
	smallStreak int
	coalesce    time.Duration
}

var framePool = sync.Pool{
	New: func() any {
		return make([]byte, MaxFrame)
	},
}

func NewAdaptiveReader(modes ...Mode) *AdaptiveReader {
	mode := ModeInteractive
	if len(modes) != 0 {
		mode = modes[0]
	}
	r := &AdaptiveReader{
		target:   InitialFrame,
		minimum:  MinFrame,
		coalesce: time.Millisecond,
	}
	if mode == ModeThroughput {
		r.target = MaxFrame
		r.minimum = 16 << 10
		r.coalesce = 2 * time.Millisecond
	}
	return r
}

func (r *AdaptiveReader) Target() int {
	return r.target
}

// ReadFrame returns one frame and a terminal source error, if both happened in
// the same read. Callers must send non-empty data before handling the error.
func (r *AdaptiveReader) ReadFrame(conn net.Conn) ([]byte, error) {
	payload := framePool.Get().([]byte)[:r.target]
	n, readErr := conn.Read(payload)
	if n == 0 {
		ReleaseFrame(payload)
		return nil, readErr
	}
	if readErr == nil && n < r.target {
		_ = conn.SetReadDeadline(time.Now().Add(r.coalesce))
		for n < r.target {
			m, err := conn.Read(payload[n:r.target])
			n += m
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					readErr = nil
				} else {
					readErr = err
				}
				break
			}
		}
		_ = conn.SetReadDeadline(time.Time{})
	}
	return payload[:n], readErr
}

// Observe applies a bounded additive/increase, multiplicative/decrease policy
// to the next frame target. The thresholds deliberately react slowly to avoid
// oscillating on ordinary interactive traffic.
func (r *AdaptiveReader) Observe(size int, writeDuration time.Duration, writeErr error) {
	if writeErr != nil || writeDuration >= 25*time.Millisecond {
		r.shrink()
		return
	}
	if size >= r.target*3/4 && writeDuration <= 8*time.Millisecond {
		r.fullStreak++
		r.smallStreak = 0
		if r.fullStreak >= 2 && r.target < MaxFrame {
			r.target *= 2
			if r.target > MaxFrame {
				r.target = MaxFrame
			}
			r.fullStreak = 0
		}
		return
	}
	r.fullStreak = 0
	if size <= r.target/4 {
		r.smallStreak++
		if r.smallStreak >= 4 {
			r.shrink()
		}
	} else {
		r.smallStreak = 0
	}
}

func (r *AdaptiveReader) shrink() {
	r.target /= 2
	if r.target < r.minimum {
		r.target = r.minimum
	}
	r.fullStreak = 0
	r.smallStreak = 0
}

func IsNormalClose(err error) bool {
	return err == nil ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, net.ErrClosed)
}

func ReleaseFrame(payload []byte) {
	if cap(payload) < MaxFrame {
		return
	}
	framePool.Put(payload[:MaxFrame])
}
