// Command tesserac is a small client for exercising a TESSERA gateway.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type requestBody struct {
	Model    string        `json:"model,omitempty"`
	Messages []userMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type userMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseBody struct {
	Choices []struct {
		Message userMessage `json:"message"`
	} `json:"choices"`
}

type streamChunk struct {
	Choices []struct {
		Delta userMessage `json:"delta"`
	} `json:"choices"`
}

func main() {
	url := flag.String("url", "http://localhost:8080/v1/chat/completions", "TESSERA chat completions URL")
	key := flag.String("key", os.Getenv("TESSERA_API_KEY"), "Bearer API key")
	model := flag.String("model", "canned-local", "model name")
	prompt := flag.String("prompt", "hello from tesserac", "user prompt")
	stream := flag.Bool("stream", false, "stream the response")
	flag.Parse()
	if *key == "" {
		fmt.Fprintln(os.Stderr, "-key is required or set TESSERA_API_KEY")
		os.Exit(2)
	}

	payload, err := json.Marshal(requestBody{Model: *model, Messages: []userMessage{{Role: "user", Content: *prompt}}, Stream: *stream})
	if err != nil {
		fail(err)
	}
	request, err := http.NewRequest(http.MethodPost, *url, strings.NewReader(string(payload)))
	if err != nil {
		fail(err)
	}
	request.Header.Set("Authorization", "Bearer "+*key)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		fail(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		fail(fmt.Errorf("gateway returned %s: %s", response.Status, strings.TrimSpace(string(message))))
	}

	if *stream {
		streamResponse(response.Body)
		return
	}
	var result responseBody
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		fail(err)
	}
	if len(result.Choices) == 0 {
		fail(fmt.Errorf("gateway returned no choices"))
	}
	fmt.Println(result.Choices[0].Message.Content)
}

func streamResponse(body io.Reader) {
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
		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			fail(err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		fail(err)
	}
	fmt.Println()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
