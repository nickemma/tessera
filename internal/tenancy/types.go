package tenancy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrKeyNotFound     = errors.New("api key not found")
	ErrKeyRevoked      = errors.New("api key revoked")
	ErrTenantSuspended = errors.New("tenant suspended")
)

type Tenant struct {
	ID     string
	Name   string
	Status string
}

type APIKey struct {
	ID        string
	TenantID  string
	Prefix    string
	Label     string
	RevokedAt *time.Time
}

// GenerateKey returns the raw key (shown once, never stored) and its hash.
func GenerateKey() (raw string, hash []byte, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, "", err
	}
	raw = "tsk_live_" + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], raw[:17], nil
}

func HashKey(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
