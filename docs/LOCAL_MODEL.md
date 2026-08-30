# Running a real CPU model for v0

The Compose stack uses `cmd/mock-model` so the full control-plane test is deterministic and fast. The gateway can proxy any OpenAI-compatible local model without code changes.

For example, start a local llama.cpp server with its OpenAI-compatible endpoint, then run the gateway in memory mode:

```bash
TESSERA_MODEL_URL=http://localhost:8081 \
TESSERA_MODEL_NAME=qwen-0.5b \
make run
```

The model server must expose `GET /healthz`, `POST /v1/chat/completions`, and support both normal JSON and Server-Sent Events responses. Record the exact model, quantization, context length, CPU, and command before running the benchmark.

With the gateway running and its key exported, the reproducible streaming run is:

```bash
export TESSERA_API_KEY="..."
export TESSERA_MODEL_URL=http://localhost:8081
export TESSERA_MODEL_NAME=qwen-0.5b
BENCH_REQUESTS=100 BENCH_CONCURRENCY=10 make real-model
```

The command reports end-to-end p50/p95/p99 and TTFT p50/p95/p99. It measures the configured model through the gateway; it does not fabricate a result when the model endpoint is absent.

The mock model is not a performance result. It exists to verify authentication, budgets, streaming, metering, persistence, and failure behavior without downloading model weights.
