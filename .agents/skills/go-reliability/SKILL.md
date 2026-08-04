---
name: go-reliability
description: "Reliability: Use for timeouts, retries, overload, degradation, readiness, startup/drain/shutdown, recovery, rollout, or review. Own service policy; Skip synchronization, durable replay, or context semantics."
---

# Go Reliability

Resilience is **budget** arithmetic: every dependency call spends a slice of the caller's remaining end-to-end budget, and every retry spends it again at the worst possible time.

`end-to-end budget -> per-hop deadline -> failure disposition -> retry, degrade, or shed -> lifecycle stage -> rollout -> proof`

A timeout is derived from the accepted end-to-end bound rather than chosen locally. A retry earns its place only with a bounded budget, jitter, and a safely repeatable effect, because unbudgeted retries amplify load exactly when capacity is scarcest — `internal/infra/httpclient/retry.go` already encodes those three rules, and the discipline named below owns the rest. Degradation and load shedding are designed states with named behavior and signals, and readiness, startup, drain, and shutdown are contract stages where accepted work finishes or fails visibly.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected dependencies and lifecycle stages from accepted behavior, changed call paths and configuration, startup/shutdown wiring, and rollout topology. Classify criticality, then trace failure through each stage, binding end-to-end budgets, timeout, retry, overload, degradation, readiness/liveness, recovery, evidence, and rollout behavior.

Most of that arithmetic already has an owner here, and its numbers are enforced rather than conventional. `internal/infra/http` installs the request budget and the shedding and rate-limit layers, `internal/health` owns cached readiness, `cmd/service/internal/bootstrap` owns the one clamped shutdown deadline, and `internal/config/validate.go` rejects a configuration whose budgets do not nest. Read the affected owner before proposing a policy, and prefer extending it to placing a second one beside it.

## Choose The Branch

The branch decides what you return; both branches read from the same [reference selector](references/index.md), loading one entry by default and another only for an independent pressure.

- **Decision** — select when resilience policy is absent or changing. Complete when shared Decision dispositions cover every dependency and lifecycle stage with budget, failure, recovery, and rollout consequences explicit.
- **Review** — select when changed Go must conform to accepted resilience policy. Trace every affected failure path into the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and focused proof. Missing policy returns to the named Resilience Decision owner.

Hand concrete synchronization to `go-concurrency`, durable recovery to `go-distributed`, and context API misuse to `go-idiomatic`. Load [`external-api-integration`](../../../docs/universal-disciplines/external-api-integration/SKILL.md) when a dependency call creates or changes state on the other side: it forces one operation identity carried from request through ambiguous outcome to reconciliation, instead of a timeout-and-retry policy that treats an unknown outcome as a failure. Load [`durable-background-jobs`](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md) when work must outlive the request, the worker, or a deploy: it forces claim, lease, and recovery contracts instead of a goroutine with a retry loop.
