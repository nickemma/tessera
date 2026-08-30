package httpapi

import (
	"context"

	"github.com/nickemma/tessera/internal/tenancy"
)

type ctxKey struct{}

var tenantKey ctxKey

func withTenant(ctx context.Context, t *tenancy.Tenant) context.Context {
	return context.WithValue(ctx, tenantKey, t)
}

func TenantFrom(ctx context.Context) (*tenancy.Tenant, bool) {
	t, ok := ctx.Value(tenantKey).(*tenancy.Tenant)
	return t, ok
}
