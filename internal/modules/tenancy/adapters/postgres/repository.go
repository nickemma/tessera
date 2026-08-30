package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) FindTenantByKeyHash(ctx context.Context, hash []byte) (*domain.Tenant, *domain.APIKey, error) {
	var tenant domain.Tenant
	var key domain.APIKey
	err := r.pool.QueryRow(ctx, `
		select t.id, t.name, t.status, k.id, k.tenant_id, k.prefix, k.label, k.revoked_at
		from api_keys k join tenants t on t.id=k.tenant_id where k.key_hash=$1`, hash).
		Scan(&tenant.ID, &tenant.Name, &tenant.Status, &key.ID, &key.TenantID, &key.Prefix, &key.Label, &key.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrKeyNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	_, _ = r.pool.Exec(ctx, `update api_keys set last_used_at=now() where id=$1`, key.ID)
	return &tenant, &key, nil
}

func (r *Repository) CreateTenant(ctx context.Context, name string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, `insert into tenants(name) values($1) returning id, name, status`, name).
		Scan(&tenant.ID, &tenant.Name, &tenant.Status)
	return &tenant, err
}

func (r *Repository) CreateKey(ctx context.Context, tenantID, label string, hash []byte, prefix string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx, `
		insert into api_keys(tenant_id, prefix, key_hash, label) values($1,$2,$3,$4)
		returning id, tenant_id, prefix, label, revoked_at`, tenantID, prefix, hash, label).
		Scan(&key.ID, &key.TenantID, &key.Prefix, &key.Label, &key.RevokedAt)
	return &key, err
}

func (r *Repository) RevokeKey(ctx context.Context, keyID string) error {
	result, err := r.pool.Exec(ctx, `update api_keys set revoked_at=now() where id=$1 and revoked_at is null`, keyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", domain.ErrKeyNotFound, keyID)
	}
	return nil
}
