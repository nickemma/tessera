package domain

import (
	"errors"
	"strings"
)

type Request struct {
	Prompt    string `json:"prompt"`
	Model     string `json:"model,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

type Response struct {
	Text         string `json:"response"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

func (r Request) Validate() error {
	switch {
	case r.Prompt == "":
		return errors.New("prompt is required")
	case len(r.Prompt) > 8000:
		return errors.New("prompt exceeds 8000 characters")
	case r.MaxTokens < 0:
		return errors.New("max_tokens must not be negative")
	default:
		return nil
	}
}

func (r Request) EffectiveMaxTokens() int {
	if r.MaxTokens > 0 {
		return r.MaxTokens
	}
	return 128
}

func (r Request) ReservationTokens() int {
	return len(strings.Fields(r.Prompt)) + r.EffectiveMaxTokens()
}
