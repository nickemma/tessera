package app_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nickemma/tessera/internal/app"
)

func TestEndToEndPlaygroundRequest(t *testing.T) {
	application, err := app.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"canned-local","messages":[{"role":"user","content":"hello"}]}`))
	request.Header.Set("Authorization", "Bearer "+application.DemoKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	application.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("want status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"chat.completion"`) {
		t.Fatalf("unexpected completion: %s", response.Body.String())
	}

	usageRequest := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	usageRequest.Header.Set("Authorization", "Bearer "+application.DemoKey)
	usageResponse := httptest.NewRecorder()
	application.Handler.ServeHTTP(usageResponse, usageRequest)
	if usageResponse.Code != http.StatusOK || !strings.Contains(usageResponse.Body.String(), `"requests":1`) {
		t.Fatalf("unexpected usage response: %d %s", usageResponse.Code, usageResponse.Body.String())
	}
}

func TestEndToEndBudgetRejectsBeforeInference(t *testing.T) {
	application, err := app.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"too much"}],"max_tokens":6000}`))
	request.Header.Set("Authorization", "Bearer "+application.DemoKey)
	response := httptest.NewRecorder()
	application.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("want status %d, got %d: %s", http.StatusTooManyRequests, response.Code, response.Body.String())
	}
}

func TestDocumentationEndpoints(t *testing.T) {
	application, err := app.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	openAPIRequest := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIResponse := httptest.NewRecorder()
	application.Handler.ServeHTTP(openAPIResponse, openAPIRequest)
	if openAPIResponse.Code != http.StatusOK {
		t.Fatalf("want OpenAPI status %d, got %d", http.StatusOK, openAPIResponse.Code)
	}
	var specification map[string]any
	if err := json.Unmarshal(openAPIResponse.Body.Bytes(), &specification); err != nil {
		t.Fatalf("OpenAPI response is not JSON: %v", err)
	}
	if specification["openapi"] != "3.0.3" {
		t.Fatalf("unexpected OpenAPI version: %v", specification["openapi"])
	}

	docsRequest := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsResponse := httptest.NewRecorder()
	application.Handler.ServeHTTP(docsResponse, docsRequest)
	if docsResponse.Code != http.StatusOK || !strings.Contains(docsResponse.Body.String(), "Tessera Playground") {
		t.Fatalf("unexpected docs response: %d %s", docsResponse.Code, docsResponse.Body.String())
	}
}
