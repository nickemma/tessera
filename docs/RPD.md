# RPD — TESSERA v1

**System:** TESSERA — LLM Inference Platform
**Owner:** Nicholas Emmanuel
**Status:** Proposed
**Date:** August 2026

---

## 1. Problem

A team that wants to use an open model has three options, and all three are bad at organizational scale:

1. **Call a hosted API.** Fine until the data cannot leave the network, the volume makes per-token pricing worse than owned hardware, or the model you need is not offered.
2. **Run `vllm serve` on a box.** Works for one team. No authentication, no cost attribution, no budgets, no autoscaling, no answer to "who used the GPU last month" or "why was my request slow."
3. **Have every team run their own.** GPUs sit idle in N places instead of being shared, and nobody can state what inference costs the company.

The gap is not the inference engine — vLLM solved that. The gap is everything required to turn a running model into a service that several teams can depend on and someone can be accountable for.

## 2. Goal

Serve open models to multiple internal consumers with enforced per-tenant budgets, measured cost attribution, and latency you can explain — on hardware whose utilization is known rather than assumed.

## 3. Non-goal: why not just buy

| Option | When it wins | Why it does not solve this |
|---|---|---|
| OpenAI / Anthropic / Bedrock | Most of the time, honestly | Data residency constraints; sustained-volume economics; models not offered |
| Together / Fireworks / Replicate | Open models without ops | Same residency constraint; no per-team budget enforcement inside your org |
| Bare `vllm serve` | One team, one model | No authn, budgets, cost attribution, autoscaling, or routing |
| Ollama | Local development | Not a multi-tenant serving platform |

**The break-even volume is a deliverable, not an assumption.** `docs/benchmarks.md` will publish the token volume below which this project loses to a hosted API. If the answer is "we are below it," that is a legitimate finding and it gets published.

## 4. Users

| Role | Needs |
|---|---|
| **Application team** | An OpenAI-compatible endpoint, a key, and a predictable latency envelope |
| **Platform engineer** | To add a model without touching the gateway; to know GPU utilization |
| **Engineering manager** | Monthly cost per team, and a budget that is enforced rather than reported after the fact |
| **Security** | No cross-tenant data leakage, including via cache; auditable access |

## 5. User stories

- As an **application team**, I want an OpenAI-compatible endpoint so that I can point an existing SDK at it with a base-URL change.
- As an **application team**, I want streaming responses so that time-to-first-token is what my users feel, not total generation time.
- As an **application team**, I want to know which model actually served my request, so that a silent fallback does not become a mystery quality regression.
- As a **platform engineer**, I want to deploy a new model version as a canary revision so that a bad model does not take down the endpoint.
- As a **platform engineer**, I want autoscaling driven by queue depth so that capacity tracks demand rather than CPU noise.
- As a **manager**, I want a per-team token budget that rejects requests when exhausted, so that spend is bounded rather than discovered.
- As **security**, I want the semantic cache namespaced per tenant so that one tenant can never receive a response generated from another's prompt.

## 6. Functional requirements

**Gateway**
1. Expose an OpenAI-compatible `/v1/chat/completions` and `/v1/completions`, streaming and non-streaming.
2. Authenticate every request by tenant API key.
3. Enforce per-tenant token budget and request rate before dispatching to a GPU.
4. Return `429` with a reset timestamp when a budget or rate limit is exhausted.
5. Route to a model based on requested model name, tenant policy, and deadline.
6. Fall back to an alternate model on upstream failure, and report the substitution in the response.
7. Look up a per-tenant semantic cache before inference; return cache hits without dispatch.
8. Meter prompt and completion tokens per request into a durable usage ledger.
9. Report model served, cache status, and token usage in the response.

**Serving**
10. Deploy models as KServe InferenceServices backed by vLLM.
11. Support multiple concurrent models on shared or dedicated GPU nodes.
12. Autoscale replicas on queue depth; support scale-to-zero for non-latency-sensitive models.
13. Roll out a new model version as a canary before shifting full traffic.
14. Drain a node gracefully: no in-flight stream is truncated without a typed error.

**Operations**
15. Provision GPU node pools via Terraform, including spot/preemptible policy.
16. Reconcile all serving configuration via Argo CD from git.
17. Expose the metrics listed in the README, including cost attribution per tenant.
18. Publish a benchmark report with method, curves, cost, and break-even analysis.

## 7. Acceptance criteria

**Streaming and correctness**
> Given a streaming request
> When the response begins
> Then the first token arrives within the TTFT SLO, and the stream terminates with a usage trailer stating prompt tokens, completion tokens, model served, and cache status.

> Given a GPU node is drained during an in-flight stream
> When the node terminates
> Then the client receives a typed error, never a silently truncated response.

