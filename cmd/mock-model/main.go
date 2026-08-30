package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type request struct {
	Model    string `json:"model"`
	Messages []struct {
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	http.HandleFunc("/v1/chat/completions", complete)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		panic(err)
	}
}

func complete(w http.ResponseWriter, r *http.Request) {
	var input request
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prompt := ""
	if len(input.Messages) > 0 {
		prompt = input.Messages[len(input.Messages)-1].Content
	}
	text := fmt.Sprintf("(mock model) received %d characters", len(prompt))
	model := input.Model
	if model == "" {
		model = "mock-local"
	}
	if !input.Stream {
		writeJSON(w, map[string]any{"id": "mock-1", "object": "chat.completion", "created": time.Now().Unix(), "model": model, "choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": text}, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": len(strings.Fields(prompt)), "completion_tokens": len(strings.Fields(text))}})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	chunk := map[string]any{"id": "mock-1", "object": "chat.completion.chunk", "created": time.Now().Unix(), "model": model, "choices": []any{map[string]any{"delta": map[string]string{"role": "assistant", "content": text}, "finish_reason": nil}}}
	writeSSE(w, chunk)
	if flusher != nil {
		flusher.Flush()
	}
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeSSE(w http.ResponseWriter, value any) {
	data, _ := json.Marshal(value)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}
