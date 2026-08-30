# TESSERA v0 Runbook

## Start locally

```bash
docker compose up --build
sh scripts/e2e.sh
sh scripts/chaos.sh
```

`chaos.sh` temporarily stops Redis, the model, and PostgreSQL, then sends SIGTERM to the gateway. It restores the Compose services before exiting.

Open `http://localhost:8080/docs` for the browser playground.

## Gateway is unhealthy

Check `GET /healthz`, then inspect gateway logs. In Compose, verify PostgreSQL, Redis, and model health before restarting the gateway.

## Requests return 503 budget unavailable

This is intentional fail-closed behavior. Check Redis connectivity and the `TESSERA_REDIS_URL`. Do not bypass the budget check to restore availability.

## Requests are slow

Check `/metrics`, model health, upstream queueing, and client cancellation. The v0 mock model is not a performance benchmark; real latency measurements begin when the local model provider is configured.

## Usage is missing

The PostgreSQL ledger is fed asynchronously. Check database connectivity and gateway logs, then allow the bounded ledger buffer to drain. Usage events are idempotent by event ID.
