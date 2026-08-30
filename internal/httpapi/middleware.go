func Authenticate(svc *tenancy.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || raw == "" {
				writeError(w, http.StatusUnauthorized, "unauthenticated", "missing api key")
				return
			}

			tenant, err := svc.Authenticate(r.Context(), raw)
			if err != nil {
				slog.Warn("auth failed", "err", err, "prefix", safePrefix(raw))
				writeError(w, http.StatusUnauthorized, "unauthenticated", "invalid api key")
				return
			}

			next.ServeHTTP(w, r.WithContext(withTenant(r.Context(), tenant)))
		})
	}
}

func safePrefix(raw string) string {
	if len(raw) < 17 {
		return "invalid"
	}
	return raw[:17]
}
