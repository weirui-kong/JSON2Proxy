package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Logger struct {
	base *log.Logger
	file *limitedFile
}

func New(path, size string) (*Logger, error) {
	n, e := parseSize(size)
	if e != nil {
		return nil, e
	}
	l := &Logger{base: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)}
	if n > 0 {
		f, e := newLimitedFile(path, n)
		if e != nil {
			return nil, e
		}
		l.file = f
		l.base.SetOutput(io.MultiWriter(os.Stdout, f))
	}
	return l, nil
}
func (l *Logger) Printf(format string, args ...any) { l.base.Printf(format, args...) }
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
func parseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || s == "0" || s == "0M" {
		return 0, nil
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "K"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "KB"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "M"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "MB"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "G"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "GB"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "GB")
	}
	v, e := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if e != nil || v < 0 {
		return 0, fmt.Errorf("invalid log_size %q", s)
	}
	return v * mult, nil
}

type limitedFile struct {
	path string
	max  int64
	f    *os.File
	mu   sync.Mutex
}

func newLimitedFile(path string, max int64) (*limitedFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if e != nil {
		return nil, e
	}
	return &limitedFile{path, max, f, sync.Mutex{}}, nil
}
func (l *limitedFile) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n, e := l.f.Write(p)
	if e == nil {
		e = l.trim()
	}
	return n, e
}
func (l *limitedFile) trim() error {
	info, e := l.f.Stat()
	if e != nil || info.Size() <= l.max {
		return e
	}
	if e = l.f.Close(); e != nil {
		return e
	}
	all, e := os.ReadFile(l.path)
	if e != nil {
		return e
	}
	keep := all
	if int64(len(keep)) > l.max {
		keep = keep[len(keep)-int(l.max):]
		if i := strings.IndexByte(string(keep), '\n'); i >= 0 {
			keep = keep[i+1:]
		}
	}
	tmp := l.path + ".tmp"
	if e = os.WriteFile(tmp, keep, 0640); e != nil {
		return e
	}
	if e = os.Rename(tmp, l.path); e != nil {
		return e
	}
	l.f, e = os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, 0640)
	return e
}
func (l *limitedFile) Close() error { l.mu.Lock(); defer l.mu.Unlock(); return l.f.Close() }
