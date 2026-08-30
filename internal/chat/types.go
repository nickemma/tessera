package chat

import "errors"

type Request struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

type Response struct {
	Response     string `json:"response"`
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
	}
	return nil
}
