# TESSERA v0 — The Build Plan

**Build first. Learn the concept at the moment the build needs it. See it work the same day.**

Ten milestones, weeks 1–10. Every milestone ends with something you can run and show. Nothing is taught before there's a reason to want it.

---

## How each milestone works

```
1. What we're building        one sentence, concrete
2. Build it                   working in the first session
3. Why it works               the concept, now that you've seen it
4. Why this and not that      the alternatives, and when they'd win
5. Break it                   a failure that exposes what you don't yet know
6. Fix it                     which is the next thing you learn
```

The tradeoff conversation is the part that matters most. Anyone can wire up a handler. Being able to say *"we used X, we rejected Y, and here's the situation where Y would be the right call"* is what separates an engineer from someone following a tutorial — and it's what system-design interviews are actually testing.

---

## M1 · A server that answers

**Week 1 · sessions 1–2**

### What you'll see working

```bash
curl -X POST localhost:8080/v1/chat -d '{"prompt":"hello"}'
{"response":"(canned)","tokens":7}
```

In the first three hours. A real HTTP service, running, responding.

### What you learn making it

The shape of a Go program. Packages and the `internal/` convention. HTTP handlers and what a `ResponseWriter` actually is. JSON encoding and decoding, and why the decoder can bite you. Errors as values, and why Go has no exceptions. Config from the environment. The project layout you'll use for the next year — ports and adapters, from commit one, so it never needs retrofitting.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| stdlib `net/http` + `chi` router | Gin, Echo, Fiber | A framework wins when you want batteries — validation, binding, middleware ecosystem — and don't mind the abstraction. Stdlib wins when you're building infrastructure and every layer needs to be inspectable. You're building infrastructure. |
| Ports and adapters from day one | Flat `main.go`, grow later | Flat wins for a script or a prototype you'll throw away. It loses the moment you want to swap Postgres for an in-memory fake in tests — which is week 2. |
| Config from environment | Config file, flags | Files win for complex nested config. Env wins for containers, and you're going to be in a container by week 8. |

### Break it

Send malformed JSON. Send a 500MB body. Send no `Content-Type`. Send nothing and hold the connection open.

### Fix it → what you learn next

Request size limits. Read timeouts. Which is why M1 ends with you knowing what a slowloris attack is without anyone having used the word.

### Done when

Running, responds correctly, rejects garbage with useful errors, has a `Makefile`, and a stranger can `make run` from a clean clone.

---

## M2 · It knows who you are

**Weeks 1–2 · sessions 3–6**

### What you'll see working

```bash
curl -H "Authorization: Bearer tsk_live_..." localhost:8080/v1/chat -d '...'
{"response":"..."}

curl localhost:8080/v1/chat -d '...'
{"error":"missing api key","code":"unauthenticated"}
```

### What you learn making it

Postgres and migrations. The repository pattern — and this is where **interfaces** finally make sense, because you'll write a real one and a fake one and swap them in tests. Pointers and value semantics, because you'll hit them in the same hour. Middleware chains. Hashing API keys rather than storing them, and constant-time comparison so you don't leak the key one byte at a time through timing.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| API keys | JWT, OAuth2, mTLS | JWT wins when you need stateless verification at scale and can live with revocation being hard. OAuth wins when a third party owns the identity. mTLS wins service-to-service. API keys win for machine clients where you need instant revocation — which is exactly what a budget-enforcing platform needs. |
| Store a hash, not the key | Store encrypted, store plaintext | Encrypted wins only if you must show the key again later. You never should — show it once at creation. |
| `pgx` directly | GORM, sqlc, ent | An ORM wins when you have fifty tables and a team that rotates. Raw SQL wins when the queries matter and you want to know exactly what hits the database. |

### Break it

Two requests with the same key at once. A key that was revoked one millisecond ago. A key from tenant A used against tenant B's data. Twenty thousand keys in the table — is your lookup still fast?

### Done when

