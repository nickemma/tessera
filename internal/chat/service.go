package chat

import (
	"context"
	"fmt"
	"strings"
)

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Complete(ctx context.Context, req Request) Response {
	out := fmt.Sprintf("(canned) received %d characters", len(req.Prompt))
	return Response{
		Response:     out,
		InputTokens:  estimateTokens(req.Prompt),
		OutputTokens: estimateTokens(out),
	}
}

// crude on purpose — a real tokenizer arrives in M4
func estimateTokens(s string) int {
	return len(strings.Fields(s))
}
