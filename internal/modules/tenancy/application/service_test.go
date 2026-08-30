package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nickemma/tessera/internal/modules/tenancy/adapters/memory"
	"github.com/nickemma/tessera/internal/modules/tenancy/application"
	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
)

func TestAuthenticateRejectsRevokedKey(t *testing.T) {
	repository := memory.NewRepository()
	service := application.NewService(repository)
	tenant, err := service.CreateTenant(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	rawKey, key, err := service.CreateKey(context.Background(), tenant.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.RevokeKey(context.Background(), key.ID); err != nil {
		t.Fatal(err)
	}

	_, err = service.Authenticate(context.Background(), rawKey)
	if !errors.Is(err, domain.ErrKeyRevoked) {
		t.Fatalf("want ErrKeyRevoked, got %v", err)
	}
}

func TestAuthenticateCannotCrossTenants(t *testing.T) {
	repository := memory.NewRepository()
	service := application.NewService(repository)
	first, _ := service.CreateTenant(context.Background(), "first")
	second, _ := service.CreateTenant(context.Background(), "second")
	firstKey, _, _ := service.CreateKey(context.Background(), first.ID, "first")

	authenticated, err := service.Authenticate(context.Background(), firstKey)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.ID == second.ID {
		t.Fatal("first tenant key authenticated as second tenant")
	}
}
