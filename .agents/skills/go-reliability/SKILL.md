---
name: go-reliability
description: "Reliability: Use for timeout/retry/overload/readiness/startup/drain/shutdown/recovery/rollout. Own resilience; Skip synchronization/durable replay/context semantics."
---

# Go Reliability

Resilience is **budget** arithmetic: every dependency call and retry spends the
caller's remaining end-to-end budget.

`budget -> per-hop deadline -> failure disposition -> retry, degrade, or shed -> lifecycle -> rollout -> proof`

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
affected dependencies and lifecycle stages from accepted behavior, call paths,
configuration, startup/shutdown wiring, and rollout topology. Derive timeouts
from the accepted bound. A retry needs a bounded budget, jitter, and a safely
repeatable effect; degradation and load shedding need named behavior and
signals. Startup, readiness, drain, and shutdown state how accepted work
finishes or fails visibly.

Read the existing owner before adding policy: `internal/infra/http` owns request
budgets and shedding, `internal/health` owns cached readiness,
`cmd/service/internal/bootstrap` owns the clamped shutdown deadline, and
`internal/config/validate.go` enforces nested budgets. Load the [reference
selector](references/index.md) for a new outbound wait, pooled acquire,
concurrency or queue bound, readiness, drain, or teardown pressure.

For a **Decision**, cover every dependency and lifecycle stage with budget,
failure, recovery, and rollout consequences. For **Review**, trace each affected
failure path into the shared finding envelope.

Hand synchronization to `go-concurrency`, durable recovery to `go-distributed`,
context semantics to `go-idiomatic`, external state changes to [external API
integration](../../../docs/universal-disciplines/external-api-integration/SKILL.md),
and request-surviving work to [durable jobs](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md).
