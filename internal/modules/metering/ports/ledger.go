package ports

import (
	"context"
	"time"

	"github.com/nickemma/tessera/internal/modules/metering/domain"
)

type Ledger interface {
	Record(ctx context.Context, event domain.UsageEvent) error
	Summary(ctx context.Context, tenantID string, from, to time.Time) (domain.Summary, error)
}
