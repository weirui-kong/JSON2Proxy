package mobile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// totalsInterval is how often running totals are written while they change.
const totalsInterval = 10 * time.Second

// trafficTotals are the bytes of every engine run so far.
type trafficTotals struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// totalsRecorder adds the running engine's counters to the totals saved by
// earlier runs and writes the sum to a file, so the totals survive engine
// restarts and the process being stopped.
type totalsRecorder struct {
	path     string
	base     trafficTotals
	counters func() trafficTotals // bytes since this engine started

	stop chan struct{}
	done chan struct{}

	mu    sync.Mutex
	saved trafficTotals
}

// startTotals loads the totals in path and saves them periodically until
// close. An empty path keeps nothing: the totals are this run's counters.
func startTotals(path string, counters func() trafficTotals) *totalsRecorder {
	r := &totalsRecorder{path: path, counters: counters, stop: make(chan struct{}), done: make(chan struct{})}
	if path == "" {
		close(r.done)
		return r
	}
	base, err := loadTotals(path)
	if err != nil {
		logf(LogWarn, "Totals", "Starting traffic totals from zero: "+err.Error())
	}
	r.base, r.saved = base, base
	go r.loop()
	return r
}

func (r *totalsRecorder) current() trafficTotals {
	c := r.counters()
	return trafficTotals{Up: r.base.Up + c.Up, Down: r.base.Down + c.Down}
}

func (r *totalsRecorder) loop() {
	defer close(r.done)
	ticker := time.NewTicker(totalsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.save()
		case <-r.stop:
			return
		}
	}
}

// close stops the periodic writes and saves the final totals. Call it after
// the engine has closed its flows so their last bytes are counted.
func (r *totalsRecorder) close() {
	if r.path == "" {
		return
	}
	close(r.stop)
	<-r.done
	r.save()
}

func (r *totalsRecorder) save() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.current()
	if now == r.saved {
		return
	}
	if err := writeTotals(r.path, now); err != nil {
		logf(LogWarn, "Totals", "Saving traffic totals failed: "+err.Error())
		return
	}
	r.saved = now
}

func loadTotals(path string) (trafficTotals, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return trafficTotals{}, nil
	}
	if err != nil {
		return trafficTotals{}, err
	}
	var t trafficTotals
	if err := json.Unmarshal(data, &t); err != nil || t.Up < 0 || t.Down < 0 {
		return trafficTotals{}, errors.New("unreadable totals file")
	}
	return t, nil
}

// writeTotals replaces path through a temporary file, so a reader never sees
// a partial write.
func writeTotals(path string, t trafficTotals) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".totals-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
