package canned

import (
	"context"
	"fmt"
	"strings"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Complete(ctx context.Context, request domain.Request, emit func(string) error) (domain.Response, error) {
	if err := ctx.Err(); err != nil {
		return domain.Response{}, err
	}

	text := fmt.Sprintf("(canned) received %d characters", len(request.Prompt))
	if emit != nil {
		if err := emit(text); err != nil {
			return domain.Response{Text: text, Model: modelName(request.Model), InputTokens: estimateTokens(request.Prompt), OutputTokens: estimateTokens(text)}, err
		}
	}
	return domain.Response{
		Text:         text,
		Model:        modelName(request.Model),
		InputTokens:  estimateTokens(request.Prompt),
		OutputTokens: estimateTokens(text),
	}, nil
}

func modelName(requested string) string {
	if requested == "" {
		return "canned-local"
	}
	return requested
}

// This deliberately crude estimate is replaced by the model tokenizer when
// the real inference provider is introduced.
func estimateTokens(value string) int {
	return len(strings.Fields(value))
}
