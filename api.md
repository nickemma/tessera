# TESSERA V0 API and Test Walkthrough

This guide is meant to be followed from top to bottom. Run every command from the repository root:

```bash
cd /home/techieemma/projects/tessera
```

There are two ways to run TESSERA:

- **Fast mode:** the gateway runs locally with in-memory state and the canned provider.
- **Full mode:** Docker Compose runs the gateway, PostgreSQL, Redis, and the deterministic mock model.

Use only one mode at a time because both use port `8080`.

## 1. Check prerequisites

```bash
go version
docker version
```

The repository currently builds with Go 1.26. Docker is only required for full mode.

## 2. Run fast mode

In terminal 1, start the gateway:

```bash
make run
```

Leave that terminal running. In terminal 2, set the base URL:

```bash
export TESSERA_URL=http://localhost:8080
```

Check that the gateway is alive:

```bash
curl -i "$TESSERA_URL/healthz"
```

Expected result: HTTP `200` and a response containing `"status":"ok"`.

## 3. Open the playground and get the demo key

Open this URL in a browser:

```text
http://localhost:8080/playground
```

The same page is also available at `/docs`. It contains a prompt box, model field, streaming checkbox, and the local demo API key.

For terminal testing, extract the key automatically:

```bash
export TESSERA_API_KEY="$(curl -fsS "$TESSERA_URL/playground" | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')"
test -n "$TESSERA_API_KEY" && echo "API key loaded"
```

This key is for local testing only. It is regenerated when the gateway restarts in fast mode.

## 4. Test authentication

An API request without a key must be rejected:

```bash
curl -i -X POST "$TESSERA_URL/v1/chat/completions" \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"hello"}]}'
```

Expected result: HTTP `401`.

## 5. Send a normal OpenAI-compatible request

```bash
curl -sS -X POST "$TESSERA_URL/v1/chat/completions" \
  -H "Authorization: Bearer $TESSERA_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "model":"canned-local",
    "messages":[{"role":"user","content":"Explain what TESSERA does in one sentence."}],
    "max_tokens":128
  }'
```

The response contains an assistant message, token usage, and Tessera's remaining budget.

## 6. Test streaming

Use `-N` so curl prints Server-Sent Events immediately:

```bash
curl -N -sS -X POST "$TESSERA_URL/v1/chat/completions" \
  -H "Authorization: Bearer $TESSERA_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "model":"canned-local",
    "messages":[{"role":"user","content":"Stream this response."}],
    "stream":true
  }'
```

A successful stream ends with:

```text
data: [DONE]
```

## 7. Test the native TESSERA endpoint

TESSERA also exposes a smaller native request shape:

```bash
curl -sS -X POST "$TESSERA_URL/v1/chat" \
  -H "Authorization: Bearer $TESSERA_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"model":"canned-local","prompt":"hello from the native API"}'
```

The text-completion compatibility endpoint is also available:

```bash
curl -sS -X POST "$TESSERA_URL/v1/completions" \
  -H "Authorization: Bearer $TESSERA_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"model":"canned-local","prompt":"hello from completions"}'
```

## 8. Check usage and metrics

Usage is scoped to the authenticated tenant:

```bash
curl -sS "$TESSERA_URL/v1/usage" \
  -H "Authorization: Bearer $TESSERA_API_KEY"
```

Prometheus-format metrics are public in V0:

```bash
curl -sS "$TESSERA_URL/metrics" | grep tessera_
```

After successful requests, look for `tessera_requests_total`, `tessera_completed_total`, `tessera_input_tokens_total`, `tessera_output_tokens_total`, `tessera_request_latency_seconds`, and `tessera_ttft_seconds`.

## 9. Use the CLI

The `tesserac` command is another way to exercise the same API:

```bash
go run ./cmd/tesserac \
  -key "$TESSERA_API_KEY" \
  -prompt "hello from the TESSERA CLI"

go run ./cmd/tesserac \
  -key "$TESSERA_API_KEY" \
  -stream \
  -prompt "stream from the TESSERA CLI"
```

The key can also be supplied through `TESSERA_API_KEY` without passing `-key`.

## 10. Run the full dependency-backed playground

Stop fast mode with `Ctrl-C` in terminal 1 first. Then start the full stack:

```bash
make compose-up
```

Wait until the services are healthy:

```bash
docker compose ps
curl -fsS http://localhost:8080/healthz
```

Extract the new Compose demo key because the gateway creates a fresh local playground key when its container starts:

```bash
export TESSERA_URL=http://localhost:8080
export TESSERA_API_KEY="$(curl -fsS "$TESSERA_URL/playground" | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')"
```

Repeat steps 5–9. In full mode, tenant and usage data are backed by PostgreSQL, while budgets and rate limits are backed by Redis.

## 11. Run automated verification

The end-to-end test checks authentication, completion, streaming, usage persistence, metrics, and budget rejection:

```bash
make e2e
```

The failure-day test temporarily stops Redis, the model, and PostgreSQL, then sends SIGTERM to the gateway and verifies recovery. It restores the services before it exits:

```bash
make chaos
```

Run the Go quality checks:

```bash
make build
make test
go vet ./...
```

## 12. Run a local benchmark

Load the key and run a small JSON benchmark:

```bash
export TESSERA_API_KEY="$(curl -fsS "$TESSERA_URL/playground" | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')"
go run ./bench -key "$TESSERA_API_KEY" -requests 10 -concurrency 10
```

Measure streaming latency and time-to-first-event:

```bash
go run ./bench -key "$TESSERA_API_KEY" \
  -requests 10 \
  -concurrency 10 \
  -stream
```

The Compose mock model is a control-plane test double. Its numbers must not be presented as real model throughput.

## 13. Connect a real OpenAI-compatible local model

The gateway can proxy a local model server that exposes `/healthz` and `/v1/chat/completions` with JSON and SSE support.

Start the model server separately, then run the gateway with:

```bash
export TESSERA_MODEL_URL=http://localhost:8081
export TESSERA_MODEL_NAME=qwen-0.5b
make run
```

In another terminal, run the real-model benchmark:

```bash
export TESSERA_API_KEY="$(curl -fsS http://localhost:8080/playground | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')"
BENCH_REQUESTS=100 BENCH_CONCURRENCY=10 make real-model
```

Record the exact model, quantization, context length, CPU/GPU, runtime command, workload, and warm-up policy in [`docs/benchmarks.md`](docs/benchmarks.md). Do not download model weights or rent a GPU without deciding on the cost first.

## 14. Stop the full stack

```bash
make compose-down
```

This stops and removes the containers but keeps the named PostgreSQL volume. Do not use `docker compose down -v` unless you intentionally want to delete the local database volume and all stored usage data.

## API route summary

| Method | Route | Authentication | Purpose |
|---|---|---|---|
| GET | `/healthz` | No | Liveness check |
| GET | `/playground` | No | Browser playground and demo key |
| GET | `/docs` | No | Playground alias |
| GET | `/openapi.json` | No | OpenAPI contract |
| GET | `/metrics` | No | Prometheus metrics |
| POST | `/v1/chat` | Bearer key | Native chat API |
| POST | `/v1/chat/completions` | Bearer key | OpenAI-compatible chat API |
| POST | `/v1/completions` | Bearer key | OpenAI-compatible text API |
| GET | `/v1/usage` | Bearer key | Current tenant usage |

For the machine-readable request/response contract, use:

```bash
curl -sS http://localhost:8080/openapi.json | python3 -m json.tool
```
