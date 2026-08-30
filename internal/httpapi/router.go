package httpapi

import "net/http"

func maxBytes(n int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, n)
		next.ServeHTTP(w, r)
	})
}

func NewRouter(chatHandler *ChatHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat", chatHandler.Complete)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return maxBytes(1<<20, mux) // 1 MiB
}
