package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/nickemma/tessera/internal/chat"
)

type ChatHandler struct {
	svc *chat.Service
}

func NewChatHandler(svc *chat.Service) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) Complete(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req chat.Request
	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "empty_body", "request body is empty")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.svc.Complete(r.Context(), req))
}
