package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/nickemma/tessera/internal/modules/budget/adapters/memory"
	"github.com/nickemma/tessera/internal/modules/budget/domain"
)

func TestReserveIsAtomic(t *testing.T) {
	store := memory.NewStore()
	store.SetBudget("tenant-a", 10)

	var wait sync.WaitGroup
	results := make(chan error, 100)
	for range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, err := store.Reserve(context.Background(), "tenant-a", 1)
			results <- err
		}()
	}
	wait.Wait()
	close(results)

	allowed := 0
	for err := range results {
		if err == nil {
			allowed++
		} else if err != domain.ErrExceeded {
			t.Fatalf("unexpected reservation error: %v", err)
		}
	}
	if allowed != 10 {
		t.Fatalf("want 10 allowed reservations, got %d", allowed)
	}
}
