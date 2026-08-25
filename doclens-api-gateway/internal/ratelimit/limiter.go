package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	clients map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func New(ratePerSecond, burst int) *Limiter {
	return &Limiter{
		rate:    float64(ratePerSecond),
		burst:   float64(burst),
		clients: make(map[string]*bucket),
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.clients[key]
	if !ok {
		l.clients[key] = &bucket{tokens: l.burst - 1, seen: now}
		return true
	}
	elapsed := now.Sub(b.seen).Seconds()
	b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
	b.seen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-maxAge)
	for key, b := range l.clients {
		if b.seen.Before(cutoff) {
			delete(l.clients, key)
		}
	}
}
