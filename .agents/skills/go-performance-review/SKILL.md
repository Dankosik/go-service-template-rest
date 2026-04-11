---
name: go-performance-review
description: "Review Go code changes for hot-path regressions, latency/throughput/allocation/contention risk, and benchmark/pprof/trace evidence quality. Use whenever a Go PR, diff, incident fix, or optimization touches hot paths, batching, serialization, fan-out, caching, query count, sync.Pool, pprof, benchstat, or tail-latency behavior, even if the user frames it as a generic code review."
---

# Go Performance Review

## Purpose
Protect changed Go hot paths from measurable latency, throughput, allocation, contention, and work-amplification regressions, and reject performance claims that are not backed by the right kind of evidence.

## Specialist Stance
- Review performance from hot-path shape and evidence, not from folklore or micro-optimization taste.
- Prioritize query count, allocation churn, serialization, batching, fan-out width, lock contention, and retry amplification where they move user-visible latency or capacity.
- Require benchmark, profile, trace, or focused command evidence proportional to the claim.
- Hand off design, concurrency, DB/cache, or reliability ownership when the performance symptom needs a different primary fix.

## Scope
- review changed request paths, fan-out or fan-in paths, loops, serialization, batching, queueing, locking, and outbound I/O that can move latency or throughput
- review whether the diff still fits approved performance budgets or clearly stated hot-path expectations
- review benchmark, `pprof`, trace, load-test, query-count, and cache-hit evidence quality
- review missing mandatory evidence when the change is high-risk or complexity-increasing
- review performance-visible API shape when synchronous behavior, pagination, payload size, or fallback strategy creates deterministic latency cliffs

## Lazy Reference Loading
Keep this `SKILL.md` as the review routing guide. Load examples from `references/` only when the diff touches that surface or you need a sharper evidence pattern:

- `references/performance-evidence-quality.md` - claim-to-evidence fit, missing proof findings, residual-risk wording, and proof gaps for local vs service-level performance claims.
- `references/benchmark-and-benchstat-review.md` - benchmark methodology, `testing.B`, `B.Loop`, `-benchmem`, repeated runs, benchstat interpretation, and noisy or non-representative benchmark findings.
- `references/pprof-and-profile-selection.md` - CPU, heap, allocs, goroutine, block, and mutex profile selection; live `net/http/pprof` collection; and pprof evidence review.
- `references/trace-block-mutex-and-contention.md` - execution trace, block and mutex profiles, scheduler stalls, lock contention, queueing, and fan-out/fan-in tail-latency evidence.
- `references/hot-path-cost-model.md` - loops, asymptotic regressions, repeated encode/decode work, copy amplification, serialization, batching, and fan-out cost models.
- `references/db-cache-and-io-amplification.md` - `N+1`, query count, DB round trips, cache miss/fallback amplification, dependency timing, and request-path I/O proof.
- `references/allocation-gc-and-syncpool-review.md` - allocation churn, GC pressure, heap vs allocs profiles, runtime metrics, `sync.Pool` review, and buffer reuse evidence.

Use the reference examples to shape local findings, not to invent blockers. Prefer the smallest evidence-backed correction and escalate when the performance fix changes architecture, data ownership, retry policy, or API semantics.

## Boundaries
Do not:
- invent performance blockers from style preference or micro-optimization taste
- accept complex optimizations because they "feel faster"
- take primary ownership of concurrency correctness, DB/cache correctness, or resilience policy when performance is only the symptom surface
- treat one tiny benchmark as proof of end-to-end improvement
- ask for invasive redesign when a smaller safe fix or evidence request is enough

## Core Defaults
- Evidence first. No guess-driven blocking calls.
- Review the changed hot path and directly impacted dependencies before adjacent code.
- Keep the simpler implementation unless measured benefit justifies extra complexity.
- Treat unbounded work growth, queue growth, retry amplification, and lock-held-across-I/O as defects until disproven.
- Missing proof on a high-risk hot-path change is itself a review result, not a neutral state.

## Expertise

### Budget And Spec Conformance
- Validate changed paths against explicit latency, throughput, allocation, GC, or contention expectations when they exist.
- When a budget exists, compare the diff against that budget before suggesting local tuning.
- When no budget exists, still judge local merge risk and state that the budget reference is missing.
- Treat complexity-increasing changes on a critical path as unproven until the evidence clears them.

### Symptom-To-Evidence Selection
Use the smallest evidence that actually fits the symptom:
- localized CPU or allocation claim: `go test -run '^$' -bench ... -benchmem` plus before or after `benchstat`
- unclear CPU bottleneck: CPU profile
- allocation or GC pressure claim: `-benchmem` plus heap or allocs profile
- lock wait, blocking, queue buildup, or fan-out stall: mutex profile, block profile, and often `go tool trace`
- scheduler behavior, wakeups, goroutine bursts, or p99 spikes: `go tool trace`, optionally paired with block or mutex profiles
- request-path DB/cache/API latency cliff: representative request or load evidence, query-count evidence, dependency timings, or cache hit/miss data
- service-level latency claim: do not clear it with a microbenchmark alone

### Benchmark And Measurement Quality
- Require baseline-vs-current comparison, not an isolated "it seems fast now" number.
- Prefer `benchstat` over eyeballing a single benchmark run.
- Expect `-benchmem` when allocations or GC are part of the claim.
- Prefer repeated runs with stable variance; do not treat one lucky run or a noisy delta as proof of improvement.
- Check that setup is outside the timed loop when practical, inputs are realistic, and tiny toy data is not being used to justify production-path complexity.
- Watch for weak methodology: benchmarking mocks only, measuring constructor or setup inside the loop, no result sink on pure code, or no explanation of workload shape.
- Treat benchmark evidence as local proof; require broader evidence before approving end-to-end claims.

