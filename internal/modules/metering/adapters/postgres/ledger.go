package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickemma/tessera/internal/modules/metering/domain"
)

type Ledger struct{ pool *pgxpool.Pool }

func NewLedger(pool *pgxpool.Pool) *Ledger { return &Ledger{pool: pool} }

func (l *Ledger) Record(ctx context.Context, event domain.UsageEvent) error {
	_, err := l.pool.Exec(ctx, `
		insert into usage_events
		(id, tenant_id, model, input_tokens, output_tokens, cost_usd, cache_hit, fallback, status, occurred_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		on conflict (id) do nothing`, event.ID, event.TenantID, event.Model, event.InputTokens, event.OutputTokens, event.CostUSD, event.CacheHit, event.Fallback, event.Status, event.OccurredAt)
	return err
}

func (l *Ledger) Summary(ctx context.Context, tenantID string, from, to time.Time) (domain.Summary, error) {
	var summary domain.Summary
	err := l.pool.QueryRow(ctx, `
		select count(*), coalesce(sum(input_tokens),0), coalesce(sum(output_tokens),0), coalesce(sum(cost_usd),0)
		from usage_events where tenant_id=$1 and occurred_at >= $2 and occurred_at < $3`, tenantID, from, to).
		Scan(&summary.Requests, &summary.InputTokens, &summary.OutputTokens, &summary.CostUSD)
	return summary, err
}

func (l *Ledger) Ping(ctx context.Context) error {
	if err := l.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	return nil
}
