#!/bin/sh
set -eu

BASE_URL=${TESSERA_E2E_URL:-http://localhost:8080}

for attempt in $(seq 1 60); do
	if curl -fsS "$BASE_URL/healthz" >/dev/null 2>&1; then
		break
	fi
	if [ "$attempt" -eq 60 ]; then
		echo "gateway did not become healthy" >&2
		exit 1
	fi
	sleep 1
done

playground=$(curl -fsS "$BASE_URL/playground")
api_key=$(printf '%s' "$playground" | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')
if [ -z "$api_key" ]; then
	echo "could not extract playground API key" >&2
	exit 1
fi

headers="Authorization: Bearer $api_key"
initial_usage=$(curl -fsS -H "$headers" "$BASE_URL/v1/usage")
initial_requests=$(printf '%s\n' "$initial_usage" | sed -n 's/.*"requests":\([0-9]*\).*/\1/p')
initial_requests=${initial_requests:-0}
expected_requests=$((initial_requests + 2))

completion=""
for attempt in $(seq 1 20); do
	if completion=$(curl -fsS -H "$headers" -H 'Content-Type: application/json' \
		-d '{"model":"mock-local","messages":[{"role":"user","content":"hello from e2e"}]}' \
		"$BASE_URL/v1/chat/completions"); then
		break
	fi
	if [ "$attempt" -eq 20 ]; then
		echo "gateway did not recover after dependency restoration" >&2
		exit 1
	fi
	sleep 1
done
printf '%s\n' "$completion" | grep -q 'chat.completion'

stream=$(curl -fsS -H "$headers" -H 'Content-Type: application/json' \
	-d '{"model":"mock-local","messages":[{"role":"user","content":"stream this"}],"stream":true}' \
	"$BASE_URL/v1/chat/completions")
printf '%s\n' "$stream" | grep -q '\[DONE\]'

for attempt in $(seq 1 20); do
	usage=$(curl -fsS -H "$headers" "$BASE_URL/v1/usage")
	current_requests=$(printf '%s\n' "$usage" | sed -n 's/.*"requests":\([0-9]*\).*/\1/p')
	if [ "${current_requests:-0}" -ge "$expected_requests" ]; then
		break
	fi
	if [ "$attempt" -eq 20 ]; then
		echo "usage ledger did not receive both successful requests: $usage" >&2
		exit 1
	fi
	sleep 1
done

metrics=$(curl -fsS "$BASE_URL/metrics")
printf '%s\n' "$metrics" | grep -q 'tessera_requests_total'

if curl -sS -o /tmp/tessera-e2e-budget.json -w '%{http_code}' -H "$headers" -H 'Content-Type: application/json' \
	-d '{"messages":[{"role":"user","content":"reject me"}],"max_tokens":6000}' \
	"$BASE_URL/v1/chat/completions" | grep -q '^429$'; then
	echo "budget rejection: ok"
else
	echo "budget rejection: failed" >&2
	exit 1
fi

echo "tessera v0 e2e: ok"
