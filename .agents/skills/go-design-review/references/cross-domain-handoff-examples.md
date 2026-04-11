# Cross-Domain Handoff Examples

## Behavior Change Thesis
When loaded for symptom "design review found a seam but domain correctness belongs elsewhere," this file makes the model write one design-shaped finding plus a targeted specialist handoff instead of pretending to own every review domain or handing off everything generically.

## When To Load
Load this when the design reviewer can name a boundary, ownership, or proof-shape risk, but deep correctness requires API, chi, data/cache, security, reliability, concurrency, performance, QA, or delivery expertise.

Use this as a routing reference, not primary design guidance. If a narrower design reference explains the finding, load that one first and use this only for the handoff wording.

## Decision Rubric
- If the design seam is broken, write the design finding first; do not hide it as a handoff.
- If the design seam is intact and only specialist implementation detail is risky, hand off without inventing a design finding.
- Hand off only domain depth that changes merge risk or proof obligations.
- Name the exact specialist question: not "please review data," but "inspect transaction scope and cache invalidation atomicity."
- Add a design escalation only when the specialist answer could change ownership, architecture, or approved behavior.

## Imitate
```text
Findings:
- [medium] [go-design-review] internal/infra/http/router.go:52
  Issue: The router change bypasses the generated OpenAPI handler seam for one endpoint.
  Impact: Manual routing can drift from the contract source of truth and route-level observability.
  Suggested fix: Keep the route behind the generated handler path unless the API design is reopened.
  Reference: `docs/repo-architecture.md` API and HTTP ownership rows.

Handoffs:
- `go-chi-review`: verify chi mount order, fallback behavior, route labels, CORS/OPTIONS policy, and generated-route integration.
```

Copy this shape when the design finding is route/contract seam drift and chi semantics need specialist proof.

```text
Findings:
- [high] [go-design-review] internal/app/invoices/service.go:89
  Issue: App code now assumes repository writes and cache invalidation happen atomically, but no design owner was updated for that cross-resource invariant.
  Impact: The app layer is depending on a data/cache behavior that is invisible in the ownership map.
  Suggested fix: Keep the app contract explicit and route transaction/cache semantics through the data design or repository owner.
  Reference: task `design/ownership-map.md` if present.

Handoffs:
- `go-db-cache-review`: inspect transaction scope, context/resource safety, cache key correctness, invalidation, and fallback behavior.
```

Copy this shape when the design finding is invariant ownership and the specialist proof is data/cache mechanics.

```text
Findings:
- [critical] [go-design-review] cmd/service/internal/bootstrap/worker.go:33
  Issue: The HTTP binary now starts a durable background loop not present in the approved runtime flow.
  Impact: Workload lifecycle, shutdown, retry, and scaling ownership changed without design approval.
  Suggested fix: Remove the loop from this diff or reopen design for async workload ownership.
  Reference: `docs/repo-architecture.md` background/async extension path.

Handoffs:
- `go-concurrency-review`: inspect goroutine lifecycle, channels, WaitGroups, and shutdown safety.
- `go-reliability-review`: inspect retry, backpressure, overload, degradation, and lifecycle readiness.
```

Copy this shape when one design drift creates two independent proof obligations.

## Reject
```text
Handoffs:
- `go-security-review`
- `go-db-cache-review`
- `go-qa-review`
- `go-performance-review`
```

Reject because it names review lanes without the seam or specialist question.

```text
Findings:
- [high] [go-design-review] internal/infra/http/admin.go:77
  Issue: The authorization condition may be wrong.
```

Reject as a design finding if ownership is intact and the question is purely auth correctness; hand off to `go-security-review` instead.

## Handoff Map
- `go-chi-review`: chi router, middleware order/scope, fallback, CORS/OPTIONS, route labels, generated handler integration.
- `go-db-cache-review`: SQL access discipline, transactions, context/resource safety, cache keys, invalidation, stampede/fallback risk.
- `go-security-review`: trust boundaries, auth, tenant isolation, injection, SSRF, secrets, abuse resistance.
- `go-reliability-review`: deadlines, retries, backpressure, degradation, startup/shutdown, rollout safety.
- `go-concurrency-review`: goroutines, channels, WaitGroups, mutexes, atomics, timers, worker pools.
- `go-performance-review`: hot paths, batching, serialization, fan-out, caching, contention, benchmark evidence.
- `go-qa-review`: test obligations, scenario coverage, assertion strength, determinism, validation readiness.
- `go-devops-spec` or delivery review lane: generated-code policy, CI, rollout, release, compatibility, migration choreography.

## Agent Traps
- Do not duplicate a specialist review as a design finding when the design seam is intact.
- Do not let external examples override approved repo ownership.
- Do not use this file as a broad checklist; it is for routing a known seam.

## Validation Shape
Validate that the handoff question is narrow enough for the specialist to answer and that the design review output still includes any design drift needed to understand merge risk.