**Budgets**
> Given a tenant that has consumed its period budget
> When it sends another request
> Then the gateway returns 429 with a reset time, **no GPU work occurs**, and the rejection is counted.

> Given Redis is unavailable
> When a request arrives
> Then the request is rejected rather than served unmetered — budgets fail closed.

**Cache isolation**
> Given tenant A has a cached response for a prompt
> When tenant B sends a semantically identical prompt
> Then B receives a freshly generated response and A's cached entry is never returned.

**Routing**
> Given the primary model is unavailable
> When a request arrives with an eligible fallback
> Then the fallback serves it and the response reports the substitution.

**Autoscaling**
> Given sustained load raising queue depth above threshold
> When the autoscaler evaluates
> Then replicas increase, and queue wait returns below SLO within the documented scale-up window.

**Canary**
> Given a new model revision deployed at 10% traffic
> When it fails its readiness or error-rate gate
> Then traffic returns to the previous revision without operator action.

**Capacity**
> Given the load test at maximum sustainable throughput
> When it completes
> Then `docs/benchmarks.md` contains measured TTFT, inter-token latency, tokens/sec, GPU utilization, and cost per million tokens — with the method stated and no estimated values.

## 8. Non-functional requirements

| Property | Target |
|---|---|
| Availability | 99.5% / 30 days |
| TTFT | p99 < 1s below capacity |
| Inter-token latency | p99 < 50ms |
| GPU utilization under sustained load | > 70% |
| Gateway overhead | < 10ms added at p99 |
| Cross-tenant isolation | No shared cache entries; no shared budget state |
| Cost attribution | Every served token attributed to exactly one tenant |

## 9. Out of scope for v1

- Fine-tuning and training
- Multi-region or multi-cloud serving
- Embedding-model serving as a separate product surface (the cache uses one internally)
- RAG orchestration — that is [LATTICE](https://github.com/nickemma/lattice)
- Speculative decoding and quantization experiments (v2, once a baseline exists to compare against)
- A chat UI

## 10. Success metrics

- Two internal consumers served through the gateway with enforced budgets
- A published benchmark report with measured numbers and a stated break-even volume
- GPU utilization above 70% under sustained load — or a documented explanation of why not
- A second engineer resolves an "inference is slow" alert using only the runbook

## 11. Build order

Each row runs and is measurable. Do not start a row until the row above works.

| # | Increment | Requirements | Done when |
|---|---|---|---|
| 1 | Terraform GPU node pool + NVIDIA device plugin | 15 | `nvidia-smi` runs in a scheduled pod |
| 2 | vLLM on one node, one model, direct access | 10 | `curl` gets a completion |
| 3 | Baseline benchmark, no platform | 18 | TTFT, tokens/sec, GPU util recorded — **the number everything else is compared against** |
| 4 | Gateway: auth + passthrough streaming | 1, 2 | OpenAI SDK works against it; overhead measured |
| 5 | Usage metering + ledger | 8, 9 | Token counts land in Postgres and reconcile with vLLM's own counts |
| 6 | Budgets + rate limits | 3, 4 | Over-budget tenant is rejected with zero GPU work |
| 7 | KServe + model registry | 10, 11, 13 | Two models served; canary rollout works |
| 8 | Queue-depth autoscaling via KEDA | 12 | Load test triggers scale-up within window |
| 9 | Routing + fallback | 5, 6 | Primary killed mid-load; fallback serves and reports it |
| 10 | Semantic cache, per-tenant namespaced | 7 | Hit rate measured; cross-tenant isolation test passes |
| 11 | Full observability + dashboards | 17 | Every README metric is live in Grafana |
| 12 | Graceful drain + chaos | 14 | Node killed under load; no truncated streams |
| 13 | Capacity report + break-even | 18 | `docs/benchmarks.md` complete, no estimated cells |

Row 3 is the one people skip. Without a pre-platform baseline, you cannot say what the gateway costs you, and "gateway overhead p99 < 10ms" becomes an assertion rather than a measurement.

---

## 12. Open question — resolve before row 1

**What GPU access is available?** The answer changes the shape of this project:

| Access | Implication |
|---|---|
| Owned/dedicated GPU on K8s | Full build order as written |
| Cloud GPU, spot/preemptible | Row 1 must handle preemption; adds real engineering, and it is *good* engineering to demonstrate |
| A few rented hours (Runpod, Lambda, Vast) | Benchmarks are real but capacity claims must be scoped honestly to the hardware tested |
| CPU only | Use a small model (1–3B) — the platform engineering is identical, the numbers are just smaller. **Still worth building; state the hardware.** |

The last row matters: the platform work — gateway, budgets, routing, cache, autoscaling, observability — is the hireable part, and none of it requires an A100. Do not defer this project waiting for hardware.
