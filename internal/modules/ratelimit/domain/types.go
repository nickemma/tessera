package domain

import "errors"

var (
	ErrLimited     = errors.New("rate limit exceeded")
	ErrUnavailable = errors.New("rate limiter unavailable")
)
