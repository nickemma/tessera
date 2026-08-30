package memory

import (
	"context"
	"sync"
	"time"

	"github.com/nickemma/tessera/internal/modules/ratelimit/domain"
)

type Limiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	bucket map[string]bucket
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func NewLimiter(ratePerSecond, burst int) *Limiter {
	return &Limiter{rate: float64(ratePerSecond), burst: float64(burst), bucket: make(map[string]bucket)}
}

func (l *Limiter) Allow(_ context.Context, tenantID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	current, ok := l.bucket[tenantID]
	if !ok {
		current = bucket{tokens: l.burst, seen: now}
	}
	current.tokens += now.Sub(current.seen).Seconds() * l.rate
	if current.tokens > l.burst {
		current.tokens = l.burst
	}
	current.seen = now
	if current.tokens < 1 {
		l.bucket[tenantID] = current
		return domain.ErrLimited
	}
	current.tokens--
	l.bucket[tenantID] = current
	return nil
}
