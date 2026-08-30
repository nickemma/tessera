# Progress

**Paste the `POSITION` block at the start of every session.**

---

## POSITION

```
Project:    1 of 5 — TESSERA v0
Milestone:  10 of 10 — Measured and permanent
Building:   real CPU model measurement when a runtime/weight is supplied
Last built: dependency-backed Compose stack with PostgreSQL, Redis, mock model, expanded chaos checks, latency metrics, and published control-plane sample
Uptime:     not live (target: week 10)
```

---

## TESSERA v0 — weeks 1–10

```
[█████░░░░░] 5/10 milestones
```

| # | Milestone | What runs when it's done | Weeks | Status |
|---|---|---|---|---|
| 1 | A server that answers | `POST /v1/chat` responds | 1 | ✓ |
| 2 | It knows who you are | API keys, tenants, auth | 1–2 | ✓ |
| 3 | It won't let you overspend | budget rejection before the model | 2–3 | ✓ |
| 4 | It talks to a real model | streamed tokens from a local model | 3–4 | △ |
| 5 | It survives being hammered | 2k rps, rejects cleanly, no leaks | 4–5 | △ |
| 6 | It tells you what it's doing | dashboard + graceful shutdown | 5–6 | △ |
| 7 | It remembers what it cost | `/v1/usage` returns real numbers | 6–7 | ✓ |
| 8 | It doesn't care where it runs | 15MB image, `compose up` | 7–8 | ✓ |
| 9 | It survives you | failure day, all 8 failures handled | 8–9 | △ |
| 10 | Measured and permanent | benchmarks published, running for good | 10 | → |

---

## The five projects

```
P1  TESSERA v0    weeks  1–10   [░░░░░░░░░░]  0/10  ← HERE
P2  LATTICE       weeks 11–26   [░░░░░░░░░░]
P3  PLINTH        weeks 27–34   [░░░░░░░░░░]
P4  TESSERA v1    weeks 35–46   [░░░░░░░░░░]
P5  SYNAPSE-AI    weeks 47–52   [░░░░░░░░░░]
```

---


## Build log

One line per session. What runs now that didn't before.

```
2026-08-30  M1  Clean server foundation and canned inference endpoint
2026-08-30  M7  PostgreSQL/Redis adapters, mock model, Compose, e2e script, and playground
2026-08-30  M9  Redis/model chaos exercises and recovery-aware e2e test
2026-08-30  M10 Load runner sample against dependency-backed Compose stack
2026-08-30  M6/M9 Latency and TTFT metrics; PostgreSQL outage and SIGTERM recovery checks
```

---

## Tradeoffs decided

The record of why things are the way they are. This is what you'll be asked about.

| Milestone | Chose | Over | Because |
|---|---|---|---|
| | | | |

---

## Failures survived

Root cause must be a decision, not an event.

| Date | What broke | Root cause | Fixed by |
|---|---|---|---|
| | | | |

---

## Debt

Flagged and deferred. Reviewed at each milestone.

```
(empty)
```

---

## Published

| Date | Title | Where |
|---|---|---|
| | | |
