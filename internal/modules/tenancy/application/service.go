package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
	"github.com/nickemma/tessera/internal/modules/tenancy/ports"
)

type Service struct {
	repository ports.Repository
}

func NewService(repository ports.Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*domain.Tenant, error) {
	tenant, key, err := s.repository.FindTenantByKeyHash(ctx, domain.HashKey(rawKey))
	if err != nil {
		if !errors.Is(err, domain.ErrKeyNotFound) {
			return nil, fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
		}
		return nil, err
	}
	if key.RevokedAt != nil {
		return nil, domain.ErrKeyRevoked
	}
	if tenant.Status != "active" {
		return nil, domain.ErrTenantSuspended
	}
	return tenant, nil
}

func (s *Service) CreateTenant(ctx context.Context, name string) (*domain.Tenant, error) {
	return s.repository.CreateTenant(ctx, name)
}

func (s *Service) CreateKey(ctx context.Context, tenantID, label string) (string, *domain.APIKey, error) {
	raw, hash, prefix, err := domain.GenerateKey()
	if err != nil {
		return "", nil, err
	}
	key, err := s.repository.CreateKey(ctx, tenantID, label, hash, prefix)
	if err != nil {
		return "", nil, err
	}
	return raw, key, nil
}
