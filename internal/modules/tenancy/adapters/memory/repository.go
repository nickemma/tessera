package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
)

type Repository struct {
	mu       sync.RWMutex
	sequence atomic.Uint64
	tenants  map[string]domain.Tenant
	keys     map[string]domain.APIKey
	byHash   map[string]string
}

func NewRepository() *Repository {
	return &Repository{
		tenants: make(map[string]domain.Tenant),
		keys:    make(map[string]domain.APIKey),
		byHash:  make(map[string]string),
	}
}

func (r *Repository) nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, r.sequence.Add(1))
}

func (r *Repository) FindTenantByKeyHash(_ context.Context, hash []byte) (*domain.Tenant, *domain.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keyID, ok := r.byHash[string(hash)]
	if !ok {
		return nil, nil, domain.ErrKeyNotFound
	}
	key, ok := r.keys[keyID]
	if !ok {
		return nil, nil, domain.ErrKeyNotFound
	}
	tenant, ok := r.tenants[key.TenantID]
	if !ok {
		return nil, nil, errors.New("tenant not found")
	}
	return copyTenant(tenant), copyKey(key), nil
}

func (r *Repository) CreateTenant(_ context.Context, name string) (*domain.Tenant, error) {
	if name == "" {
		return nil, errors.New("tenant name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	tenant := domain.Tenant{ID: r.nextID("tenant"), Name: name, Status: "active"}
	r.tenants[tenant.ID] = tenant
	return copyTenant(tenant), nil
}

func (r *Repository) CreateKey(_ context.Context, tenantID, label string, hash []byte, prefix string) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tenants[tenantID]; !ok {
		return nil, errors.New("tenant not found")
	}
	key := domain.APIKey{ID: r.nextID("key"), TenantID: tenantID, Prefix: prefix, Label: label}
	r.keys[key.ID] = key
	r.byHash[string(hash)] = key.ID
	return copyKey(key), nil
}

func (r *Repository) RevokeKey(_ context.Context, keyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, ok := r.keys[keyID]
	if !ok {
		return domain.ErrKeyNotFound
	}
	now := time.Now().UTC()
	key.RevokedAt = &now
	r.keys[keyID] = key
	return nil
}

func copyTenant(value domain.Tenant) *domain.Tenant { return &value }

func copyKey(value domain.APIKey) *domain.APIKey {
	if value.RevokedAt != nil {
		revoked := *value.RevokedAt
		value.RevokedAt = &revoked
	}
	return &value
}
