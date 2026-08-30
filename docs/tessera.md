# TESSERA

**A multi-tenant LLM inference platform — the gateway, the economics, and eventually the GPUs.**

`github.com/nickemma/tessera` · Project 1 (weeks 1–10) and Project 4 (weeks 35–46)

---

## What we're trying to achieve

Anyone can call an LLM API. Almost nobody can run one as a **platform** — where multiple teams share the same models, each has a budget, each is isolated from the others, and somebody can answer the question "what did last month cost, and which tenant caused it?"

That gap is the entire project.

Tessera is the thing that sits between users and models: it authenticates, checks whether a tenant can afford the request *before* it reaches a GPU, routes to the right model, caches what it can, meters what it can't, and reports the cost. The model is the least interesting part. The platform around it is where all the engineering is.

**The claim it earns you:** *"I ran a multi-tenant inference platform. Here's the p99, here's the cost per million tokens, here's what happened the night the metering store went down."*

---

## Why it's first

Its v0 needs a Go HTTP service, Postgres, Redis, and Docker. That's reachable from where you are now, which means something is running by week 3 and shipped by week 10. Nothing else in the portfolio has a floor that low.

And it never turns off. From week 10 to week 52 it accumulates uptime, incidents, and an operating history — which is the one thing a portfolio genuinely cannot fake.

---

# PART ONE — TESSERA v0 (weeks 1–10)

CPU only. No GPU spend. A tiny model. **The model being slow is irrelevant** — every platform concern is exercised identically whether the model takes 50ms or 5 seconds.

## What it does when it's done

```
client
  │  API key
  ▼
gateway (Go)
  ├─ authenticate the tenant
  ├─ check the token budget          ← before the model, always
  ├─ rate limit
  ├─ route to a model
  ├─ stream the response back
  └─ meter what was used
       │
       ▼
  small model on CPU (vLLM or llama.cpp, Qwen 0.5B class)
```

Plus: Postgres for tenants and usage, Redis for budgets and rate limits, Prometheus metrics, structured logs, health checks, graceful shutdown, and a Docker Compose stack that comes up with one command.

## What it teaches — and when

Each phase drags in the stages it needs. This is the whole point of the reordering.

### Phase 1 · Weeks 1–2 — Go, and the machine underneath it *(S1)*

You're two lessons in already. Finish the block: memory, CPU and cache, the kernel, processes and signals, files and descriptors, then Go proper — types, pointers, interfaces, goroutines, channels, context, mutexes, races, the GC, profiling.

**Built by the end:** a Go HTTP service that reads config, logs in JSON, exposes `/healthz` and `/metrics`, handles SIGTERM gracefully, and refuses to die badly.

### Phase 2 · Week 3 — Serving many users from one box *(S4)*

Bounded concurrency, backpressure, timeouts on every I/O path, retries with jitter, idempotency, circuit breakers, rate limiting.

**Built:** the gateway skeleton — it accepts requests, enforces a concurrency ceiling, rejects fast when saturated, and never leaks a goroutine.

**First public artifact ships here.** Week 3.

### Phase 3 · Week 4 — Sockets, HTTP, TLS *(S3)*

What's under `net/http`. The TCP handshake, the TLS handshake, HTTP/1.1 versus HTTP/2, keep-alives, connection pooling, and — because you're about to stream tokens — how server-sent events actually work over a long-lived connection.

**Built:** TLS termination, streaming responses, connection pooling to the model backend.

### Phase 4 · Weeks 5–6 — Tenancy, budgets, metering *(S6 first half)*

Now the platform logic. Tenants, API keys with rotation, per-tenant token budgets in Redis, usage records in Postgres, per-tenant rate limits.

The critical design decision, and we'll spend a session on it: **budgets fail closed, caches fail open.** If Redis is down you reject rather than serve unmetered. If the cache is down you serve slower. Getting that distinction right is the difference between a platform and a demo.

**Built:** a tenant can run out of budget and be rejected before the model is touched. There's a test that proves tenant A cannot read tenant B's usage.

### Phase 5 · Week 7 — Actually serving a model *(S6 second half)*

Tokens, context windows, why context length costs memory, prefill versus decode in plain English. Run Qwen 0.5B on CPU behind vLLM or llama.cpp. Wire it up. Stream tokens through the gateway.

**Measured:** baseline TTFT and tokens/sec *without* your gateway, then *with* it. The overhead in milliseconds is your first credibility number.

### Phase 6 · Weeks 8–9 — Containers and Kubernetes *(S2, S5)*

Processes, cgroups, namespaces — then a container from scratch in Go (~200 lines) so Docker stops being magic. Then Docker properly: multi-stage builds, a Go binary on `scratch` under 20MB. Then Kubernetes as a user on `kind`: pods, deployments, services, probes, resource limits, config and secrets.

**Built:** the whole stack running on `kind`, and a Compose file for local work.

### Phase 7 · Week 10 — Break it, measure it, ship it

Failure day. I inject: Redis dies mid-request · Postgres connection pool exhausted · the model backend hangs and never responds · a client disconnects mid-stream · 500 concurrent requests · the container hits its memory limit · a tenant's budget goes negative through a race.

Then: load test, publish the numbers, write the four reflection questions, and **turn it on for good**.

## Done means

