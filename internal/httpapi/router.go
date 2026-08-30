package httpapi

import "net/http"

func NewRouter(chatHandler *ChatHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat", chatHandler.Complete)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}
