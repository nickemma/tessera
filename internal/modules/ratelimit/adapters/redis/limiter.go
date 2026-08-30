package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/nickemma/tessera/internal/modules/ratelimit/domain"
	redisclient "github.com/redis/go-redis/v9"
)

type Limiter struct {
	client redisclient.UniversalClient
	rate   int
	burst  int
}

func NewLimiter(client redisclient.UniversalClient, ratePerSecond, burst int) *Limiter {
	return &Limiter{client: client, rate: ratePerSecond, burst: burst}
}

func (l *Limiter) Allow(ctx context.Context, tenantID string) error {
	key := "tessera:ratelimit:" + tenantID
	now := time.Now().UnixMilli()
	allowed, err := tokenScript.Run(ctx, l.client, []string{key}, l.rate, l.burst, now).Int()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	if allowed == 0 {
		return domain.ErrLimited
	}
	return nil
}

var tokenScript = redisclient.NewScript(`
local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens'))
local last = tonumber(redis.call('HGET', KEYS[1], 'last'))
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
if not tokens then tokens = burst end
if not last then last = now end
tokens = math.min(burst, tokens + ((now - last) / 1000.0) * rate)
local allowed = 0
if tokens >= 1 then tokens = tokens - 1; allowed = 1 end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'last', now)
redis.call('EXPIRE', KEYS[1], 3600)
return allowed
`)
