# Boundary And Ownership Drift

## When To Load
Load this when a Go diff changes where behavior lives, bypasses a component owner, or moves policy across app, domain, infra, HTTP, config, bootstrap, telemetry, or migration boundaries.

Use repository-approved `spec.md`, `design/`, and `docs/repo-architecture.md` first. External links below calibrate Go package and decision-documentation patterns only.

## Concrete Review Examples
Finding example: `internal/app/orders` imports `internal/infra/http` to build HTTP problem responses.

Review shape:

```text
[high] [go-design-review] internal/app/orders/service.go:42
Issue: The app layer now depends on the HTTP adapter to shape domain errors, reversing the approved transport-agnostic app boundary.
Impact: Future non-HTTP callers inherit transport semantics and must import the HTTP package, so a local handler convenience becomes a system ownership change.
Suggested fix: Return a domain/app error shape from `internal/app/orders` and keep HTTP response mapping in `internal/infra/http`.
Reference: task `design/ownership-map.md` if present; otherwise `docs/repo-architecture.md` app and HTTP boundary rows.
```

Finding example: a handler starts reading env/config flags directly because it needs a request-specific option.

Review shape:

```text
[high] [go-design-review] internal/infra/http/widgets.go:88
Issue: The HTTP adapter now owns runtime config lookup that the repository assigns to `internal/config` and bootstrap wiring.
Impact: Config precedence and validation can diverge between startup and request handling, even if this endpoint works in tests.
Suggested fix: Pass the already validated config value through the composition root or add an approved app/adapter option owned by bootstrap.
Reference: `docs/repo-architecture.md` config and bootstrap ownership rows.
```

Finding example: a new concrete adapter is constructed from inside `internal/app`.

Review shape:

```text
[critical] [go-design-review] internal/app/reporting/service.go:61
Issue: The use case now constructs the Postgres adapter directly, bypassing the bootstrap composition root.
Impact: App behavior becomes coupled to a concrete integration and can no longer be reused by another binary or test seam without pulling in persistence lifecycle.
Suggested fix: Move construction back to bootstrap and pass the dependency through an app-facing contract only if the app layer needs inversion.
Reference: `docs/repo-architecture.md` stable dependency direction.
```

## Non-Findings To Avoid
- Do not flag a new same-package helper solely because it extracts code; flag only when ownership, source of truth, or dependency direction changes.
- Do not object to `internal/` packages in a server repository by default; Go's official module guidance recommends keeping server logic internal when it is not exported for other modules.
- Do not require a new domain abstraction just because an adapter exists. A concrete type wired by bootstrap can be simpler when no consumer-owned interface is needed.
- Do not turn a small placement concern into a full architecture redesign. Keep the finding to the line that crosses ownership.

## Smallest Safe Correction
- Move the crossed behavior back to the owning package.
- Pass already validated data or a narrow dependency from the composition root rather than importing outward from app/domain code.
- Keep transport mapping at the transport edge and business policy in app/domain code.
- If the diff reveals a real missing owner, ask for a design escalation instead of inventing the new owner in the review comment.

## Escalation Rules
- Escalate to `go-design-spec` or `go-architect-spec` when the smallest correction changes approved component ownership.
- Hand off to `go-chi-review` when the boundary issue is specifically HTTP router, middleware, OpenAPI handler, fallback, or route-label behavior.
- Hand off to `go-db-cache-review` when ownership drift centers on transactions, repository contracts, cache keys, or datastore lifecycle.
- Mark as a design escalation, not a code-only finding, when multiple packages already depend on the new boundary and a local move is no longer safe.

## Exa Source Links
- [Organizing a Go module - The Go Programming Language](https://go.dev/doc/modules/layout)
- [Go Code Review Comments - Interfaces and Package Names](https://go.dev/wiki/CodeReviewComments)
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Architecture Decision Record - Martin Fowler](https://martinfowler.com/bliki/ArchitectureDecisionRecord.html)
