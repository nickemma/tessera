package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/v1/chat/completions", "completion endpoint")
	key := flag.String("key", "", "Bearer API key")
	requests := flag.Int("requests", 100, "total requests")
	concurrency := flag.Int("concurrency", 10, "concurrent workers")
	stream := flag.Bool("stream", false, "request Server-Sent Events and measure time to first event")
	model := flag.String("model", "mock-local", "model name")
	prompt := flag.String("prompt", "benchmark request", "user prompt")
	flag.Parse()
	if *key == "" || *requests < 1 || *concurrency < 1 {
		panic("-key is required; -requests and -concurrency must be positive")
	}
	if *concurrency > *requests {
		*concurrency = *requests
	}

	client := &http.Client{Timeout: 2 * time.Minute}
	latencies := make(chan time.Duration, *requests)
	ttfts := make(chan time.Duration, *requests)
	var succeeded atomic.Int64
	var failed atomic.Int64
	jobs := make(chan struct{})
	var wait sync.WaitGroup
	for worker := 0; worker < *concurrency; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range jobs {
				start := time.Now()
				payload := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":%q}],"stream":%t}`, *model, *prompt, *stream)
				request, _ := http.NewRequest(http.MethodPost, *url, strings.NewReader(payload))
				request.Header.Set("Authorization", "Bearer "+*key)
				request.Header.Set("Content-Type", "application/json")
				response, err := client.Do(request)
				if err != nil {
					latencies <- time.Since(start)
					ttfts <- 0
					failed.Add(1)
					continue
				}
				firstEvent := time.Duration(0)
				if *stream {
					buffer := make([]byte, 1)
					for {
						read, readErr := response.Body.Read(buffer)
						if read > 0 {
							firstEvent = time.Since(start)
							break
						}
						if readErr != nil {
							break
						}
					}
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				latencies <- time.Since(start)
				ttfts <- firstEvent
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					succeeded.Add(1)
				} else {
					failed.Add(1)
				}
			}
		}()
	}
	for i := 0; i < *requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)
	wait.Wait()
	close(latencies)
	close(ttfts)

	values := make([]time.Duration, 0, len(latencies))
	for latency := range latencies {
		values = append(values, latency)
	}
	if len(values) == 0 {
		fmt.Println("no completed requests")
		return
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	percentile := func(percent int) time.Duration { return values[(len(values)-1)*percent/100] }
	message := fmt.Sprintf("requests=%d concurrency=%d stream=%t succeeded=%d failed=%d p50=%s p95=%s p99=%s", *requests, *concurrency, *stream, succeeded.Load(), failed.Load(), percentile(50), percentile(95), percentile(99))
	if *stream {
		firstValues := make([]time.Duration, 0, *requests)
		for value := range ttfts {
			if value > 0 {
				firstValues = append(firstValues, value)
			}
		}
		if len(firstValues) > 0 {
			sort.Slice(firstValues, func(i, j int) bool { return firstValues[i] < firstValues[j] })
			firstPercentile := func(percent int) time.Duration { return firstValues[(len(firstValues)-1)*percent/100] }
			message += fmt.Sprintf(" ttft_p50=%s ttft_p95=%s ttft_p99=%s", firstPercentile(50), firstPercentile(95), firstPercentile(99))
		}
	}
	fmt.Println(message)
}
