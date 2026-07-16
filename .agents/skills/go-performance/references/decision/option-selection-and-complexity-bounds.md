# Option Selection And Complexity Bounds

## Behavior Change Thesis
When loaded for symptom "the proposed optimization class is under-justified or disproportionately complex," this file makes the model choose the least-complex option that can meet the budget instead of likely mistake reaching for cache, PGO, object pooling, async workflows, or fan-out by habit.

## When To Load
Load as a challenge/smell-triage reference when the spec is comparing optimization classes and the narrow positive reference does not already decide the path. Do not load by default when a narrower reference matches, such as PGO lifecycle, memory budgets, payload bounds, or concurrency capacity.

## Decision Rubric
- First fix the contract: if budget, workload, or proof is missing, block or route to the narrower reference before selecting an optimization.
- Prefer no new mechanism when the current path already meets the budget or the bottleneck is unproven.
- Prefer bounding existing work before adding acceleration: smaller payloads, page limits, query count limits, batching, or eliminating redundant round trips.
- Prefer algorithmic/data-flow changes before runtime or compiler tuning when the bottleneck is structural.
- Allow caching only after cacheability, staleness, hit ratio, stampede protection, and cache-down behavior are explicit.
- Allow fan-out or worker pools only after parallelizability, concurrency caps, downstream capacity, deadlines, and cancellation are explicit.
- Allow object pooling or allocation-focused work only after allocation rate, lifetime, contention, and memory/latency trade-offs are part of the budget.
- Allow PGO only when CPU-bound behavior and representative profile lifecycle are already proven or planned.
- Allow async behavior only when client-visible acceptance, status, retry, and completion semantics are handed off.

## Imitate
- Checkout read is `p99` slow because response shape includes two optional nested collections by default. The spec selects bounded representation and pagination before cache or encoder replacement. Copy the lower-complexity option tied to the observed bottleneck.
- Bulk import has high CPU in parsing and stable memory. The spec compares parser algorithm improvement and PGO, selects parser work first, and leaves PGO as a later option only after representative CPU profiles exist. Copy the staged complexity.
- Catalog endpoint is slow during cache-down mode. The spec rejects adding a second cache until origin concurrency, DB pool wait, and fallback deadline are budgeted. Copy the fallback-first pressure test.

## Reject
- "Add Redis" before stating cacheability, freshness, key dimensions, hit ratio, and cache-down fallback.
- "Use PGO" while the bottleneck hypothesis is DB wait, lock contention, or payload size.
- "Use sync.Pool" without allocation pressure, object lifetime, contention, and correctness ownership boundaries.
- "Spawn workers" when the downstream DB pool is saturated or when cancellation behavior is unknown.
- "Make it async" to hide latency without durable acceptance and completion-status semantics.

## Agent Traps
- Treating a powerful technique as an architecture decision by itself.
- Adding a mechanism because it is locally familiar instead of because it is the cheapest way to satisfy the budget.
- Comparing options only by latency while ignoring memory, complexity, correctness handoff, rollout, and operational failure modes.
- Leaving the rejected option unstated, which lets implementation drift back to the flashier choice.

## Validation Shape
The spec should show a compact option table:

| Option | Budget Fit | Added Risk | Proof Needed | Decision |
| --- | --- | --- | --- | --- |
| Bound payload/page size | Likely meets p95/p99 | API contract handoff | payload-bucket benchmark plus body-size telemetry | select |
| Add cache | Unknown | freshness, stampede, cache-down DB pressure | cache hit/miss/cache-down proof | reject until cacheability is explicit |
| Enable PGO | Not aimed at current bottleneck | profile lifecycle and rollout complexity | representative CPU profile plus canary | reject |

Validation passes when the selected option is the least complex option that plausibly meets the budget and each rejected higher-complexity option has a concrete rejection reason.
