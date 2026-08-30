package domain

import "errors"

var (
	ErrExceeded  = errors.New("budget exceeded")
	ErrAvailable = errors.New("budget store unavailable")
)

type Reservation struct {
	ID       string
	TenantID string
	Tokens   int
}
