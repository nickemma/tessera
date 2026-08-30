package httpapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nickemma/tessera/internal/app"
)

func TestChatComplete(t *testing.T) {
	server := newServer(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"prompt":"hello tessera"}`))
	request.Header.Set("Authorization", "Bearer "+server.demoKey)
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("want status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"response":"(canned) received 13 characters"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestChatRejectsInvalidRequest(t *testing.T) {
	server := newServer(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"prompt":""}`))
	request.Header.Set("Authorization", "Bearer "+server.demoKey)
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want status %d, got %d", http.StatusUnprocessableEntity, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	server := newServer(t)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("want status %d, got %d", http.StatusOK, response.Code)
	}
}

type testServer struct {
	handler http.Handler
	demoKey string
}

func newServer(t *testing.T) testServer {
	t.Helper()
	application, err := app.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return testServer{handler: application.Handler, demoKey: application.DemoKey}
}
