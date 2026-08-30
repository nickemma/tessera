package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nickemma/tessera/internal/modules/tenancy/application"
	"github.com/nickemma/tessera/internal/modules/tenancy/domain"
)

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, requestID)))
	})
}

type requestIDKey struct{}

type tenantContextKey struct{}

func withTenant(ctx context.Context, tenant *domain.Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenant)
}

func tenantFrom(ctx context.Context) (*domain.Tenant, bool) {
	tenant, ok := ctx.Value(tenantContextKey{}).(*domain.Tenant)
	return tenant, ok
}

func Authenticate(service *application.Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || rawKey == "" {
				writeError(w, http.StatusUnauthorized, "unauthenticated", "missing api key")
				return
			}

			tenant, err := service.Authenticate(r.Context(), rawKey)
			if err != nil {
				if !errors.Is(err, domain.ErrKeyNotFound) && !errors.Is(err, domain.ErrKeyRevoked) && !errors.Is(err, domain.ErrTenantSuspended) {
					logger.Error("authentication failed", "err", err)
					writeError(w, http.StatusServiceUnavailable, "auth_unavailable", "authentication service unavailable")
					return
				}
				writeError(w, http.StatusUnauthorized, "unauthenticated", "invalid api key")
				return
			}

			next.ServeHTTP(w, r.WithContext(withTenant(r.Context(), tenant)))
		})
	}
}
