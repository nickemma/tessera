package tenancy_test

import (
	"context"
	"errors"
	"testing"
)

func TestAuthenticate_RevokedKey(t *testing.T) {
	repo := NewMemoryRepo()
	tenant, _ := repo.CreateTenant(context.Background(), "acme")
	raw, hash, prefix, _ := GenerateKey()
	key, _ := repo.CreateKey(context.Background(), tenant.ID, "test", hash, prefix)
	_ = repo.RevokeKey(context.Background(), key.ID)

	_, err := NewService(repo).Authenticate(context.Background(), raw)
	if !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("want ErrKeyRevoked, got %v", err)
	}
}
