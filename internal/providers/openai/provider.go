package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
)

type Provider struct {
	client  *http.Client
	baseURL string
	name    string
}

func New(baseURL, name string, timeout time.Duration) *Provider {
	return NewWithClient(baseURL, name, &http.Client{Timeout: timeout})
}

func NewWithClient(baseURL, name string, client *http.Client) *Provider {
	return &Provider{client: client, baseURL: strings.TrimRight(baseURL, "/"), name: name}
}

func (p *Provider) Complete(ctx context.Context, request domain.Request, emit func(string) error) (domain.Response, error) {
	upstream := openAIRequest{Model: request.Model, Messages: []message{{Role: "user", Content: request.Prompt}}, MaxTokens: request.MaxTokens, Stream: emit != nil}
	body, err := json.Marshal(upstream)
	if err != nil {
		return domain.Response{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return domain.Response{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(httpRequest)
	if err != nil {
		return domain.Response{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return domain.Response{}, fmt.Errorf("model returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	if emit != nil {
		return p.readStream(response.Body, request, emit)
	}
	var result openAIResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return domain.Response{}, err
	}
	if len(result.Choices) == 0 {
		return domain.Response{}, fmt.Errorf("model returned no choices")
	}
	text := result.Choices[0].Message.Content
	return domain.Response{Text: text, Model: modelName(result.Model, p.name), InputTokens: result.Usage.PromptTokens, OutputTokens: result.Usage.CompletionTokens}, nil
}

func (p *Provider) readStream(body io.Reader, request domain.Request, emit func(string) error) (domain.Response, error) {
	var text strings.Builder
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk openAIChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return domain.Response{Text: text.String(), Model: modelName(request.Model, p.name), InputTokens: estimate(request.Prompt), OutputTokens: estimate(text.String())}, err
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}
		text.WriteString(content)
		if err := emit(content); err != nil {
			return domain.Response{Text: text.String(), Model: modelName(request.Model, p.name), InputTokens: estimate(request.Prompt), OutputTokens: estimate(text.String())}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return domain.Response{Text: text.String(), Model: modelName(request.Model, p.name), InputTokens: estimate(request.Prompt), OutputTokens: estimate(text.String())}, err
	}
	return domain.Response{Text: text.String(), Model: modelName(request.Model, p.name), InputTokens: estimate(request.Prompt), OutputTokens: estimate(text.String())}, nil
}

type openAIRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
	Stream    bool      `json:"stream,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openAIChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func modelName(requested, fallback string) string {
	if requested != "" {
		return requested
	}
	return fallback
}

func estimate(value string) int { return len(strings.Fields(value)) }
