---
name: go-concurrency-review
description: "Review Go code changes for goroutine lifecycle, cancellation, channel ownership, shared-state synchronization, bounded concurrency, and shutdown safety."
---

# Go Concurrency Review

## Purpose
Protect changed concurrent paths from race, deadlock, leak, shutdown, and unbounded-concurrency defects.

## Scope
- review goroutines, channels, mutexes, wait groups, `errgroup`, worker pools, pipelines, fan-out or fan-in paths
- review goroutine lifecycle and explicit termination behavior
- review cancellation, deadline propagation, and blocking-operation escape paths
- review channel ownership, close semantics, queue bounds, and send or receive behavior
- review synchronization for shared mutable state
- review overload, backpressure, and bounded-concurrency behavior
- review concurrent error propagation and shutdown safety
- review race-evidence expectations for significant concurrent changes

## Boundaries
Do not:
- turn concurrency review into broad style cleanup or architecture redesign
- take primary ownership of benchmark proof, DB/cache correctness, or resilience policy unless concurrency is the direct root cause
- accept timing luck, sleep-based reasoning, or scheduler luck as proof of correctness
- hide uncertain concurrent behavior behind vague “seems safe” language

## Core Defaults
- Scheduling-dependent correctness is a defect until explicit synchronization proves otherwise.
- Every goroutine needs a clear completion, cancellation, or shutdown path.
- Prefer explicit ownership of channels, locks, and shared state.
- Prefer bounded concurrency, bounded queues, and explicit backpressure over open-ended fan-out.
- Prefer the smallest safe fix that restores deterministic concurrent behavior.

## Expertise

### Goroutine Lifecycle
- Every started goroutine must have a reason to stop.
- Flag fire-and-forget goroutines unless process-lifetime ownership and failure irrelevance are explicit.
- Verify downstream early exit cannot strand senders or workers indefinitely.
- Require timers, tickers, and background loops to stop cleanly.

### Cancellation And Deadline Semantics
- Require `context.Context` to drive coordinated cancellation.
- Flag request-path replacement of request context with `context.Background()`.
- Require blocking operations to have a cancel or timeout path.
- Prefer `errgroup.WithContext` for fail-fast sibling work when one failure should stop the group.

### Channel Ownership And Closure
- Make closer ownership explicit, usually on the sending side.
- Flag multiple possible closers, double-close risk, and send-on-closed risk.
- Treat buffered channels as bounded queues with intentional capacity, not silent backlog.
- Require explicit behavior for full queues and abandoned receivers.

### Shared State Synchronization
- Flag unsynchronized shared mutation, including concurrent map access.
- Keep critical sections small and obvious.
- Prefer `sync.Mutex` by default; use more complex locking only when justified.
- Require clear happens-before guarantees rather than inference from timing.

### Bounded Concurrency And Backpressure
- Flag unbounded goroutine growth, unbounded worker pools, and unbounded queue growth.
- Require concurrency lanes to critical dependencies to stay isolated and bounded.
- Prefer fast failure or bounded waiting over silent accumulation.
- For hot cache or origin paths, require coalescing or other stampede controls where relevant.

### Deadlock And Shutdown Safety
- Review lock ordering, channel waits, and `Wait` semantics for cyclic wait risk.
- Verify shutdown can unblock sends, receives, waits, and workers.
- Reject sleep loops or polling hacks used instead of synchronization design.
- Treat indefinite waits during shutdown as merge-unsafe.

### Error Propagation And Worker Semantics
- Errors from workers must be observable by the caller or control plane.
- Require explicit policy for first-error cancel, sibling handling, and aggregation.
- For async consumers, require durable side effects before ack or commit.
- Flag infinite or unclassified retry loops in concurrent workers.

### Concurrent DB And Cache Touchpoints
- Verify concurrent DB work preserves transaction discipline and does not hold long transactions across network calls.
- Verify concurrent cache fallback paths remain bounded and do not amplify origin pressure.
- Hand off primary DB/cache correctness depth when concurrency is only the symptom surface.

### Evidence And Validation
- Significant concurrent changes should carry race evidence, repository-equivalent race tests, or an explicit evidence gap.
- Prefer deterministic tests over sleep-based “eventual pass” checks.
- Missing race evidence on meaningful concurrency changes should become either a finding or a residual risk.

### Cross-Domain Handoffs
- Hand off latency, contention, and benchmark proof questions to `go-performance-review`.
- Hand off primary timeout, overload, and degradation policy defects to `go-reliability-review`.
- Hand off DB/query/cache contract defects to `go-db-cache-review`.
- Hand off coverage and determinism gaps in tests to `go-qa-review`.
- Hand off broader structural redesign needs to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the failed concurrency axis
- the concrete failure mode
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed deadlock, leak, race, or shutdown hang in a significant path
- `high`: high-probability concurrency defect or unbounded-concurrency risk
- `medium`: bounded but meaningful concurrency weakness
- `low`: local hardening or clarity improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-concurrency-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

In `Issue`, start with the axis context, for example `Axis: Goroutine Lifecycle; ...`.

## Escalate When
Escalate when:
- safe correction requires a new concurrency model, bounded-work policy, or shutdown contract (`go-reliability-spec`)
- the fix depends on a new async workflow, durable state machine, or coordination design (`go-distributed-architect-spec`)
- correctness depends on new DB/cache ownership or cache-coalescing contract (`go-db-cache-spec`)
- the current package or ownership boundaries make local concurrency repair unsafe (`go-design-spec`)