Tenants and keys exist, auth middleware works, keys can be rotated and revoked, and there is a **test proving tenant A cannot read tenant B's anything.** Not a claim in the README — a test.

---

## M3 · It won't let you spend money you don't have

**Weeks 2–3 · sessions 7–10**

### What you'll see working

```bash
# tenant has 100 tokens left, asks for a 500-token completion
{"error":"budget exceeded","remaining":100,"required":500}
```

And the model was never called. That's the whole point.

### What you learn making it

Redis. Atomic operations and why check-then-act is a race. **Concurrency, taught here because you now have a real race to look at** — two requests for the same tenant arriving simultaneously, both reading "100 remaining," both proceeding. Mutexes, atomics, and why neither fixes a race that spans two processes. Lua scripts for atomicity in Redis.

And the design decision that defines the platform: **budgets fail closed, caches fail open.** If Redis is down you reject rather than serve unmetered. We'll spend a full session on why that asymmetry is correct and where it would be wrong.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| Redis for budget state | Postgres, in-memory | Postgres wins if you need the budget in a transaction with other writes. In-memory wins for a single instance — and dies the moment you run two. |
| Atomic Lua decrement | Read, check, write | The naive version is simpler and correct at low traffic. It's wrong the first time two requests overlap, which is a bug you'll only see in production. |
| Reserve then reconcile | Charge after completion | Charging after is simpler and accurate. Reserving is the only way to stop a runaway before the money is spent. Platforms reserve. |
| Fail closed | Fail open | Fail open wins when availability matters more than accounting — a free tier, a demo. Fail closed wins the moment real money is involved. |

### Break it

Kill Redis mid-request. Send 100 concurrent requests for a tenant with budget for 10 — how many get through? (If the answer isn't exactly 10, you have the bug we're here to find.) Make a request fail *after* reserving — does the reservation leak?

### Done when

Budgets are enforced before the model, concurrent requests can't overspend by even one token, Redis being down means rejection not silent success, and reservations are released when a request fails.

---

## M4 · It talks to a real model

**Weeks 3–4 · sessions 11–14**

### What you'll see working

Real tokens streaming into your terminal, one at a time, from a model running on your own machine.

### What you learn making it

HTTP clients properly — connection pooling, why the default client will hurt you, timeouts at every layer. `context` propagation and cancellation, threaded end to end, which almost nobody does correctly. Streaming: server-sent events, flushing, and what happens to a stream when the client hangs up. Then the model side: tokens, context windows, why length costs memory, prefill versus decode in plain English.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| Stream the response | Buffer and return | Buffering is far simpler and fine for short completions. Streaming wins the instant a user is watching, because time-to-first-token is what they perceive as speed. |
| Proxy to a separate model process | Embed inference in the gateway | Embedding avoids a network hop. Separating means you can scale, restart, and replace the model without touching the gateway — and in week 35 you'll swap CPU for GPU without changing a line of gateway code. |
| A provider interface | Call one backend directly | Direct is less code. The interface is what makes model routing and fallback possible in v1, and it costs you thirty lines now. |

### Break it

The model backend hangs and never responds. The client disconnects mid-stream — does your goroutine leak? The model returns a 500 halfway through streaming. The backend restarts while you hold pooled connections.

### Done when

Real streamed tokens, every I/O has a timeout, client disconnect cancels the upstream request, and there is a **measured number**: baseline TTFT without your gateway versus with it. That millisecond figure is your first credibility number.

---

## M5 · It survives being hammered

**Weeks 4–5 · sessions 15–18**

### What you'll see working

A load test at 2,000 requests/sec where your service serves what it can, rejects the rest in milliseconds, and does not fall over — with a graph showing exactly where the knee is.

### What you learn making it