### Hot-Path Cost Model
- Flag asymptotic regressions, nested scans, query-in-loop patterns, repeated dependency calls, and repeated parse/encode/decode work in hot flows.
- Flag per-item work that scales with list size or fan-out width: repeated `json.Marshal` or `json.Unmarshal`, `fmt.Sprintf`, regex or time parsing, compression, hashing, or template rendering inside hot loops.
- Flag hidden copies that raise allocation cost without user value: `[]byte` to `string`, `string` to `[]byte`, map or slice cloning, or repeated buffer materialization.
- Treat payload amplification, over-fetching, and fan-out multiplication as throughput and latency risks, not just "style".

### Allocation And GC Pressure
- Look for hot-loop heap growth, temporary object churn, buffer churn, and retained large backing arrays.
- Prefer structural fixes such as batching, reducing copies, or changing data flow before suggesting syntax-level rewrites.
- Treat `sync.Pool`, manual buffer reuse, and object recycling as suspicious until profiling shows that allocations are the real bottleneck and the pool actually helps.
- Flag reuse patterns that keep oversized buffers alive, skip reset discipline, or trade away clarity for an unproven win.

### Contention And Scheduler Cost
- Flag locks held across network, disk, DB, cache, or other blocking operations.
- Flag serial bottlenecks that turn parallel request traffic into one shared critical section.
- Flag goroutine-per-item fan-out, open-ended worker creation, or implicit queues that can grow with input size.
- Require bounded concurrency and cancellation-aware blocking work on hot paths.
- Treat queue wait, mutex wait, wakeup storms, and scheduler churn as tail-latency risks even when average latency looks acceptable.

### I/O, DB, Cache, And API Latency
- Flag `N+1`, query-in-loop, repeated identical reads, deep `OFFSET` pagination, and hot-path round-trip amplification.
- Reject cache additions that are not tied to a measured bottleneck or that ignore stampede and fallback behavior.
- Flag request handlers that repeatedly serialize, deserialize, or transform the same payload when one materialization would do.
- Flag API shapes that create deterministic latency cliffs: huge synchronous uploads, synchronous long-running work, or pagination or filter contracts that degrade linearly with dataset size.
- If the request path now depends on long-running or bursty downstream work, check whether the contract should stay synchronous at all.

### Overload And Failure Interaction
- Flag retry patterns that amplify load, especially inside fan-out loops or shared critical paths.
- Flag missing explicit deadlines in performance-sensitive outbound chains.
- Require backpressure, bounded fallback, or shedding when a diff can accumulate queueing pressure.
- Treat "fallback to origin on every miss or error" as a performance risk when it can collapse the dependency under burst load.
- Keep ownership focused on performance impact; hand off full resilience-policy depth when needed.

### Cross-Domain Handoffs
- Hand off race, deadlock, goroutine lifecycle, and shutdown correctness to `go-concurrency-review`.
- Hand off DB/cache correctness, key design, invalidation, and transaction semantics to `go-db-cache-review`.
- Hand off retry, degradation, and admission-control policy depth to `go-reliability-review`.
- Hand off API contract, async acknowledgment, and payload semantics depth to `go-design-review` or the contract owner.
- Hand off test-plan depth to `go-qa-review` when the issue is mainly methodology, not performance reasoning.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the dominant performance axis (`latency`, `throughput`, `allocations`, `contention`, `I/O`, or `evidence`)
- the concrete regression or proof gap
- the evidence used, or the exact missing evidence required
- the concrete impact at the scale implied by the diff
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Use `Reference` for the relevant budget, approved perf decision, or `N/A` when no explicit budget exists.

Severity is merge-risk based:
- `critical`: proven severe regression, or missing mandatory evidence on a clearly high-risk hot-path change
- `high`: strong evidence of meaningful performance regression risk or unbounded amplification on a significant path
- `medium`: bounded but notable performance weakness
- `low`: local hardening or evidence-quality improvement

## Validation Command Patterns
Recommend only the commands that fit the finding. Useful defaults:
- `go test -run '^$' -bench BenchmarkName -benchmem ./...`
- `go test -run '^$' -bench BenchmarkName -count=6 ./... > new.txt`
- `benchstat old.txt new.txt`
- `go test -run '^$' -bench BenchmarkName -cpuprofile cpu.out -memprofile mem.out ./...`
- `go test -run '^$' -bench BenchmarkName -trace trace.out ./...`
- `go test -run '^$' -bench BenchmarkName -trace trace.out -blockprofile block.out -mutexprofile mutex.out ./...`
- `go tool trace trace.out`
- `go tool pprof -top block.out`
- `go tool pprof -top mutex.out`
- repo-specific integration or load commands when the risk is request-path or dependency-latency behavior rather than a local function

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

In `Issue`, start with the axis context when it improves clarity, for example `Axis: Evidence; ...` or `Axis: Contention; ...`.

## Escalate When
Escalate when:
- safe correction changes budgets, hot-path architecture, batching strategy, or performance trade-offs at design level (`go-performance-spec` or `go-design-spec`)
- the right answer requires new cache, query, or consistency decisions (`go-db-cache-spec`)
- overload, retry, admission-control, or degraded-mode policy must change (`go-reliability-spec`)
- latency shape depends on API contract, async acknowledgment, or distributed workflow changes (`api-contract-designer-spec` or `go-distributed-architect-spec`)
