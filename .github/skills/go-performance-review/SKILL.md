---
name: go-performance-review
description: "Review Go code changes for hot-path regression risk, performance-budget conformance, allocation and contention impact, and measurement-evidence quality."
---

# Go Performance Review

## Purpose
Protect changed hot paths from measurable latency, throughput, allocation, contention, and work-amplification regressions.

## Scope
- review changed code against explicit budgets or other approved performance expectations
- review algorithmic cost, per-request work, allocation behavior, and contention signals
- review I/O, DB, and cache efficiency when they affect performance
- review benchmark, profile, and trace evidence quality
- review missing mandatory evidence when the change is performance-sensitive

## Boundaries
Do not:
- block on performance taste or speculative micro-optimizations
- optimize by intuition without evidence
- take primary ownership of concurrency correctness, DB/cache correctness, or reliability policy when performance is only the symptom
- increase code complexity unless measured benefit justifies it

## Core Defaults
- Evidence first.
- Review changed and directly impacted hot paths first.
- Simpler code wins unless measured gains justify complexity.
- Treat unbounded work growth, unbounded queues, and retry-amplified load as defects until disproven.
- Prefer the smallest safe correction that restores budget conformance or narrows the evidence gap.

## Expertise

### Budget Conformance
- Validate changed paths against explicit latency, throughput, allocation, or contention expectations when they exist.
- Flag high-risk hot-path changes that lack the evidence needed to judge them.
- Treat missing budget context itself as a risk when the change clearly affects a critical path.

### Hot-Path And Work Amplification
- Flag algorithmic regressions, nested scans, repeated serialization, repeated dependency calls, and payload amplification in hot flows.
- Flag expensive work moved into request-critical paths without measured benefit.
- Treat hidden per-item work in list or fan-out paths as high risk when scale is meaningful.

### Benchmark, Profile, And Trace Evidence
- Require benchmark evidence for localized claims about speed or allocations.
- Require profiles when the real bottleneck is uncertain.
- Require trace evidence when scheduler behavior, blocking, or tail latency is the real concern.
- Treat microbenchmarks as insufficient for end-to-end claims unless paired with broader evidence.
- Require reproducibility basics: realistic input, clear before/after comparison, and stable measurement setup.

### Allocation And Memory Pressure
- Flag allocation growth in hot loops when evidence shows it matters.
- Prefer structural fixes over syntax tricks.
- Treat pooling and buffer reuse as suspicious unless profiling proves they help.
- Flag retention patterns that increase GC pressure without benefit.

### Contention And Scheduler Cost
- Flag unbounded concurrency or queue growth on performance-critical paths.
- Flag lock contention patterns likely to increase tail latency.
- Require bounded concurrency and explicit cancellation around blocking work.
- Keep performance ownership focused on cost and latency impact; hand off concurrency-correctness depth when needed.

### I/O, DB, And Cache Efficiency
- Flag DB round-trip amplification, query-in-loop patterns, and deep pagination or serialization costs when they affect latency.
- Reject cache “optimizations” that lack measured bottleneck evidence.
- Require cache-related performance ideas to include stampede controls and bounded fallback behavior when origin pressure matters.
- Review whether API shape creates deterministic latency cliffs or payload amplification.

### Overload And Failure Interaction
- Flag retry behavior that amplifies load or worsens tail latency.
- Flag missing explicit deadlines in performance-sensitive dependency chains.
- Require backpressure, shedding, or degraded behavior to stay explicit when they are part of the performance safety story.
- Keep performance review separate from full resilience policy ownership.

### Cross-Domain Handoffs
- Hand off race, deadlock, and lifecycle correctness to `go-concurrency-review`.
- Hand off DB/cache correctness and consistency depth to `go-db-cache-review`.
- Hand off overload, retry, and degradation policy depth to `go-reliability-review`.
- Hand off contract or payload semantics depth to `go-design-review` or the contract owner.
- Hand off benchmark or harness test-shape depth to `go-qa-review` when the issue is really test methodology.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the risk dimension (`latency`, `throughput`, `allocations`, `contention`, or `I/O`)
- the evidence type or the required missing evidence
- the concrete impact
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: proven severe regression or missing mandatory evidence on a high-risk hot-path change
- `high`: strong evidence of meaningful performance regression risk
- `medium`: bounded but notable performance weakness
- `low`: local improvement opportunity

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-performance-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
Escalate when:
- safe correction changes budgets, hot-path architecture, or performance trade-offs at design level (`go-performance-spec` or `go-design-spec`)
- the right answer requires new cache, query, or consistency decisions (`go-db-cache-spec`)
- overload, retry, or degraded-mode policy must change (`go-reliability-spec`)
- latency shape depends on API contract or async design changes (`api-contract-designer-spec` or `go-distributed-architect-spec`)
