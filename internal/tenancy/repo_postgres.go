package tenancy

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
)

func (r *PostgresRepo) FindTenantByKeyHash(ctx context.Context, hash []byte) (*Tenant, *APIKey, error) {
	const q = `
		select t.id, t.name, t.status,
		       k.id, k.tenant_id, k.prefix, k.label, k.revoked_at
		from api_keys k
		join tenants t on t.id = k.tenant_id
		where k.key_hash = $1`

	var t Tenant
	var k APIKey
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&t.ID, &t.Name, &t.Status,
		&k.ID, &k.TenantID, &k.Prefix, &k.Label, &k.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrKeyNotFound
	}
	return &t, &k, err
}
