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

// AdaptiveReader coalesces small adjacent TCP reads for a very short period and
// adjusts its target frame size from observed downstream write latency.
//
// Reads and writes remain synchronous, so TCP backpressure still bounds memory use.
type AdaptiveReader struct {
	target      int
	fullStreak  int
	smallStreak int
	coalesce    time.Duration
	buf         []byte
}

var framePool = sync.Pool{
	New: func() any {
		return make([]byte, MaxFrame)
	},
}

func NewAdaptiveReader() *AdaptiveReader {
	return &AdaptiveReader{
		target:   InitialFrame,
		coalesce: time.Millisecond,
		buf:      make([]byte, MaxFrame),
	}
}

func (r *AdaptiveReader) Target() int {
	return r.target
}

// ReadFrame returns one frame and a terminal source error, if both happened in
// the same read. Callers must send non-empty data before handling the error.
func (r *AdaptiveReader) ReadFrame(conn net.Conn) ([]byte, error) {
	n, readErr := conn.Read(r.buf[:r.target])
	if n == 0 {
		return nil, readErr
	}
	if readErr == nil && n < r.target {
		_ = conn.SetReadDeadline(time.Now().Add(r.coalesce))
		for n < r.target {
			m, err := conn.Read(r.buf[n:r.target])
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
	payload := framePool.Get().([]byte)
	copy(payload, r.buf[:n])
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
	if r.target < MinFrame {
		r.target = MinFrame
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
