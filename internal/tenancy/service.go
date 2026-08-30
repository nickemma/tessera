package tenancy

import "context"

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*Tenant, error) {
	tenant, key, err := s.repo.FindTenantByKeyHash(ctx, HashKey(rawKey))
	if err != nil {
		return nil, err
	}
	if key.RevokedAt != nil {
		return nil, ErrKeyRevoked
	}
	if tenant.Status != "active" {
		return nil, ErrTenantSuspended
	}
	return tenant, nil
}
