package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/nickemma/tessera/internal/modules/budget/domain"
	redisclient "github.com/redis/go-redis/v9"
)

type Store struct {
	client redisclient.UniversalClient
	prefix string
}

func NewStore(client redisclient.UniversalClient) *Store {
	return &Store{client: client, prefix: "tessera:budget:"}
}

func (s *Store) SetBudget(ctx context.Context, tenantID string, tokens int) error {
	return s.client.Set(ctx, s.key(tenantID), tokens, 0).Err()
}

func (s *Store) Reserve(ctx context.Context, tenantID string, tokens int) (domain.Reservation, int, error) {
	if tokens <= 0 {
		return domain.Reservation{}, 0, domain.ErrExceeded
	}
	reservationID := fmt.Sprintf("reservation-%d", time.Now().UnixNano())
	value, err := reserveScript.Run(ctx, s.client, []string{s.key(tenantID)}, tokens).Int64()
	if err != nil {
		return domain.Reservation{}, 0, fmt.Errorf("%w: %v", domain.ErrAvailable, err)
	}
	if value == -2 {
		return domain.Reservation{}, 0, fmt.Errorf("%w: budget not initialized", domain.ErrAvailable)
	}
	if value == -1 {
		remaining, _ := s.client.Get(ctx, s.key(tenantID)).Int()
		return domain.Reservation{}, remaining, domain.ErrExceeded
	}
	return domain.Reservation{ID: reservationID, TenantID: tenantID, Tokens: tokens}, int(value), nil
}

func (s *Store) Reconcile(ctx context.Context, reservation domain.Reservation, actualTokens int) error {
	if actualTokens < 0 {
		actualTokens = 0
	}
	refund := reservation.Tokens - actualTokens
	return s.settle(ctx, reservation, refund)
}

func (s *Store) Release(ctx context.Context, reservation domain.Reservation) error {
	return s.settle(ctx, reservation, reservation.Tokens)
}

func (s *Store) settle(ctx context.Context, reservation domain.Reservation, refund int) error {
	if refund <= 0 {
		return nil
	}
	marker := s.prefix + "settled:" + reservation.ID
	created, err := s.client.SetNX(ctx, marker, "1", 24*time.Hour).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAvailable, err)
	}
	if !created {
		return nil
	}
	if err := s.client.IncrBy(ctx, s.key(reservation.TenantID), int64(refund)).Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAvailable, err)
	}
	return nil
}

func (s *Store) Remaining(ctx context.Context, tenantID string) (int, error) {
	value, err := s.client.Get(ctx, s.key(tenantID)).Int()
	if err == redisclient.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domain.ErrAvailable, err)
	}
	return value, nil
}

func (s *Store) key(tenantID string) string { return s.prefix + tenantID }

var reserveScript = redisclient.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then return -2 end
if tonumber(current) < tonumber(ARGV[1]) then return -1 end
return redis.call('DECRBY', KEYS[1], ARGV[1])
`)
