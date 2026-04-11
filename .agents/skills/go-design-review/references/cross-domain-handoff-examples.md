# Cross-Domain Handoff Examples

## When To Load
Load this when design review detects that a diff has crossed a domain seam, but the deep correctness question belongs to a specialist review lane.

Use this to keep the review output focused. The design reviewer names the shape risk, then hands off the domain-specific proof instead of pretending to own every domain.

## Concrete Review Examples
HTTP contract handoff:

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

Data handoff:

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

Security handoff:

```text
Findings:
- [high] [go-design-review] internal/infra/http/admin.go:77
  Issue: Authorization moved from middleware into endpoint-local branching without a design note.
  Impact: The trust boundary becomes per-handler convention rather than a structural guard.
  Suggested fix: Restore the approved auth owner or reopen design for endpoint-local authorization semantics.
  Reference: task security/design decisions if present.

Handoffs:
- `go-security-review`: verify fail-closed behavior, tenant isolation, authz checks, and abuse resistance.
```

Reliability and concurrency handoff:

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

QA handoff:

```text
Findings:
- [medium] [go-design-review] internal/infra/http/problems.go:121
  Issue: Error classification is now centralized, which is the right owner, but the diff changes multiple endpoint-visible outcomes.
  Impact: The design seam is cleaner, yet the changed response matrix needs proof.
  Suggested fix: Keep the owning seam and add validation for affected endpoints.
  Reference: task `plan.md` validation section if present.

Handoffs:
- `go-qa-review`: verify scenario traceability, assertion strength, and contract coverage for changed outcomes.
```

## Non-Findings To Avoid
- Do not duplicate a specialist review as a design finding if the design seam is intact and the issue is purely implementation detail.
- Do not hand off everything. Hand off only domain depth that changes merge risk or proof obligations.
- Do not hide a design finding inside a handoff. If ownership drift exists, make it a finding and then add the handoff.
- Do not let external examples override approved repo ownership.

## Smallest Safe Correction
- State the design-shape issue in one finding.
- Add one targeted handoff with the exact specialist question.
- If the specialist question could change ownership or architecture, mark a design escalation too.
- Keep suggested fixes local unless the approved design must be reopened.

## Escalation Rules
- Use `go-chi-review` for chi router, middleware order/scope, fallback, CORS/OPTIONS, route labels, and generated handler integration.
- Use `go-db-cache-review` for SQL access discipline, transactions, context/resource safety, cache keys, invalidation, and stampede/fallback risk.
- Use `go-security-review` for trust boundaries, auth, tenant isolation, injection, SSRF, secrets, and abuse resistance.
- Use `go-reliability-review` for deadlines, retries, backpressure, degradation, startup/shutdown, and rollout safety.
- Use `go-concurrency-review` for goroutines, channels, WaitGroups, mutexes, atomics, timers, and worker pools.
- Use `go-performance-review` for hot paths, batching, serialization, fan-out, caching, contention, and benchmark evidence.
- Use `go-qa-review` for test obligations, scenario coverage, assertion strength, determinism, and validation readiness.

## Exa Source Links
- [Organizing a Go module - The Go Programming Language](https://go.dev/doc/modules/layout)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Architecture Decision Record - Martin Fowler](https://martinfowler.com/bliki/ArchitectureDecisionRecord.html)
