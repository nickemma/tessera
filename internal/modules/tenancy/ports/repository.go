package ports

import (
	"context"

	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
)

type Repository interface {
	FindTenantByKeyHash(ctx context.Context, hash []byte) (*domain.Tenant, *domain.APIKey, error)
	CreateTenant(ctx context.Context, name string) (*domain.Tenant, error)
	CreateKey(ctx context.Context, tenantID, label string, hash []byte, prefix string) (*domain.APIKey, error)
	RevokeKey(ctx context.Context, keyID string) error
}
