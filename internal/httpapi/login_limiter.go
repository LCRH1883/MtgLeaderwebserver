package httpapi

import (
	"sync"
	"time"
)

type loginLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	max     int
	entries map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		window:  5 * time.Minute,
		max:     10,
		entries: make(map[string][]time.Time),
	}
}

func (l *loginLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	ts := l.entries[key]

	kept := ts[:0]
	for _, t := range ts {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	ts = kept
	if len(ts) >= l.max {
		l.entries[key] = ts
		return false
	}

	ts = append(ts, now)
	l.entries[key] = ts
	return true
}
