---
name: go-performance-review
description: "Use when a Go diff affects hot paths, batching, serialization, fan-out, query count, caching, allocation, contention, `sync.Pool`, or benchmark/profile evidence; Own measurable performance-regression risk and proof quality; Skip when budgets are unset or correctness belongs to concurrency, DB/cache, reliability, or domain review."
---

# Go Performance Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review measured hot-path regressions, query/fan-out/serialization/allocation/lock costs, and justified optimization complexity. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate changed performance budget or architecture to `go-performance-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
