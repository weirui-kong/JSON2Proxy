package flow

import (
	"errors"
	"testing"
	"time"
)

func TestByteWindowAdaptsWithinBounds(t *testing.T) {
	w := NewByteWindow()
	for range 40 {
		size := w.Limit()
		if err := w.Acquire(size); err != nil {
			t.Fatal(err)
		}
		w.Release(size, time.Millisecond, nil)
	}
	if got := w.Limit(); got <= InitialWindow || got > MaxWindow {
		t.Fatalf("grown window = %d", got)
	}
	for range 20 {
		if err := w.Acquire(MinWindow); err != nil {
			t.Fatal(err)
		}
		w.Release(MinWindow, 30*time.Millisecond, nil)
	}
	if got := w.Limit(); got != MinWindow {
		t.Fatalf("minimum window = %d, want %d", got, MinWindow)
	}
}

func TestByteWindowCloseUnblocksAcquire(t *testing.T) {
	w := NewByteWindow()
	if err := w.Acquire(InitialWindow); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		result <- w.Acquire(1)
	}()
	w.Close()
	select {
	case err := <-result:
		if !errors.Is(err, ErrWindowClosed) {
			t.Fatalf("Acquire error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Acquire remained blocked after Close")
	}
}
