package domain

import "time"

type UsageEvent struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	CacheHit     bool      `json:"cache_hit"`
	Fallback     bool      `json:"fallback"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type Summary struct {
	Requests     int     `json:"requests"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}