Goroutines and the scheduler, taught now because you have something to hammer. Why "goroutines are cheap" is misleading. Bounded concurrency with a semaphore channel. Backpressure, and rejecting fast as a *feature*. Rate limiting. Retries with jitter, and why naive retries make an outage worse. Idempotency. Circuit breakers. Then profiling — `pprof`, goroutine leaks, allocation hotspots, and the escape analysis that explains them.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| Reject when saturated | Queue everything | Queuing wins when work is durable and latency doesn't matter — a job system. Rejecting wins for interactive requests, where a queued request is one the user already gave up on. |
| Token bucket | Leaky bucket, sliding window | Sliding window is more accurate and more expensive. Token bucket allows bursts, which real clients need. |
| Semaphore channel | Worker pool with a queue | A worker pool wins when work items are uniform and cheap. A semaphore wins when each request holds a long-lived stream. |
| Circuit breaker on the model backend | Retry harder | Retrying wins for transient blips. A breaker wins when the backend is genuinely down and your retries are what's keeping it down. |

### Break it

Ramp until it breaks and find out *what* broke — file descriptors, memory, CPU, or the backend. Make the model slow instead of dead. Have 500 clients connect and send one byte per minute.

### Done when

Concurrency has a ceiling, rejection is fast and counted, no goroutine leaks under an hour of load, and you have a published RPS-versus-p99 curve with the hardware named.

---

## M6 · It tells you what it's doing

**Weeks 5–6 · sessions 19–22**

### What you'll see working

A Grafana dashboard with your own request rate, latency histogram, and rejection count. And `kill -TERM` draining in-flight requests cleanly instead of dropping them.

### What you learn making it

Processes, signals, and exit codes — SIGTERM versus SIGKILL and why one is catchable. Graceful shutdown, properly, including what to do with a stream that's halfway through. Structured logging with request IDs. Prometheus: counters, gauges, and **histograms rather than averages** — because an average latency is a number that hides every problem you care about. Cardinality, and how a well-meaning label brings down your metrics backend.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| Histograms | Averages, p99 gauges | Averages are cheaper and useless. A pre-computed p99 can't be aggregated across instances; a histogram can. |
| Structured JSON logs | Plain text | Text wins for a human reading one machine. JSON wins the moment a machine has to search across many. |
| Pull metrics (Prometheus) | Push (StatsD) | Push wins for short-lived jobs that die before a scrape. Pull wins for long-running services and gives you health for free. |

### Break it

`kill -TERM` mid-stream. `kill -9` and compare. Fill the disk while logging. Add a label with unbounded cardinality — the tenant's prompt — and watch what happens.

### Done when

`/healthz`, `/readyz`, `/metrics`, JSON logs with request IDs, graceful shutdown that drains, and a dashboard you'd actually look at during an incident.

---

## M7 · It remembers what it cost

**Weeks 6–7 · sessions 23–26**

### What you'll see working

```bash
curl localhost:8080/v1/usage?tenant=acme&month=2026-09
{"requests":18422,"input_tokens":2.1e6,"output_tokens":840000,"cost_usd":31.40}
```

The question every finance team asks and almost no platform can answer.

### What you learn making it

Writing metering off the hot path — a buffered channel and a flusher goroutine, so the user's latency doesn't pay for your bookkeeping. At-least-once delivery and idempotency keys. What to do when the flusher's buffer fills. Aggregation queries, and indexes that make them not terrible.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| Async batched writes | Synchronous write per request | Synchronous is simpler and can't lose a record. Async keeps p99 clean. The right answer depends on whether losing one usage record is worse than adding 4ms to every request — and for a platform, it usually isn't. |
| Buffer with a bounded channel | Unbounded buffer | Unbounded never blocks, and turns a slow database into an out-of-memory kill. Bounded forces you to decide what happens when it's full — and deciding is the point. |
| Store raw events, aggregate on read | Store pre-aggregated | Pre-aggregating is faster to read and impossible to correct. Raw events let you recompute when you inevitably change the pricing model. |

### Break it

Kill the process with 500 unflushed records. Make Postgres unreachable for two minutes. Send the same request twice with the same idempotency key.

