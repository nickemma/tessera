# TESSERA Benchmarks

## V0 control-plane sanity sample — 2026-08-30

This is a local Docker Compose measurement of gateway overhead and control-plane behavior. It is **not** an inference-performance result: the upstream is `cmd/mock-model`, which returns one deterministic response immediately.

| Mode | Requests | Concurrency | Success | p50 | p95 | p99 | TTFT p50 | TTFT p95 | TTFT p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| JSON | 10 | 10 | 10/10 | 28.5 ms | 43.0 ms | 43.0 ms | — | — | — |
| SSE | 10 | 10 | 10/10 | 34.2 ms | 53.1 ms | 53.1 ms | 32.8 ms | 51.2 ms | 51.2 ms |

Environment: AMD Ryzen 7 PRO 5850U, 16 vCPUs, x86_64, Go 1.26.6, Docker Engine 29.7.1. The gateway, mock model, PostgreSQL 17, and Redis 7 ran locally through Compose. The workload was the benchmark default prompt, with a warm stack and no warm-up requests.

Commands:

```bash
key=$(curl -fsS http://localhost:8080/playground | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')
go run ./bench -key "$key" -requests 10 -concurrency 10
go run ./bench -key "$key" -requests 10 -concurrency 10 -stream
```

The benchmark runner also supports `-model`, `-prompt`, `-requests`, `-concurrency`, and `-stream`. For a real local model, use `make real-model` as described in [`LOCAL_MODEL.md`](LOCAL_MODEL.md).

## What is still unmeasured

- Real CPU-model quality, TTFT, tokens/sec, and gateway-versus-direct comparison: no model runtime or weights are present in this workspace.
- The planned 2k-RPS rejection/capacity curve: the current runner is a transparent smoke/load tool, not a production load generator.
- GPU utilization, KV-cache behavior, autoscaling, and break-even cost: these belong to the V1 GPU environment.

Every future result must state hardware, model, quantization, context length, workload distribution, concurrency, warm-up policy, and the exact command used to reproduce it.
