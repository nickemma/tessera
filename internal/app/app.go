package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	budgetmemory "github.com/nickemma/tessera/internal/modules/budget/adapters/memory"
	budgetredis "github.com/nickemma/tessera/internal/modules/budget/adapters/redis"
	budgetports "github.com/nickemma/tessera/internal/modules/budget/ports"
	gatewayapp "github.com/nickemma/tessera/internal/modules/gateway/application"
	inferenceapp "github.com/nickemma/tessera/internal/modules/inference/application"
	inferenceports "github.com/nickemma/tessera/internal/modules/inference/ports"
	meteringmemory "github.com/nickemma/tessera/internal/modules/metering/adapters/memory"
	meteringpostgres "github.com/nickemma/tessera/internal/modules/metering/adapters/postgres"
	meteringapp "github.com/nickemma/tessera/internal/modules/metering/application"
	meteringports "github.com/nickemma/tessera/internal/modules/metering/ports"
	ratelimitmemory "github.com/nickemma/tessera/internal/modules/ratelimit/adapters/memory"
	ratelimitredis "github.com/nickemma/tessera/internal/modules/ratelimit/adapters/redis"
	ratelimitports "github.com/nickemma/tessera/internal/modules/ratelimit/ports"
	tenancymemory "github.com/nickemma/tessera/internal/modules/tenancy/adapters/memory"
	tenancypostgres "github.com/nickemma/tessera/internal/modules/tenancy/adapters/postgres"
	tenancyapp "github.com/nickemma/tessera/internal/modules/tenancy/application"
	tenancyports "github.com/nickemma/tessera/internal/modules/tenancy/ports"
	"github.com/nickemma/tessera/internal/platform/config"
	platformmetrics "github.com/nickemma/tessera/internal/platform/metrics"
	"github.com/nickemma/tessera/internal/providers/canned"
	"github.com/nickemma/tessera/internal/providers/openai"
	"github.com/nickemma/tessera/internal/providers/resilient"
	"github.com/nickemma/tessera/internal/transport/httpapi"
	redisclient "github.com/redis/go-redis/v9"
)

// App is the composition root's public runtime surface.
type App struct {
	Handler http.Handler
	DemoKey string
	close   func()
}

func New(logger *slog.Logger) (*App, error) {
	return NewWithConfig(config.Load(), logger)
}

func NewWithConfig(cfg config.Config, logger *slog.Logger) (*App, error) {
	ctx := context.Background()
	var closeResources []func()
	registry := platformmetrics.New()

	var tenancyRepository tenancyports.Repository
	var ledger meteringports.Ledger
	if cfg.Storage == "postgres" || cfg.DatabaseURL != "" {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("create postgres pool: %w", err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("connect to postgres: %w", err)
		}
		closeResources = append(closeResources, pool.Close)
		tenancyRepository = tenancypostgres.NewRepository(pool)
		persistentLedger := meteringpostgres.NewLedger(pool)
		asyncLedger := meteringapp.NewAsyncLedger(persistentLedger, 1000)
		ledger = asyncLedger
		closeResources = append(closeResources, asyncLedger.Close)
	} else {
		tenancyRepository = tenancymemory.NewRepository()
		ledger = meteringmemory.NewLedger()
	}

	tenancy := tenancyapp.NewService(tenancyRepository)
	tenant, err := tenancy.CreateTenant(ctx, "playground")
	if err != nil {
		return nil, fmt.Errorf("create playground tenant: %w", err)
	}
	demoKey, _, err := tenancy.CreateKey(ctx, tenant.ID, "playground")
	if err != nil {
		return nil, fmt.Errorf("create playground key: %w", err)
	}

	var budget budgetports.Store
	var limiter ratelimitports.Limiter
	if cfg.RedisURL != "" {
		options, err := redisclient.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("parse redis URL: %w", err)
		}
		client := redisclient.NewClient(options)
		if err := client.Ping(ctx).Err(); err != nil {
			client.Close()
			return nil, fmt.Errorf("connect to redis: %w", err)
		}
		closeResources = append(closeResources, func() { _ = client.Close() })
		redisBudget := budgetredis.NewStore(client)
		if err := redisBudget.SetBudget(ctx, tenant.ID, cfg.TenantBudget); err != nil {
			return nil, fmt.Errorf("initialize budget: %w", err)
		}
		budget = redisBudget
		limiter = ratelimitredis.NewLimiter(client, cfg.RateLimitPerSec, cfg.RateLimitBurst)
	} else {
		memoryBudget := budgetmemory.NewStore()
		memoryBudget.SetBudget(tenant.ID, cfg.TenantBudget)
		budget = memoryBudget
		limiter = ratelimitmemory.NewLimiter(cfg.RateLimitPerSec, cfg.RateLimitBurst)
	}

	var provider inferenceports.Provider
	if cfg.ModelURL != "" {
		provider = resilient.New(openai.New(cfg.ModelURL, cfg.ModelName, time.Duration(cfg.RequestTimeoutSec)*time.Second), 3, 5*time.Second)
	} else {
		provider = resilient.New(canned.New(), 3, 5*time.Second)
	}
	inference := inferenceapp.NewService(provider)
	gateway := gatewayapp.NewService(inference, budget, limiter, ledger, cfg.MaxConcurrent, cfg.CostPerMillion, registry)

	return &App{
		Handler: httpapi.NewRouter(httpapi.Dependencies{Gateway: gateway, Tenancy: tenancy, DemoKey: demoKey, Metrics: registry}, logger),
		DemoKey: demoKey,
		close: func() {
			for i := len(closeResources) - 1; i >= 0; i-- {
				closeResources[i]()
			}
		},
	}, nil
}

func (a *App) Close() {
	if a.close != nil {
		a.close()
	}
}
