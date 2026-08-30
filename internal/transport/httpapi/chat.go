package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	budgetdomain "github.com/nickemma/tessera/internal/modules/budget/domain"
	gatewayapp "github.com/nickemma/tessera/internal/modules/gateway/application"
	gatewaydomain "github.com/nickemma/tessera/internal/modules/gateway/domain"
	"github.com/nickemma/tessera/internal/modules/inference/domain"
	ratelimitdomain "github.com/nickemma/tessera/internal/modules/ratelimit/domain"
	"github.com/nickemma/tessera/internal/platform/apperrors"
	"github.com/nickemma/tessera/internal/providers/resilient"
)

type ChatHandler struct {
	service *gatewayapp.Service
	logger  *slog.Logger
}

func NewChatHandler(service *gatewayapp.Service, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{service: service, logger: logger}
}

func (h *ChatHandler) Complete(w http.ResponseWriter, r *http.Request) {
	var request domain.Request
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	tenant, ok := tenantFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "missing tenant context")
		return
	}

	result, err := h.service.Complete(r.Context(), tenant.ID, request, nil)
	if err != nil {
		writeServiceError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, result.Response)
}

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Stream    bool   `json:"stream,omitempty"`
}

func (h *ChatHandler) OpenAIComplete(w http.ResponseWriter, r *http.Request) {
	var request openAIRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	if len(request.Messages) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_request", "messages is required")
		return
	}
	tenant, ok := tenantFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "missing tenant context")
		return
	}

	chatRequest := domain.Request{
		Model:     request.Model,
		Prompt:    promptFromMessages(request.Messages),
		MaxTokens: request.MaxTokens,
	}
	if request.Stream {
		stream := newStreamWriter(w, request.Model)
		result, err := h.service.Complete(r.Context(), tenant.ID, chatRequest, stream.Emit)
		if err != nil {
			if !stream.started {
				writeServiceError(w, h.logger, err)
			}
			stream.SetUsage(result.Response)
			return
		}
		stream.Finish(result.Response)
		return
	}

	result, err := h.service.Complete(r.Context(), tenant.ID, chatRequest, nil)
	if err != nil {
		writeServiceError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, toOpenAIResponse(result.Response, result.Remaining))
}

func (h *ChatHandler) CompleteText(w http.ResponseWriter, r *http.Request) {
	var request completionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}
	tenant, ok := tenantFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "missing tenant context")
		return
	}
	chatRequest := domain.Request{Model: request.Model, Prompt: request.Prompt, MaxTokens: request.MaxTokens}
	if request.Stream {
		stream := newStreamWriter(w, request.Model)
		result, err := h.service.Complete(r.Context(), tenant.ID, chatRequest, stream.Emit)
		if err != nil {
			if !stream.started {
				writeServiceError(w, h.logger, err)
			}
			stream.SetUsage(result.Response)
			return
		}
		stream.Finish(result.Response)
		return
	}
	result, err := h.service.Complete(r.Context(), tenant.ID, chatRequest, nil)
	if err != nil {
		writeServiceError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, completionResponse(result.Response))
}

func (h *ChatHandler) Usage(w http.ResponseWriter, r *http.Request) {
	tenant, ok := tenantFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "missing tenant context")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	from, err := time.Parse("2006-01", month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_month", "month must use YYYY-MM")
		return
	}
	to := from.AddDate(0, 1, 0)
	summary, err := h.service.Usage(r.Context(), tenant.ID, from, to)
	if err != nil {
		writeServiceError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "empty_body", "request body is empty")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
}

func writeServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		writeError(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	if errors.Is(err, budgetdomain.ErrExceeded) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "token budget exceeded", "code": "budget_exceeded", "reset_at": time.Now().UTC().Add(time.Hour)})
		return
	}
	if errors.Is(err, budgetdomain.ErrAvailable) {
		writeError(w, http.StatusServiceUnavailable, "budget_unavailable", "budget service unavailable")
		return
	}
	if errors.Is(err, ratelimitdomain.ErrLimited) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "rate limit exceeded", "code": "rate_limited", "reset_at": time.Now().UTC().Add(time.Second)})
		return
	}
	if errors.Is(err, ratelimitdomain.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "rate_limiter_unavailable", "rate limiter unavailable")
		return
	}
	if errors.Is(err, gatewaydomain.ErrSaturated) {
		writeError(w, http.StatusServiceUnavailable, "gateway_saturated", "gateway is at capacity")
		return
	}
	if errors.Is(err, resilient.ErrOpen) {
		writeError(w, http.StatusServiceUnavailable, "model_unavailable", "model temporarily unavailable")
		return
	}
	logger.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func promptFromMessages(messages []openAIMessage) string {
	var prompt string
	for _, message := range messages {
		if prompt != "" {
			prompt += "\n"
		}
		prompt += message.Role + ": " + message.Content
	}
	return prompt
}

type openAICompletionResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []openAIChoice  `json:"choices"`
	Usage   openAIUsage     `json:"usage"`
	Tessera tesseraResponse `json:"tessera"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type completionResponseBody struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []completionChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

type completionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
}

type tesseraResponse struct {
	RemainingTokens int `json:"remaining_tokens"`
}

func toOpenAIResponse(response domain.Response, remaining int) openAICompletionResponse {
	return openAICompletionResponse{
		ID:      "chatcmpl-local",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   response.Model,
		Choices: []openAIChoice{{Index: 0, Message: openAIMessage{Role: "assistant", Content: response.Text}, FinishReason: "stop"}},
		Usage:   openAIUsage{PromptTokens: response.InputTokens, CompletionTokens: response.OutputTokens, TotalTokens: response.InputTokens + response.OutputTokens},
		Tessera: tesseraResponse{RemainingTokens: remaining},
	}
}

func completionResponse(response domain.Response) completionResponseBody {
	return completionResponseBody{ID: "cmpl-local", Object: "text_completion", Created: time.Now().Unix(), Model: response.Model, Choices: []completionChoice{{Text: response.Text, Index: 0, FinishReason: "stop"}}, Usage: openAIUsage{PromptTokens: response.InputTokens, CompletionTokens: response.OutputTokens, TotalTokens: response.InputTokens + response.OutputTokens}}
}

type streamWriter struct {
	w       http.ResponseWriter
	model   string
	started bool
	created int64
}

func newStreamWriter(w http.ResponseWriter, requestedModel string) *streamWriter {
	return &streamWriter{w: w, model: requestedModel, created: time.Now().Unix()}
}

func (s *streamWriter) Emit(text string) error {
	if !s.started {
		s.w.Header().Set("Content-Type", "text/event-stream")
		s.w.Header().Set("Cache-Control", "no-cache")
		s.w.Header().Set("Connection", "keep-alive")
		s.w.Header().Add("Trailer", "X-Tessera-Usage")
		s.w.WriteHeader(http.StatusOK)
		s.started = true
	}
	chunk := map[string]any{
		"id": "chatcmpl-local", "object": "chat.completion.chunk", "created": s.created, "model": s.model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]string{"role": "assistant", "content": text}, "finish_reason": nil}},
	}
	return s.write(chunk)
}

func (s *streamWriter) Finish(response domain.Response) {
	if !s.started {
		_ = s.Emit(response.Text)
	}
	if response.Model != "" {
		s.model = response.Model
	}
	_ = s.write(map[string]any{"id": "chatcmpl-local", "object": "chat.completion.chunk", "created": s.created, "model": s.model, "choices": []map[string]any{{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"}}})
	_, _ = io.WriteString(s.w, "data: [DONE]\n\n")
	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	s.SetUsage(response)
}

func (s *streamWriter) SetUsage(response domain.Response) {
	s.w.Header().Set("X-Tessera-Usage", string(mustJSON(openAIUsage{PromptTokens: response.InputTokens, CompletionTokens: response.OutputTokens, TotalTokens: response.InputTokens + response.OutputTokens})))
}

func (s *streamWriter) write(value any) error {
	_, err := io.WriteString(s.w, "data: "+string(mustJSON(value))+"\n\n")
	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return err
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
