---
name: go-chi-review
description: "Review Go chi routing changes for router ownership, middleware scope and order, HTTP fallback behavior, route observability semantics, and OpenAPI or generated-route integration."
---

# Go Chi Review

## Purpose
Protect transport-routing correctness in changed `github.com/go-chi/chi/v5` code so routing behavior, middleware boundaries, and HTTP-visible semantics stay explicit and predictable.

## Scope
- review router topology, route ownership, and subrouter boundaries
- review route collision, shadowing, override, and registration-order risk
- review middleware order, scope, and lifecycle assumptions
- review `404`, `405`, `Allow`, `OPTIONS`, and CORS behavior on affected surfaces
- review route observability semantics, including low-cardinality route labeling
- review OpenAPI or generated-route integration with manual chi wiring
- review transport lifecycle implications when routing changes affect startup, readiness, or fallback behavior

## Boundaries
Do not:
- redesign the broader architecture unless routing correctness cannot be restored locally
- take primary ownership of business invariants, payload semantics, DB/cache policy, or deep security and reliability analysis
- block on style-only comments with no concrete transport or runtime impact
- leave API-visible routing behavior to framework defaults when the change affects contract semantics

## Core Defaults
- Prefer explicit route ownership and explicit subrouter boundaries.
- Prefer explicit middleware order and scope over implicit inheritance.
- Treat registration-order-dependent behavior as risky until proven deliberate.
- Treat raw-path observability labels as unsafe because they create high-cardinality telemetry.
- Prefer the smallest safe routing fix that restores deterministic behavior.

## Expertise

### Router Topology And Ownership
- Verify root router, grouped routes, and mounted subrouters keep ownership obvious.
- Flag duplicate or ambiguous `method + pattern` registration.
- Flag pattern routes that can shadow more specific static routes.
- Reject hidden overlap between generated routes and manual routes.
- Require route ownership to remain testable and observable.

### Middleware Order And Scope
- Validate middleware order for request IDs, auth context, body limits, logging, tracing, panic recovery, and response shaping.
- Flag reorderings that change behavior without explicit reason.
- Verify route-local middleware does not silently widen or narrow coverage.
- Flag middleware that depends on route pattern before the route is resolved.

### HTTP Fallback And Policy Semantics
- Verify `NotFound`, `MethodNotAllowed`, `Allow`, `OPTIONS`, and CORS behavior remain deliberate and contract-consistent.
- Flag reliance on defaults for API-visible behavior.
- Check that preflight handling is complete for affected routes and scopes.
- Treat inconsistent `404` vs `405` behavior across related surfaces as a correctness risk.

### Route Observability Semantics
- Require route labels, spans, and logs to use template-level route semantics when available.
- Flag use of raw request paths or unbounded path fragments as metric or trace labels.
- Verify logs, metrics, and traces describe the same route identity.
- Require explicit fallback behavior when route pattern extraction is unavailable.

### OpenAPI And Generated Wiring
- Verify generated handlers and manual chi routes coexist without collision or ownership ambiguity.
- Preserve no-touch boundaries for generated artifacts.
- Flag runtime drift between the intended contract and the actual chi wiring.
- Check that middleware and fallback behavior around generated routes matches the surrounding transport policy.

### Transport Lifecycle Signals
- Review routing changes for startup registration safety, readiness expectations, and panic-recovery boundaries.
- Flag transport changes that can make unmatched or disallowed requests behave unpredictably during shutdown or degraded startup.
- Keep lifecycle concerns focused on the routing layer; hand off broader resilience policy when needed.

### Cross-Domain Handoffs
- Hand off trust-boundary and tenant-isolation root causes to `go-security-review`.
- Hand off timeout, degradation, and fallback policy defects to `go-reliability-review`.
- Hand off benchmark or hot-path evidence questions to `go-performance-review`.
- Hand off goroutine, channel, and shutdown-coordination defects to `go-concurrency-review`.
- Hand off broader architecture drift to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete routing or middleware defect
- runtime or contract-visible impact
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed routing defect that makes behavior merge-unsafe
- `high`: strong evidence of major routing or HTTP-policy drift
- `medium`: bounded but meaningful transport-correctness risk
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
[severity] [go-chi-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

Use `Reference` for the relevant contract, design note, or approved decision when one exists.

## Escalate When
Escalate when:
- safe correction changes route ownership, router topology, or middleware strategy in a non-local way (`go-chi-spec`)
- API-visible behavior such as method/status/fallback/CORS semantics must change (`api-contract-designer-spec`)
- route observability semantics need a new telemetry contract (`go-observability-engineer-spec`)
- routing fix depends on new timeout, fallback, or startup/shutdown policy (`go-reliability-spec`)
- transport correction exposes broader seam or architecture drift (`go-design-spec`)
