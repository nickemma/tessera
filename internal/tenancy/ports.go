package tenancy

import "context"

type Repository interface {
	FindTenantByKeyHash(ctx context.Context, hash []byte) (*Tenant, *APIKey, error)
	CreateTenant(ctx context.Context, name string) (*Tenant, error)
	CreateKey(ctx context.Context, tenantID, label string, hash []byte, prefix string) (*APIKey, error)
	RevokeKey(ctx context.Context, keyID string) error
}
