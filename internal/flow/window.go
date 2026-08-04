package flow

import (
	"errors"
	"sync"
	"time"
)

const (
	MinWindow     = 64 << 10
	InitialWindow = 256 << 10
	MaxWindow     = 4 << 20
	WindowStep    = 256 << 10
)

var ErrWindowClosed = errors.New("flow window closed")

// ByteWindow bounds queued frame bytes and adapts the bound from downstream
// write latency. It is deliberately an application memory window.
type ByteWindow struct {
	mu         sync.Mutex
	cond       *sync.Cond
	limit      int
	used       int
	fastStreak int
	closed     bool
}

func NewByteWindow() *ByteWindow {
	w := &ByteWindow{limit: InitialWindow}
	w.cond = sync.NewCond(&w.mu)
	return w
}

func (w *ByteWindow) Acquire(size int) error {
	if size <= 0 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for !w.closed && w.used+size > w.limit {
		w.cond.Wait()
	}
	if w.closed {
		return ErrWindowClosed
	}
	w.used += size
	return nil
}

func (w *ByteWindow) Release(size int, writeDuration time.Duration, writeErr error) {
	if size <= 0 {
		return
	}
	w.mu.Lock()
	saturated := w.used >= w.limit*3/4
	w.used -= size
	if w.used < 0 {
		w.used = 0
	}
	switch {
	case writeErr != nil || writeDuration >= 25*time.Millisecond:
		w.limit /= 2
		if w.limit < MinWindow {
			w.limit = MinWindow
		}
		w.fastStreak = 0
	case saturated && writeDuration <= 8*time.Millisecond:
		w.fastStreak++
		if w.fastStreak >= 4 && w.limit < MaxWindow {
			w.limit += WindowStep
			if w.limit > MaxWindow {
				w.limit = MaxWindow
			}
			w.fastStreak = 0
		}
	default:
		w.fastStreak = 0
	}
	w.cond.Broadcast()
	w.mu.Unlock()
}

func (w *ByteWindow) Close() {
	w.mu.Lock()
	w.closed = true
	w.cond.Broadcast()
	w.mu.Unlock()
}

func (w *ByteWindow) Limit() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.limit
}
