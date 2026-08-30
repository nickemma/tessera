#!/bin/sh
set -eu

BASE_URL=${TESSERA_E2E_URL:-http://localhost:8080}
playground=$(curl -fsS "$BASE_URL/playground")
api_key=$(printf '%s' "$playground" | sed -n 's/.*id="key" value="\([^"]*\)".*/\1/p')
headers="Authorization: Bearer $api_key"
body='{"messages":[{"role":"user","content":"failure exercise"}]}'

restore_dependencies() {
	docker compose start postgres redis model >/dev/null 2>&1 || true
}
trap restore_dependencies EXIT INT TERM

docker compose stop redis >/dev/null
redis_status=$(curl -sS -o /dev/null -w '%{http_code}' -H "$headers" -H 'Content-Type: application/json' -d "$body" "$BASE_URL/v1/chat/completions")
if [ "$redis_status" != 503 ]; then
	echo "Redis outage expected 503, got $redis_status" >&2
	docker compose start redis >/dev/null
	exit 1
fi
docker compose start redis >/dev/null

docker compose stop model >/dev/null
model_status=0
for attempt in 1 2 3 4; do
	model_status=$(curl -sS -o /dev/null -w '%{http_code}' -H "$headers" -H 'Content-Type: application/json' -d "$body" "$BASE_URL/v1/chat/completions")
done
docker compose start model >/dev/null
if [ "$model_status" != 503 ] && [ "$model_status" != 500 ]; then
	echo "model outage expected a service failure, got $model_status" >&2
	exit 1
fi

docker compose stop postgres >/dev/null
usage_status=$(curl -sS -o /dev/null -w '%{http_code}' -H "$headers" "$BASE_URL/v1/usage")
if [ "$usage_status" != 500 ] && [ "$usage_status" != 503 ]; then
	echo "PostgreSQL outage expected usage to fail closed, got $usage_status" >&2
	exit 1
fi
docker compose start postgres >/dev/null

docker compose kill -s SIGTERM gateway >/dev/null
for attempt in 1 2 3 4 5 6 7 8 9 10; do
	docker compose start gateway >/dev/null 2>&1 || true
	if curl -fsS "$BASE_URL/healthz" >/dev/null 2>&1; then
		break
	fi
	if [ "$attempt" -eq 10 ]; then
		echo "gateway did not recover after SIGTERM" >&2
		exit 1
	fi
	sleep 1
done

echo "tessera v0 chaos exercises: ok (redis=$redis_status model=$model_status postgres=$usage_status sigterm=ok)"
