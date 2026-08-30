package tenancy

import (
	"context"
	"errors"
)

type FakeRepository struct {
	tenants map[string]*Tenant
	keys    map[string]*APIKey
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		tenants: make(map[string]*Tenant),
		keys:    make(map[string]*APIKey),
	}
}

func (r *FakeRepository) Add(tenant *Tenant, rawKey string) {
	hash := string(HashKey(rawKey))

	r.tenants[hash] = tenant
	r.keys[hash] = &APIKey{
		TenantID: tenant.ID,
	}
}

func (r *FakeRepository) FindTenantByKeyHash(
	_ context.Context,
	hash []byte,
) (*Tenant, *APIKey, error) {
	k := string(hash)

	key, ok := r.keys[k]
	if !ok {
		return nil, nil, errors.New("tenant key not found")
	}

	tenant, ok := r.tenants[k]
	if !ok {
		return nil, nil, errors.New("tenant not found")
	}

	return tenant, key, nil
}
