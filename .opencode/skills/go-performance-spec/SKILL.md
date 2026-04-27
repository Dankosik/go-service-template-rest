---
name: go-performance-spec
description: "Design performance-first specifications for Go services. Use when planning or revising latency, throughput, allocation, contention, or capacity behavior and you need explicit hot-path budgets, benchmark/profile/trace acceptance criteria, and performance risk controls before coding. Skip when the task is a local code fix, endpoint-only API payload design, schema-only modeling, CI/container setup, or low-level micro-optimization implementation."
---

# Go Performance Spec

## Purpose
Turn performance intent into measurable, reproducible pre-coding contracts for latency, throughput, allocation, contention, and capacity.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Specialist Stance
- Treat performance as budgets, bottleneck hypotheses, and measurement plans, not generic speed advice.
- Decide what must be fast, at what percentile or throughput, under what input shape, and with what proof.
- Prefer bounded, observable performance risks over speculative micro-optimization or tool-led rewrites.
- Hand off API, data, reliability, and concurrency design when performance depends on those primary decisions.
- If another domain is only affected, return the consequence as `constraint_only`, `proof_only`, or explicit `no new decision required` instead of widening the design.

## Scope
Use this skill to define or review latency, throughput, allocation, contention, and capacity behavior, including hot-path budgets, measurement protocol, acceptance thresholds, and production validation signals.

## Operating Loop
1. Frame the affected operation class, workload, hot path, and user-visible or system-visible performance objective.
2. Load at most one reference by default from the selector below. Load more only when the task clearly spans independent decision pressures, such as memory budgets plus overload semantics.
3. Compare viable options only when a real `live fork` exists before selecting a performance contract. Keep unproven numeric targets marked as assumptions.
4. Write section-ready spec content with budgets, workload shape, selected and rejected options, measurement protocol, thresholds, runtime telemetry, and rollout checkpoints.
5. Stop at the pre-coding boundary. Do not drift into low-level optimization, implementation review, or benchmark result interpretation without a specification decision to support.

## Reference Files
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load a reference only when its behavior-change thesis matches the symptom in the prompt. If you cannot name the likely mistake it prevents, do not load it. If a narrower positive reference matches, prefer it over a broad neighboring reference.

| Symptom | Behavior Change | Load |
| --- | --- | --- |
| The request says "make X faster" or lacks a budgeted hot path | Choose operation budgets, component reserve, and a measurable bottleneck hypothesis instead of generic speed goals or dashboard averages | `references/budget-modeling-and-hot-path-maps.md` |
| The proof depends on traffic mix, tenant skew, cache state, dependency state, or fixture size | Choose representative workload buckets and labels instead of median-only or toy fixtures | `references/workload-profile-and-input-shape.md` |
| The spec must decide what kind of proof is sufficient | Choose a symptom-matched measurement protocol and variance rule instead of a familiar microbenchmark or single best run | `references/measurement-protocols.md` |
| The proof type is known but the spec needs concrete Go benchmark/profile/trace commands | Write executable proof obligations and avoid benchmark/profile traps instead of vague "run benchmarks" instructions | `references/benchmark-profile-and-trace-plans.md` |
| The proposed optimization class is under-justified or disproportionately complex | Choose the least-complex option that can meet the budget instead of reaching for cache, PGO, pooling, or fan-out by habit | `references/option-selection-and-complexity-bounds.md` |
| The performance idea adds fan-out, queues, workers, locks, or capacity changes | Require bounded concurrency, saturation signals, and cancellation/deadline proof instead of "more goroutines" or "more workers" | `references/concurrency-contention-and-capacity.md` |
| The bottleneck crosses DB, cache, pagination, retry, or API-visible behavior | Surface contract handoffs and fallback budgets instead of hiding semantic changes inside "performance" | `references/db-cache-api-performance-contracts.md` |
| The decision needs canary, rollback, runtime telemetry, or production validation | Tie rollout gates to budget metrics and actions instead of "watch dashboards" | `references/runtime-telemetry-and-rollout-checkpoints.md` |
| The risk is allocation rate, live heap, GC CPU, GOGC/GOMEMLIMIT, or container memory | Specify memory envelopes and GC trade-offs instead of vague "reduce allocations" or unsafe tuning | `references/memory-allocation-and-gc-budgets.md` |
| The proposal involves PGO, default.pgo, profile merging, profile freshness, or source skew | Require representative CPU profile lifecycle and rollback checks instead of enabling PGO because it "usually helps" | `references/pgo-profile-lifecycle.md` |
| The performance envelope depends on overload, shedding, degraded results, queues, retries, or tenant fairness | Define capacity-protection semantics and retry limits instead of best-effort overload behavior | `references/overload-backpressure-and-load-shedding.md` |
| The spec must connect latency to SLI/SLO, histograms, aggregation windows, or error-budget risk | Use percentile/window/label-aware thresholds instead of averages or copied dashboard values | `references/latency-sli-slo-and-histogram-thresholds.md` |
| Payload shape, JSON work, response size, streaming, flushing, or large-body behavior dominates | Bound representation and streaming semantics instead of optimizing serialization around unbounded payloads | `references/payload-serialization-and-streaming-budgets.md` |

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
- whether a real `live fork` exists
- when a `live fork` exists, the viable options, the selected option, and at least one explicit rejection reason
- measurement protocol and thresholds
- latency/throughput/allocation/complexity/cost trade-offs
- only the downstream architecture, API, data/cache, reliability, observability, or delivery effects that force a new decision, handoff, or proof obligation now
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the performance spec or review, cover:
- hot-path map
- budget decomposition by operation class
- bottleneck hypotheses
- benchmark/profile/trace plan
- acceptance thresholds and pass/fail rules
- performance-sensitive rollout and rollback checkpoints
- downstream decision/proof consequences only when another domain must act now; otherwise use `no new decision required in <domain>`

## Escalate Or Reject
- affected critical paths without explicit budgets
- no reproducible measurement protocol
- performance claims based only on anecdote or microbenchmark
- concurrency-sensitive designs without bounded-concurrency and validation plans
- DB/cache-heavy optimization without query/cache constraints and fallback implications
- API-visible performance behavior changed without explicit contract impact
- no runtime telemetry path to detect and validate the claimed improvement
- critical performance ambiguity deferred to coding
