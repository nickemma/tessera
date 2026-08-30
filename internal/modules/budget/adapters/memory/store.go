package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nickemma/tessera/internal/modules/budget/domain"
)

type Store struct {
	mu           sync.Mutex
	sequence     atomic.Uint64
	remaining    map[string]int
	reservations map[string]domain.Reservation
}

func NewStore() *Store {
	return &Store{remaining: make(map[string]int), reservations: make(map[string]domain.Reservation)}
}

func (s *Store) SetBudget(tenantID string, tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remaining[tenantID] = tokens
}

func (s *Store) Reserve(_ context.Context, tenantID string, tokens int) (domain.Reservation, int, error) {
	if tokens <= 0 {
		return domain.Reservation{}, 0, domain.ErrExceeded
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	available, ok := s.remaining[tenantID]
	if !ok {
		return domain.Reservation{}, 0, domain.ErrExceeded
	}
	if available < tokens {
		return domain.Reservation{}, available, domain.ErrExceeded
	}
	reservation := domain.Reservation{
		ID:       fmt.Sprintf("reservation-%d", s.sequence.Add(1)),
		TenantID: tenantID,
		Tokens:   tokens,
	}
	s.remaining[tenantID] = available - tokens
	s.reservations[reservation.ID] = reservation
	return reservation, s.remaining[tenantID], nil
}

func (s *Store) Reconcile(_ context.Context, reservation domain.Reservation, actualTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	reserved, ok := s.reservations[reservation.ID]
	if !ok {
		return nil
	}
	if actualTokens < 0 {
		actualTokens = 0
	}
	if actualTokens < reserved.Tokens {
		s.remaining[reserved.TenantID] += reserved.Tokens - actualTokens
	}
	delete(s.reservations, reservation.ID)
	return nil
}

func (s *Store) Release(_ context.Context, reservation domain.Reservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	reserved, ok := s.reservations[reservation.ID]
	if !ok {
		return nil
	}
	s.remaining[reserved.TenantID] += reserved.Tokens
	delete(s.reservations, reservation.ID)
	return nil
}

func (s *Store) Remaining(_ context.Context, tenantID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.remaining[tenantID], nil
}
