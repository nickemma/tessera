package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

type Registry struct {
	requests       atomic.Uint64
	completed      atomic.Uint64
	budgetRejected atomic.Uint64
	rateLimited    atomic.Uint64
	saturated      atomic.Uint64
	inputTokens    atomic.Uint64
	outputTokens   atomic.Uint64
	histogramMu    sync.Mutex
	requestLatency histogram
	ttft           histogram
}

type histogram struct {
	buckets []uint64
	count   uint64
	sum     float64
}

var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10}

func New() *Registry {
	return &Registry{
		requestLatency: histogram{buckets: make([]uint64, len(latencyBuckets))},
		ttft:           histogram{buckets: make([]uint64, len(latencyBuckets))},
	}
}

func (r *Registry) Request()        { r.requests.Add(1) }
func (r *Registry) Completed()      { r.completed.Add(1) }
func (r *Registry) BudgetRejected() { r.budgetRejected.Add(1) }
func (r *Registry) RateLimited()    { r.rateLimited.Add(1) }
func (r *Registry) Saturated()      { r.saturated.Add(1) }
func (r *Registry) Tokens(input, output int) {
	r.inputTokens.Add(uint64(input))
	r.outputTokens.Add(uint64(output))
}

func (r *Registry) RequestLatency(seconds float64) {
	r.observe(&r.requestLatency, seconds)
}

func (r *Registry) TTFT(seconds float64) {
	r.observe(&r.ttft, seconds)
}

func (r *Registry) observe(target *histogram, seconds float64) {
	if seconds < 0 {
		return
	}
	r.histogramMu.Lock()
	defer r.histogramMu.Unlock()
	target.count++
	target.sum += seconds
	for i, bucket := range latencyBuckets {
		if seconds <= bucket {
			target.buckets[i]++
		}
	}
}

func (r *Registry) Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# TYPE tessera_requests_total counter\ntessera_requests_total %d\n", r.requests.Load())
	fmt.Fprintf(w, "# TYPE tessera_completed_total counter\ntessera_completed_total %d\n", r.completed.Load())
	fmt.Fprintf(w, "# TYPE tessera_budget_rejections_total counter\ntessera_budget_rejections_total %d\n", r.budgetRejected.Load())
	fmt.Fprintf(w, "# TYPE tessera_rate_limited_total counter\ntessera_rate_limited_total %d\n", r.rateLimited.Load())
	fmt.Fprintf(w, "# TYPE tessera_saturated_total counter\ntessera_saturated_total %d\n", r.saturated.Load())
	fmt.Fprintf(w, "# TYPE tessera_input_tokens_total counter\ntessera_input_tokens_total %d\n", r.inputTokens.Load())
	fmt.Fprintf(w, "# TYPE tessera_output_tokens_total counter\ntessera_output_tokens_total %d\n", r.outputTokens.Load())
	r.histogramMu.Lock()
	writeHistogram(w, "tessera_request_latency_seconds", r.requestLatency)
	writeHistogram(w, "tessera_ttft_seconds", r.ttft)
	r.histogramMu.Unlock()
}

func writeHistogram(w http.ResponseWriter, name string, value histogram) {
	fmt.Fprintf(w, "# TYPE %s histogram\n", name)
	for i, bucket := range latencyBuckets {
		fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", name, bucket, value.buckets[i])
	}
	fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n%s_sum %g\n%s_count %d\n", name, value.count, name, value.sum, name, value.count)
}
