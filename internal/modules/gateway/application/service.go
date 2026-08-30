package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	budgetdomain "github.com/nickemma/tessera/internal/modules/budget/domain"
	budgetports "github.com/nickemma/tessera/internal/modules/budget/ports"
	gatewaydomain "github.com/nickemma/tessera/internal/modules/gateway/domain"
	inferenceapp "github.com/nickemma/tessera/internal/modules/inference/application"
	inferencedomain "github.com/nickemma/tessera/internal/modules/inference/domain"
	meteringdomain "github.com/nickemma/tessera/internal/modules/metering/domain"
	meteringports "github.com/nickemma/tessera/internal/modules/metering/ports"
	ratelimitdomain "github.com/nickemma/tessera/internal/modules/ratelimit/domain"
	ratelimitports "github.com/nickemma/tessera/internal/modules/ratelimit/ports"
	"github.com/nickemma/tessera/internal/platform/metrics"
)

type Service struct {
	inference      *inferenceapp.Service
	budget         budgetports.Store
	limiter        ratelimitports.Limiter
	ledger         meteringports.Ledger
	semaphore      chan struct{}
	metrics        *metrics.Registry
	costPerMillion float64
}

type Result struct {
	Response  inferencedomain.Response
	Remaining int
}

func NewService(inference *inferenceapp.Service, budget budgetports.Store, limiter ratelimitports.Limiter, ledger meteringports.Ledger, maxConcurrent int, costPerMillion float64, registry *metrics.Registry) *Service {
	return &Service{inference: inference, budget: budget, limiter: limiter, ledger: ledger, semaphore: make(chan struct{}, maxConcurrent), costPerMillion: costPerMillion, metrics: registry}
}

func (s *Service) Complete(ctx context.Context, tenantID string, request inferencedomain.Request, emit func(string) error) (Result, error) {
	startedAt := time.Now()
	if s.metrics != nil {
		defer func() { s.metrics.RequestLatency(time.Since(startedAt).Seconds()) }()
	}
	if s.metrics != nil {
		s.metrics.Request()
	}
	firstToken := false
	if emit != nil {
		wrappedEmit := emit
		emit = func(chunk string) error {
			if !firstToken && s.metrics != nil {
				firstToken = true
				s.metrics.TTFT(time.Since(startedAt).Seconds())
			}
			return wrappedEmit(chunk)
		}
	}
	if err := s.limiter.Allow(ctx, tenantID); err != nil {
		if s.metrics != nil && errors.Is(err, ratelimitdomain.ErrLimited) {
			s.metrics.RateLimited()
		}
		return Result{}, err
	}
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
		if s.metrics != nil {
			s.metrics.Saturated()
		}
		return Result{}, gatewaydomain.ErrSaturated
	}

	reservation, remaining, err := s.budget.Reserve(ctx, tenantID, request.ReservationTokens())
	if err != nil {
		if s.metrics != nil && errors.Is(err, budgetdomain.ErrExceeded) {
			s.metrics.BudgetRejected()
		}
		return Result{}, err
	}

	response, err := s.inference.Complete(ctx, request, emit)
	if err != nil {
		actualTokens := response.InputTokens + response.OutputTokens
		if actualTokens > 0 {
			_ = s.budget.Reconcile(ctx, reservation, actualTokens)
		} else {
			_ = s.budget.Release(ctx, reservation)
		}
		_ = s.ledger.Record(ctx, meteringdomain.UsageEvent{
			ID: fmt.Sprintf("usage-%d", time.Now().UnixNano()), TenantID: tenantID, Model: response.Model,
			InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, Status: "failed", OccurredAt: time.Now().UTC(),
		})
		return Result{Response: response}, err
	}

	actualTokens := response.InputTokens + response.OutputTokens
	if err := s.budget.Reconcile(ctx, reservation, actualTokens); err != nil {
		return Result{}, fmt.Errorf("reconcile budget: %w", err)
	}
	if remaining, err = s.budget.Remaining(ctx, tenantID); err != nil {
		return Result{}, err
	}

	if err := s.ledger.Record(ctx, meteringdomain.UsageEvent{
		ID:           fmt.Sprintf("usage-%d", time.Now().UnixNano()),
		TenantID:     tenantID,
		Model:        response.Model,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		CostUSD:      float64(actualTokens) / 1000000 * s.costPerMillion,
		Status:       "succeeded",
		OccurredAt:   time.Now().UTC(),
	}); err != nil {
		return Result{}, fmt.Errorf("record usage: %w", err)
	}
	if s.metrics != nil {
		s.metrics.Completed()
		s.metrics.Tokens(response.InputTokens, response.OutputTokens)
	}

	return Result{Response: response, Remaining: remaining}, nil
}

func (s *Service) Usage(ctx context.Context, tenantID string, from, to time.Time) (meteringdomain.Summary, error) {
	return s.ledger.Summary(ctx, tenantID, from, to)
}

func IsBudgetExceeded(err error) bool { return errors.Is(err, budgetdomain.ErrExceeded) }
