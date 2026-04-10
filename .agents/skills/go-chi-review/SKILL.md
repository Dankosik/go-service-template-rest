---
name: go-chi-review
description: "Review Go chi routing changes for router ownership, chi-specific middleware and mount semantics, HTTP fallback policy, route observability labels, and OpenAPI/generated-route integration. Use whenever a Go diff touches github.com/go-chi/chi/v5 routers, middleware order or scope, NotFound/MethodNotAllowed/OPTIONS/CORS handling, route labeling, or manual wiring around generated handlers, even if the request is framed as a generic code review."
---

# Go Chi Review

## Purpose
Protect changed `github.com/go-chi/chi/v5` transport code from routing, middleware, and HTTP-policy regressions, with emphasis on chi-specific runtime traps that can silently change behavior or panic at startup.

## Specialist Stance
- Review chi behavior as runtime semantics, not framework trivia.
- Prioritize route ownership, middleware scope, fallback policy, and low-cardinality observability over style-only comments.
- Treat startup panics, route-context mutation, and generated/manual route drift as merge-risk signals.
- Hand off payload contracts, business invariants, DB/cache, and broad security or reliability design when they become primary.

## Scope
- review router topology, path ownership, and subrouter boundaries
- review chi-specific registration semantics such as `Use`, `Group`, `Route`, `With`, and `Mount`
- review route collision, shadowing, override, and registration-order risk
- review middleware order, scope, and route-lifecycle assumptions
- review `404`, `405`, `Allow`, `HEAD`, `OPTIONS`, and CORS behavior on affected surfaces
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
- Deterministic routing behavior beats framework convenience.
- Treat chi runtime semantics as reviewable behavior, not implementation trivia.
- Treat registration-order-dependent behavior as risky until proven deliberate.
- Treat raw-path observability labels as unsafe because they create high-cardinality telemetry.
- When multiple chi defects coexist, prioritize the one that corrupts live route state, startup safety, or advertised HTTP capability most directly.
- Prefer the smallest safe routing fix that restores deterministic behavior.

## Reference Files
Load these files lazily when a finding needs chi-specific examples or validation shape:
- [references/chi-router-registration-hazards.md](references/chi-router-registration-hazards.md): route registration, startup panic, `Use`, duplicate `Mount`, and subtree ownership hazards.
- [references/middleware-order-and-scope.md](references/middleware-order-and-scope.md): middleware order, route identity timing, and `Use`/`With`/`Group`/`Route`/`Mount` scope mistakes.
- [references/route-context-and-match-probing.md](references/route-context-and-match-probing.md): `RouteContext`, `RoutePattern`, `Match`, and `Find` probing hazards.
- [references/http-fallback-head-options-cors.md](references/http-fallback-head-options-cors.md): `404`, `405`, `Allow`, `HEAD`, `OPTIONS`, and CORS review checks.
- [references/generated-and-manual-route-drift.md](references/generated-and-manual-route-drift.md): OpenAPI/generated handler wiring, manual route overlap, and generated-route drift.
- [references/route-observability-labels.md](references/route-observability-labels.md): low-cardinality route labels, span names, metrics, and log route identity.

Keep findings review-oriented after reading references: exact file/line, runtime impact, smallest safe fix, and a validation command. Do not turn these references into design-spec output.

## Expertise

### Chi Runtime Semantics
- `Use(...)` middleware on a mux executes before route resolution. Flag route-dependent logic in global middleware unless it runs after `next.ServeHTTP(...)` or derives route identity safely.
- `With(...)` and `Group(...)` create inline routers with copied middleware stacks. Verify scope widening or narrowing is intentional rather than an accidental side effect of refactoring.
- `Route(pattern, fn)` creates a new router and mounts it. `Mount(pattern, h)` reserves `pattern`, `pattern/`, and `pattern/*`; review path ownership with that wildcard behavior in mind.
- `NotFound` and `MethodNotAllowed` handlers can propagate into mounted or inline routers. Verify subrouters do not silently inherit or bypass policy.

### Registration And Startup Hazards
- Flag `Use(...)` added after the first route registration on the same mux. In chi this panics at startup.
- Flag `Mount(...)` on an already owned pattern. In chi this panics when mounting on an existing path.
- Flag diffs that rely on registration order to make one handler “win” ownership, especially when manual handlers and mounts overlap.
- Treat startup panics from router construction as `high` or `critical` merge risk, even if the steady-state routing logic looks correct.

