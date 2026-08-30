package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	gatewayapp "github.com/nickemma/tessera/internal/modules/gateway/application"
	tenancyapp "github.com/nickemma/tessera/internal/modules/tenancy/application"
	"github.com/nickemma/tessera/internal/platform/metrics"
)

type Dependencies struct {
	Gateway *gatewayapp.Service
	Tenancy *tenancyapp.Service
	DemoKey string
	Metrics *metrics.Registry
}

func NewRouter(dependencies Dependencies, logger *slog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(maxBytes(1 << 20))
	router.Use(requestID)

	handler := NewChatHandler(dependencies.Gateway, logger)
	router.Get("/openapi.json", openAPI)
	router.Get("/playground", playground(dependencies.DemoKey))
	router.Get("/docs", playground(dependencies.DemoKey))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/metrics", dependencies.Metrics.Handler)

	router.Group(func(protected chi.Router) {
		protected.Use(Authenticate(dependencies.Tenancy, logger))
		protected.Post("/v1/chat", handler.Complete)
		protected.Post("/v1/chat/completions", handler.OpenAIComplete)
		protected.Post("/v1/completions", handler.CompleteText)
		protected.Get("/v1/usage", handler.Usage)
	})

	return router
}

func maxBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
