---
name: go-performance-review
description: "Use when a Go diff affects hot paths, batching, serialization, fan-out, query count, caching, allocation, contention, `sync.Pool`, or benchmark/profile evidence; Own measurable performance-regression risk and proof quality; Skip when budgets are unset or correctness belongs to concurrency, DB/cache, reliability, or domain review."
---

# Go Performance Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Trigger, Scope, And Boundary

Review changed request/hot paths, loops, copies, parsing/serialization, batching, fan-out, queueing, locks, allocations/GC, DB/cache/dependency calls, retries/fallbacks, and benchmark/profile/trace evidence for material latency, throughput, allocation, contention, I/O, capacity, or tail-risk regressions.

Stay evidence-first and review-only. Do not block on micro-optimization taste, accept complex code because it “feels faster,” generalize a microbenchmark to end-to-end claims, or take primary ownership of concurrency, DB/cache correctness, resilience, architecture, or API semantics.

## Performance Invariants

1. Performance claims name the workload, scale dimension, budget/baseline, environment, metric, and fresh comparative evidence; missing mandatory hot-path proof is a review result.
2. Review the changed path and direct dependencies first; distinguish constant cost from per-item, per-request, per-retry, per-tenant, or fan-out amplification.
3. Structural work reduction—fewer round trips, copies, parses, allocations, serial sections, or retries—precedes syntax-level tuning and pooling.
4. Locks and blocking I/O do not create hidden serialization; active work, queues, fan-out, retries, fallback, and memory retention remain bounded and cancellation-aware.
5. Benchmarks measure the relevant work with stable fixtures, timer boundaries, allocation reporting, repetition, and practical as well as statistical significance.
6. Profiles and traces match the question: CPU for active work, allocs/heap for churn/retention, block/mutex/trace for waiting/contention/scheduler timelines.
7. Added optimization complexity stays only when measured benefit justifies its ownership, reset/retention risk, and proof burden; otherwise keep the simpler implementation.

## Symptom-Driven Reference Selector

State which evidence or correction choice the selected reference will change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| A performance claim or blocker lacks a dominant narrower evidence question. | [performance-evidence-quality.md](references/performance-evidence-quality.md) | Write the exact missing proof or residual risk instead of accepting “faster” or demanding broad load tests reflexively. |
| Go benchmark or `benchstat` evidence is added or relied on. | [benchmark-and-benchstat-review.md](references/benchmark-and-benchstat-review.md) | Check workload, timer, `-benchmem`, repetition, and practical significance instead of trusting any table. |
| CPU, heap, allocs, goroutine, block, mutex, or live pprof artifact is central. | [pprof-and-profile-selection.md](references/pprof-and-profile-selection.md) | Match profile type and workload to the symptom. |
| Locks, channels, queues, fan-out/fan-in, or scheduler wait affects tail latency. | [trace-block-mutex-and-contention.md](references/trace-block-mutex-and-contention.md) | Ask for timeline/block/mutex evidence and smallest wait reduction instead of generic worker pools. |
| Loops, copies, serialization, batching, fan-out growth, or repeated transforms change. | [hot-path-cost-model.md](references/hot-path-cost-model.md) | Name the scaling dimension and structural fix instead of micro-optimization folklore. |
| DB/cache/query count, dependency calls, pagination, or I/O-in-loop amplifies the request path. | [db-cache-and-io-amplification.md](references/db-cache-and-io-amplification.md) | Quantify round trips and hand correctness ownership off instead of treating all DB/cache concerns as performance. |
| Allocation churn, GC, buffer reuse, retained backing arrays, or `sync.Pool` changes. | [allocation-gc-and-syncpool-review.md](references/allocation-gc-and-syncpool-review.md) | Require allocation/retention/reset evidence instead of pooling by default. |
| Retries, fallback, admission, queueing, or deadline behavior changes on a hot path. | [retry-overload-and-tail-latency.md](references/retry-overload-and-tail-latency.md) | Identify amplification and tail collapse while handing policy ownership to reliability. |

## Evidence And Domain Finding Rules

Each finding adds the dominant axis (`latency`, `throughput`, `allocations`, `contention`, `I/O`, or `evidence`), concrete regression or proof gap, observed or required evidence, scaling impact, and the narrow command/artifact that would prove it. Use the approved budget/reference when one exists; otherwise say `N/A` rather than inventing a target.

`critical` requires a proven severe regression or missing mandatory proof on a clearly high-risk path; `high` requires strong evidence of meaningful regression or unbounded amplification. Start `Issue` with `Axis:` when useful.

Choose validation from the symptom: focused `go test -bench ... -benchmem`, repeated benchmark plus `benchstat`, CPU/memory/block/mutex profiles, `go tool trace`, query-count/integration/load evidence, or an explicit missing-proof statement. Do not dump a generic command catalog into every review.

## Success, Escalation, And Stop Conditions

Success means findings are workload- and evidence-specific, merge-risk ordered, proportional to actual scale, and recommend the smallest proven correction or exact proof gap.

Escalate when the safe correction changes performance budgets, architecture/batching, cache/query/consistency policy, retry/admission/degradation, public async/pagination/payload semantics, or distributed workflow. Stop instead of prescribing an optimization whose benefit or ownership cannot be proven.
