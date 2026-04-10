---
name: go-performance-spec
description: "Design performance-first specifications for Go services. Use when planning or revising latency, throughput, allocation, contention, or capacity behavior and you need explicit hot-path budgets, benchmark/profile/trace acceptance criteria, and performance risk controls before coding. Skip when the task is a local code fix, endpoint-only API payload design, schema-only modeling, CI/container setup, or low-level micro-optimization implementation."
---

# Go Performance Spec

## Purpose
Turn performance intent into measurable, reproducible pre-coding contracts for latency, throughput, allocation, contention, and capacity.

## Specialist Stance
- Treat performance as budgets, bottleneck hypotheses, and measurement plans, not generic speed advice.
- Decide what must be fast, at what percentile or throughput, under what input shape, and with what proof.
- Prefer bounded, observable performance risks over speculative micro-optimization or tool-led rewrites.
- Hand off API, data, reliability, and concurrency design when performance depends on those primary decisions.

## Scope
Use this skill to define or review latency, throughput, allocation, contention, and capacity behavior, including hot-path budgets, measurement protocol, acceptance thresholds, and production validation signals.

## Boundaries
Do not:
- optimize by instinct, anecdote, or microbenchmark alone
- recommend concurrency or caching changes without boundedness, correctness, and fallback implications
- reduce performance work to code-level tricks before budgets and bottlenecks are explicit
- leave runtime validation and reproducibility undefined

## Escalate When
Escalate if critical paths lack budgets, workload assumptions are missing, measurement cannot be reproduced, or proposed improvements materially affect API, data/cache correctness, reliability, or observability contracts.

## Core Defaults
- Measure first: no optimization decision is valid without a metric target and evidence protocol.
- Prefer algorithmic, data-flow, and round-trip reductions before micro-level tuning.
- Keep complexity proportional to verified bottlenecks; keep the simpler option when gains are unproven.
- Treat missing workload, budget, or measurement facts as explicit assumptions or blockers.
- Preserve compatibility and operational safety across mixed-version rollout.

## Expertise

### Budget Modeling
- Require explicit budgets for each changed critical path:
  - latency percentiles such as `p95` and `p99`
  - throughput or concurrency target
  - allocation or memory constraints
  - CPU/contention bounds where relevant
- Decompose budgets across `api -> domain -> db/cache -> outbound dependency` so debt is visible.
- Tie budgets to user-visible or system-visible outcomes; avoid global averages as primary acceptance metrics.
- For async flows, include processing latency, lag/backlog, retry, and DLQ impact budgets.

### Workload And Hot-Path Normalization
- Define workload profile before choosing options:
  - request or message shape
  - cardinality and skew
  - concurrency level
  - data distribution and hot-key behavior
- Normalize warm vs cold paths, peak vs steady load, cache-up vs cache-down, and degraded dependency behavior.
- Maintain one authoritative hot-path map per affected operation with a bottleneck hypothesis and ownership.
- Reject conclusions based on toy inputs or non-representative traffic assumptions.

### Measurement Protocol
- Every major recommendation needs a reproducible measurement protocol:
  - benchmark/profile/trace type
  - environment/runtime class
  - dataset shape and scale
  - baseline and target thresholds
  - pass/fail rule
- Keep before/after comparisons fair: same workload class, stable environment, repeated runs, and basic variance sanity checks.
- Microbenchmarks alone are insufficient for system-level claims; combine them with profile, trace, or scenario-level evidence.
- Use trace planning when scheduler, blocking, locking, or fan-out behavior matters.

### Benchmarking And Profiling
- Use `go test -bench` for focused hot paths and include `-benchmem` when allocation matters.
- Keep setup outside timed loops.
- Use the right profile for the symptom: CPU, heap, allocs, mutex, block, or goroutine.
- Profile before and after nontrivial performance changes.
- Optimize measured bottlenecks, not code that merely looks expensive.
- Consider PGO only after representative CPU profiling and validated bottleneck analysis.

### Concurrency And Contention
- For concurrency-sensitive paths, make goroutine fan-out bounds, queue/channel limits, lock hotspots, and cancellation/shutdown behavior explicit.
- Require bounded parallelism with explicit limits when concurrency is introduced for speed.
- Require race-aware validation when concurrency changes are material.
- Treat unbounded concurrency, ignored blocking behavior, or missing cancellation as performance-spec blockers.

### DB, Cache, And API Performance
- Make DB round-trip budget, query-shape limits, pool assumptions, and timeout/deadline expectations explicit.
- For cache-related performance changes, define cacheability class, hit-ratio expectation, stampede protection, and cache-down fallback behavior.
- Align with data/cache correctness constraints whenever performance choices change freshness or consistency.
- When performance is API-visible, make payload limits, pagination defaults, idempotency/retry semantics, and honest async behavior explicit.
- Include overload/backpressure outcomes such as `429` or `503` when shedding or degradation is part of the envelope.

### Observability And Delivery Alignment
- Map performance acceptance to runtime telemetry: RED metrics, saturation signals, and trace/log correlation on critical paths.
- Tie user-facing performance objectives to SLI/SLO expectations when relevant.
- Define the minimum production diagnostics needed to validate budgets.
- Translate verification into executable benchmark/profile/trace obligations and rollout checkpoints.
- Treat missing reproducible validation paths as a decision-quality defect.

## Decision Quality Bar
For every major performance recommendation, include:
- the target operation and workload
- bottleneck hypothesis and baseline assumptions
- at least two viable options
- the selected option and at least one explicit rejection reason
- measurement protocol and thresholds
- latency/throughput/allocation/complexity/cost trade-offs
- cross-domain impact on architecture, API, data/cache, reliability, observability, and delivery
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the performance spec or review, cover:
- hot-path map
- budget decomposition by operation class
- bottleneck hypotheses
- benchmark/profile/trace plan
- acceptance thresholds and pass/fail rules
- performance-sensitive rollout and rollback checkpoints

## Escalate Or Reject
- affected critical paths without explicit budgets
- no reproducible measurement protocol
- performance claims based only on anecdote or microbenchmark
- concurrency-sensitive designs without bounded-concurrency and validation plans
- DB/cache-heavy optimization without query/cache constraints and fallback implications
- API-visible performance behavior changed without explicit contract impact
- no runtime telemetry path to detect and validate the claimed improvement
- critical performance ambiguity deferred to coding
