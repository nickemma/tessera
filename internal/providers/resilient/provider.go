package resilient

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
	"github.com/nickemma/tessera/internal/modules/inference/ports"
)

var ErrOpen = errors.New("model circuit breaker open")

type Provider struct {
	next        ports.Provider
	maxFailures int
	cooldown    time.Duration
	mu          sync.Mutex
	failures    int
	openUntil   time.Time
}

func New(next ports.Provider, maxFailures int, cooldown time.Duration) *Provider {
	return &Provider{next: next, maxFailures: maxFailures, cooldown: cooldown}
}

func (p *Provider) Complete(ctx context.Context, request domain.Request, emit func(string) error) (domain.Response, error) {
	p.mu.Lock()
	if time.Now().Before(p.openUntil) {
		p.mu.Unlock()
		return domain.Response{}, ErrOpen
	}
	p.mu.Unlock()

	var response domain.Response
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		emitted := false
		response, err = p.next.Complete(ctx, request, func(text string) error {
			emitted = true
			if emit == nil {
				return nil
			}
			return emit(text)
		})
		if err == nil || emitted || attempt == 1 {
			break
		}
		select {
		case <-time.After(25 * time.Millisecond):
		case <-ctx.Done():
			return response, ctx.Err()
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		p.failures = 0
		return response, nil
	}
	p.failures++
	if p.failures >= p.maxFailures {
		p.openUntil = time.Now().Add(p.cooldown)
	}
	return response, err
}
