package ports

import (
	"context"

	"github.com/nickemma/tessera/internal/modules/budget/domain"
)

type Store interface {
	Reserve(ctx context.Context, tenantID string, tokens int) (domain.Reservation, int, error)
	Reconcile(ctx context.Context, reservation domain.Reservation, actualTokens int) error
	Release(ctx context.Context, reservation domain.Reservation) error
	Remaining(ctx context.Context, tenantID string) (int, error)
}