### Match And RouteContext Probing
- `RoutePattern()` is only reliable after downstream handling has resolved the final route stack. Reading it before `next.ServeHTTP(...)` usually produces incomplete route identity.
- `Match(...)` and `Find(...)` mutate the supplied `*chi.Context`. Flag helpers that probe alternate methods or paths using `chi.RouteContext(r.Context())` instead of a fresh `chi.NewRouteContext()`.
- Treat live-request `RouteContext` mutation in custom `405`/`Allow` logic as a primary merge-risk finding, not a secondary cleanup note.
- Review custom `Allow`, `OPTIONS`, or ownership-probing helpers for context corruption, stale route state, or incorrect method disclosure.
- Prefer a fresh probe context per check. Do not present request-context reuse as an equal alternative to isolated probe contexts.
- Require bounded fallback labels when route-template extraction fails; never fall back to raw unbounded request paths.

### Router Topology And Ownership
- Verify root router, mounted subrouter, grouped routes, and generated handlers keep one obvious owner per resource path.
- Flag duplicate or ambiguous `method + pattern` registration.
- Check for static, param, wildcard, and mount overlap that makes route ownership non-obvious or registration-order-sensitive.
- Treat split ownership of the same resource path across generated and manual routers as a `404`/`405`/`Allow`/`OPTIONS` risk, not just a style issue.
- Explain mount conflicts in subtree-ownership terms first; exact panic strings or internal chi checks are supporting evidence, not the main explanation.

### Middleware Order And Scope
- Validate middleware order for request IDs, auth context, body limits, logging, tracing, panic recovery, and response shaping.
- Flag reorderings that change behavior without explicit reason.
- Verify route-local middleware does not silently widen or narrow coverage after `Group`, `With`, `Route`, or `Mount`.
- Flag middleware that depends on final route identity before the route is resolved.

### HTTP Method And Fallback Semantics
- Verify `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, and CORS behavior remain deliberate and contract-consistent.
- Remember chi does not automatically route `HEAD` to `GET`; that requires an explicit `Head(...)` route or `middleware.GetHead`.
- Flag custom `Allow` or fallback logic that claims `HEAD` support when the router cannot actually serve it.
- When a custom `405` helper both mutates probe state and overclaims supported methods, report both defects as first-class findings.
- Check that preflight handling is complete for affected routes and scopes.
- Treat inconsistent `404` vs `405` vs `204` behavior across related surfaces as a correctness risk.

### Route Observability Semantics
- Require route labels, spans, and logs to use template-level route semantics when available.
- Flag use of raw request paths, wildcard captures, IDs, or other unbounded path fragments as metric or trace labels.
- Verify logs, metrics, and traces describe the same route identity.
- Require explicit bounded behavior for unmatched routes instead of user-controlled fallback labels.

### OpenAPI And Generated Wiring
- Verify generated handlers and manual chi routes coexist without collision or ownership ambiguity.
- Preserve no-touch boundaries for generated artifacts.
- Flag runtime drift between the intended contract and the actual chi wiring.
- Check that middleware and fallback behavior around generated routes matches the surrounding transport policy.

### Validation Strategy
- Prefer concrete `httptest` or router-constructor validation over prose for `404`/`405`/`Allow`/`HEAD`/`OPTIONS` findings.
- When startup panic risk exists, suggest constructor-level tests that exercise router assembly, not only happy-path requests.
- When route ownership is ambiguous, suggest direct method-path cases that prove which router or fallback policy actually wins.
- For observability-cardinality findings, validate that multiple concrete parameter values collapse to the same route-template label and unmatched paths collapse to one bounded fallback label.

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
- the concrete chi routing or middleware defect
- runtime or contract-visible impact
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed routing defect or startup failure that makes behavior merge-unsafe
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

Use `Reference` for the relevant contract, design note, approved decision, or chi-specific behavior when one exists.
Do not pad `Reference` with filler phrases; use it only for concrete supporting behavior or contract evidence.

## Escalate When
Escalate when:
- safe correction changes route ownership, router topology, or middleware strategy in a non-local way (`go-chi-spec`)
- API-visible behavior such as method/status/fallback/CORS semantics must change (`api-contract-designer-spec`)
- route observability semantics need a new telemetry contract (`go-observability-engineer-spec`)
- routing fix depends on new timeout, fallback, or startup/shutdown policy (`go-reliability-spec`)
- transport correction exposes broader seam or architecture drift (`go-design-spec`)
