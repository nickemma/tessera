#!/bin/sh
set -eu

: "${TESSERA_API_KEY:?set TESSERA_API_KEY to a gateway API key}"

gateway_url=${TESSERA_GATEWAY_URL:-http://localhost:8080/v1/chat/completions}
model=${TESSERA_MODEL_NAME:-local-model}
requests=${BENCH_REQUESTS:-100}
concurrency=${BENCH_CONCURRENCY:-10}

if [ -z "${TESSERA_MODEL_URL:-}" ]; then
	echo "TESSERA_MODEL_URL is required; point it at an OpenAI-compatible local model server" >&2
	exit 1
fi

if ! curl -fsS "$TESSERA_MODEL_URL/healthz" >/dev/null 2>&1; then
	echo "model health check failed: $TESSERA_MODEL_URL/healthz" >&2
	exit 1
fi

go run ./bench -url "$gateway_url" -key "$TESSERA_API_KEY" -model "$model" -requests "$requests" -concurrency "$concurrency" -stream