### Done when

Usage is queryable per tenant per period, metering adds under 1ms to p99 (measured), and a crash loses at most one flush interval — a number you can state.

---

## M8 · It doesn't care where it runs

**Weeks 7–8 · sessions 27–30**

### What you'll see working

A 15MB image. `docker compose up` bringing the whole stack — gateway, Postgres, Redis, model — in one command.

### What you learn making it

Namespaces and cgroups — and you'll build a ~200-line container in Go so Docker stops being magic. Multi-stage builds. Static linking, and why a Go binary can run on `scratch` with literally nothing else in the image. Memory limits and the OOM killer, deliberately triggered. Image layers and caching.

### Why this and not that

| Choice | Rejected | When the other wins |
|---|---|---|
| `scratch` base image | Alpine, distroless, Debian | Debian wins when you need a shell to debug in production. Distroless is the middle ground. `scratch` wins for a static Go binary where attack surface is the priority — and you lose the ability to `exec` into it, which you should feel once. |
| Compose for local | `kind` for everything | Kind is closer to production. Compose starts in three seconds and you'll run it two hundred times. Use both, for different jobs. |

### Break it

Set a memory limit below what the service needs and watch the OOM kill — then find it in `dmesg`, because your application will have logged nothing.

### Done when

Under 20MB, non-root, read-only root filesystem, `compose up` works from a clean clone, and you can explain what a container actually is without saying "lightweight VM."

---

## M9 · It survives you

**Weeks 8–9 · sessions 31–34**

### What you'll see working

The stack on `kind`. You delete a pod; traffic keeps flowing. You break a dependency; the service degrades correctly instead of dying.

### What you learn making it

Kubernetes as a user: pods, deployments, services, probes, requests and limits, config and secrets. Why liveness and readiness are different and what happens when you conflate them. Rolling updates.

Then **failure day**, the real one: Redis dies mid-request · Postgres pool exhausted · the model hangs forever · a client disconnects mid-stream · 500 concurrent requests · the container hits its memory limit · a tenant's budget goes negative through a race · a certificate expires.

For each: what *should* happen, what actually happened, and the gap between them.

### Done when

Every failure has documented, correct behaviour — not "didn't crash," but the right thing. Each one written up in `breaks/` with a root cause that is a decision, not an event.

---

## M10 · It's measured, published, and permanent

**Week 10 · sessions 35–40**

### What you'll see working

A benchmark table with your name on it, a published post, and a service that from this day forward does not get turned off.

### What you learn making it

Load testing methodology — open versus closed models, warm-up, and why most benchmarks people post are wrong. Reading a flame graph. Writing a design doc and a runbook someone else could follow.

### Done when

- Full benchmark suite, hardware named, reproducible
- `README.md` · `DESIGN_DOC.md` · `BENCHMARKS.md` · `RUNBOOK.md` · `THREAT_MODEL.md`
- `INCIDENTS.md`, empty and dated — from here it fills up on its own
- One published piece: the gateway-overhead number, or the concurrency bug from M3
- The four reflection questions answered
- **It is running. It stays running.**

---

## What got deferred, and where it went

Being explicit so nothing quietly vanishes:

| Topic | Why it's not here | Where it lands |
|---|---|---|
| CPU, cache, memory hierarchy | No build surface in a gateway. Nothing you'd do differently. | Lattice, week 11 — the storage engine, where cache layout decides the design |
| Garbage collector internals | Only interesting once you have allocation pressure | M5, from a real profile |
| TCP and TLS internals | You'll *use* them in M4 and M9 | Deep-dive in Lattice when cross-node traffic makes it matter |
| Escape analysis | Abstract before you have a hot path | M5, next to the profiler output |

**The rule:** if a concept has no visible surface, it waits until it does. The only exceptions are the ones that genuinely can't work that way — Raft is the big one, and it gets four scheduled weeks in Lattice.
