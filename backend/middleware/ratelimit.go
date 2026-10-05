package middleware

import (
	"sync"
	"time"
)

const authFailLimit = 30

type failWindow struct {
	mu    sync.Mutex
	fails []time.Time
}

var authFailWindows sync.Map

// AllowAuth сообщает, можно ли ещё проверять ключ с этого IP.
func AllowAuth(ip string) bool {
	w := windowFor(ip)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.prune(time.Now())
	return len(w.fails) < authFailLimit
}

// RecordAuthFailure учитывает неуспешную проверку ключа.
func RecordAuthFailure(ip string) {
	w := windowFor(ip)
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.prune(now)
	w.fails = append(w.fails, now)
}

func windowFor(ip string) *failWindow {
	if ip == "" {
		ip = "unknown"
	}
	v, _ := authFailWindows.LoadOrStore(ip, &failWindow{})
	return v.(*failWindow)
}

func (w *failWindow) prune(now time.Time) {
	cutoff := now.Add(-time.Minute)
	n := w.fails[:0]
	for _, t := range w.fails {
		if t.After(cutoff) {
			n = append(n, t)
		}
	}
	w.fails = n
}
