package memory

import (
	"context"
	"sync"
	"time"

	"github.com/nickemma/tessera/internal/modules/metering/domain"
)

type Ledger struct {
	mu     sync.RWMutex
	events []domain.UsageEvent
}

func NewLedger() *Ledger { return &Ledger{} }

func (l *Ledger) Record(_ context.Context, event domain.UsageEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	l.events = append(l.events, event)
	return nil
}

func (l *Ledger) Summary(_ context.Context, tenantID string, from, to time.Time) (domain.Summary, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var summary domain.Summary
	for _, event := range l.events {
		if event.TenantID != tenantID || event.OccurredAt.Before(from) || !event.OccurredAt.Before(to) {
			continue
		}
		summary.Requests++
		summary.InputTokens += event.InputTokens
		summary.OutputTokens += event.OutputTokens
		summary.CostUSD += event.CostUSD
	}
	return summary, nil
}
