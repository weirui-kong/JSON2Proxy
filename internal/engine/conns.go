package engine

import "sync"

// connSet tracks the connections of active flows, so they can all be closed
// when the device switches networks or the engine stops. Apps then reconnect
// through the current network instead of keeping flows that were set up over
// the previous one.
type connSet struct {
	mu     sync.Mutex
	next   uint64
	open   map[uint64][]func()
	closed bool
}

// add registers the close functions of one flow and returns a function that
// unregisters them when the flow ends. After closeAll(true) new flows are
// closed at once.
func (s *connSet) add(closers ...func()) (remove func()) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		for _, c := range closers {
			c()
		}
		return func() {}
	}
	if s.open == nil {
		s.open = make(map[uint64][]func())
	}
	id := s.next
	s.next++
	s.open[id] = closers
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.open, id)
		s.mu.Unlock()
	}
}

// closeAll closes every registered flow and returns how many there were.
// With final set, flows added later are closed immediately.
func (s *connSet) closeAll(final bool) int {
	s.mu.Lock()
	open := s.open
	s.open = nil
	if final {
		s.closed = true
	}
	s.mu.Unlock()
	for _, closers := range open {
		for _, c := range closers {
			c()
		}
	}
	return len(open)
}