- [ ] A tenant can authenticate, be budget-checked, get a streamed response, and be metered — end to end
- [ ] Budgets fail closed. There is a test that proves it.
- [ ] Tenant isolation is proven by test, not asserted in a README
- [ ] Gateway overhead measured in milliseconds, hardware named
- [ ] Survives all seven week-10 failures with correct behaviour, not just "doesn't crash"
- [ ] Runs on `kind` from a clean clone with one command
- [ ] `README.md` · `DESIGN_DOC.md` · `BENCHMARKS.md` · `RUNBOOK.md` · `INCIDENTS.md` (empty, dated, ready)
- [ ] **It is running, and stays running**

## Roles this alone opens

Backend Engineer (Go) · API Platform Engineer · Backend at an AI product company · junior DevOps. Not yet AI infrastructure — that's v1.

---

# PART TWO — TESSERA v1 (weeks 35–46)

Real GPUs, real inference engineering, and the platform grown up. By now you'll have Lattice and Plinth behind you, so this phase is pure AI infrastructure with no infrastructure prerequisites left to teach.

## What it teaches

### Phase 8 · Weeks 35–36 — What's inside a model *(S14)*

Intuition first, always. Embeddings. Attention taught the way you'd explain *"the cat sat on the mat because it was tired — what does 'it' refer to?"* Then Q, K, V. Then transformers, training versus inference, quantisation.

**Built:** a tiny transformer, once, following Karpathy. So that nothing downstream is magic.

### Phase 9 · Weeks 37–40 — Why serving an LLM is different *(S15)*

The heart of the project. Prefill versus decode and why they have completely different performance characteristics. KV cache and its memory arithmetic — you'll compute it by hand for a given model and batch size. Continuous batching. Chunked prefill. Prefix caching. Speculative decoding. Quantisation formats. TTFT versus inter-token latency and why one number is not enough.

The insight the whole phase turns on: **decode is memory-bandwidth bound, not compute bound.** Everything in inference optimisation follows from that.

**Built:** real vLLM on a rented GPU behind your gateway. Read vLLM's scheduler and block manager source.

**Measured:** TTFT, inter-token latency, tokens/sec at batch 1/8/32, GPU utilisation, KV cache usage ratio, **cost per million tokens**, and the break-even volume against a hosted API.

**GPU access:** rent, don't buy. L4 or A10 class on RunPod or Vast, roughly $0.30–0.80/hour. Budget $80–150 for the phase, used in scripted 6–8 hour sessions with everything prepared in advance so the clock isn't running while you think.

### Phase 10 · Weeks 41–43 — Beyond one GPU *(S16)*

GPU memory layout. Tensor, pipeline, and expert parallelism — what each splits and what each costs in communication. Collective operations. Request routing across replicas. Autoscaling on **queue depth**, not CPU. GPU scheduling on Kubernetes: device plugin, resource requests, MIG, DCGM metrics.

### Phase 11 · Weeks 44–46 — The platform *(S17)*

Everything converges. Model routing with reported fallback. Semantic caching, namespaced per tenant. Model registry, versioning, canary deployment, rollback. Budgets enforced pre-GPU at real scale. Full metering and audit.

**And Lattice folds in as the retrieval tier**, giving you one system rather than two:

```
client → gateway → retrieval (Lattice) → inference (vLLM on GPUs) → metering → audit
```

That is a distributed AI infrastructure platform, singular. It's also the architecture most companies actually run, which makes it the most interview-legible thing you can build.

**New number this unlocks:** how much of your end-to-end latency budget retrieval eats versus prefill versus decode. Very few people can answer that from measurement.

## Done means (v1)

- [ ] Real model on real GPUs, serving through your gateway, on Kubernetes
- [ ] Autoscaling on queue depth, demonstrated under load
- [ ] TTFT, ITL, tokens/sec at three batch sizes, GPU utilisation, cost per million tokens — all measured, hardware named
- [ ] Break-even analysis versus a hosted API, with the assumptions stated
- [ ] Lattice serving as the retrieval tier, with the latency split published
- [ ] Model rollback demonstrated, timed
- [ ] Alerting on KV cache usage ratio — latency degrades before throughput does, so cache pressure is the leading indicator
- [ ] `INCIDENTS.md` with real entries accumulated since week 10
- [ ] A 5-minute architecture walkthrough recorded

## Roles this opens

ML Infrastructure Engineer · Inference Engineer · AI Platform Engineer · GPU Infrastructure Engineer · most MLOps roles (see the caveat in `PROJECT-TRACK.md`) · and it upgrades every platform and SRE application you have open.

---

## Repo layout

```
tessera/
├─ cmd/gateway/
├─ internal/
│   ├─ app/                composition root
│   ├─ platform/           config, logging, metrics, tracing, errors
│   ├─ modules/
│   │   ├─ tenancy/        {domain, ports, app, adapters}
│   │   ├─ budget/
│   │   ├─ routing/
│   │   ├─ metering/
│   │   └─ inference/
│   └─ providers/          canned, vllm, openai-compatible, local
├─ internal/transport/     HTTP transport adapters
├─ migrations/  ├─ deploy/{compose,k8s,terraform}
├─ bench/  ├─ docs/  └─ README.md
```

Ports and adapters, modular monolith, repository pattern — the standard you already apply everywhere.

---

## The one thing to get right

Not the model. Not the GPUs. **The economics layer.**

Every company serving models has the same problem: they cannot tell who spent what, they cannot stop a runaway tenant before the bill arrives, and they cannot answer whether self-hosting beats the API. If Tessera answers those three questions with numbers you measured, it's worth more than a technically fancier system that can't.
