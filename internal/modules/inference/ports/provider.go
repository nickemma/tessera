package ports

import (
	"context"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
)

type Provider interface {
	Complete(ctx context.Context, request domain.Request, emit func(string) error) (domain.Response, error)
}
