package openai_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
	"github.com/nickemma/tessera/internal/providers/openai"
)

func TestProviderReadsStreamingOpenAIResponse(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/chat/completions" {
			return nil, fmt.Errorf("unexpected path: %s", request.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("data: {\"model\":\"mock\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"model\":\"mock\",\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\ndata: [DONE]\n\n")),
		}, nil
	})
	provider := openai.NewWithClient("http://model", "mock", &http.Client{Transport: transport, Timeout: time.Second})
	var chunks []string
	response, err := provider.Complete(context.Background(), domain.Request{Prompt: "test"}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "hello world" || strings.Join(chunks, "") != "hello world" {
		t.Fatalf("unexpected response: %+v chunks=%v", response, chunks)
	}
}

func TestProviderPropagatesClientCancellationWhileStreaming(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       &contextBody{ctx: request.Context()},
		}, nil
	})
	provider := openai.NewWithClient("http://model", "mock", &http.Client{Transport: transport})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := provider.Complete(ctx, domain.Request{Prompt: "test"}, func(string) error { return nil })
	if err == nil {
		t.Fatal("expected streaming request to stop after context cancellation")
	}
}

type contextBody struct{ ctx context.Context }

func (b *contextBody) Read([]byte) (int, error) {
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *contextBody) Close() error { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
