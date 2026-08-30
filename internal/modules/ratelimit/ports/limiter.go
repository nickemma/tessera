package ports

import "context"

type Limiter interface {
	Allow(ctx context.Context, tenantID string) error
}
