package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nickemma/tessera/internal/modules/metering/domain"
	"github.com/nickemma/tessera/internal/modules/metering/ports"
)

var ErrBufferFull = errors.New("usage ledger buffer full")

type AsyncLedger struct {
	sink   ports.Ledger
	queue  chan domain.UsageEvent
	done   chan struct{}
	closed chan struct{}
	once   sync.Once
}

func NewAsyncLedger(sink ports.Ledger, buffer int) *AsyncLedger {
	ledger := &AsyncLedger{sink: sink, queue: make(chan domain.UsageEvent, buffer), done: make(chan struct{}), closed: make(chan struct{})}
	go ledger.flush()
	return ledger
}

func (l *AsyncLedger) Record(ctx context.Context, event domain.UsageEvent) error {
	select {
	case l.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBufferFull
	}
}

func (l *AsyncLedger) Summary(ctx context.Context, tenantID string, from, to time.Time) (domain.Summary, error) {
	return l.sink.Summary(ctx, tenantID, from, to)
}

func (l *AsyncLedger) Close() {
	l.once.Do(func() {
		close(l.done)
		<-l.closed
	})
}

func (l *AsyncLedger) flush() {
	defer close(l.closed)
	for {
		select {
		case event := <-l.queue:
			l.writeWithRetry(event)
		case <-l.done:
			for {
				select {
				case event := <-l.queue:
					l.writeWithRetry(event)
				default:
					return
				}
			}
		}
	}
}

func (l *AsyncLedger) writeWithRetry(event domain.UsageEvent) {
	delay := 10 * time.Millisecond
	for {
		if err := l.sink.Record(context.Background(), event); err == nil {
			return
		}
		time.Sleep(delay)
		if delay < time.Second {
			delay *= 2
		}
	}
}
