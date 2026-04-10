---
name: go-chi-spec
description: "Design chi-based HTTP routing for Go services: router topology, middleware ordering, OpenAPI integration, 404/405/OPTIONS/CORS policy, route safety, and low-cardinality observability semantics."
---

# Go Chi Spec

## Purpose
Define or review chi-based transport routing so router topology, middleware behavior, fallback behavior, and route observability are explicit, stable, and testable.

## Specialist Stance
- Treat routing design as path ownership, middleware order, fallback behavior, and observability semantics.
- Keep OpenAPI as the contract owner and make chi integration serve that contract rather than re-owning it.
- Prefer explicit `net/http` composition and deterministic fallback rules over implicit framework defaults.
- Hand off API payload semantics, storage, security architecture, and reliability policy when routing no longer owns the hard decision.

## Scope
- define `chi` router topology and ownership of root vs subrouter composition
- define middleware layering, scope, and order-sensitive behavior
- define OpenAPI and `oapi-codegen` integration shape for chi-based routing
- define `NotFound`, `MethodNotAllowed`, `OPTIONS`, and CORS policy
- define controls against route conflict, shadowing, and accidental override
- define route-template observability semantics for logs, metrics, and traces
- define transport-boundary fallback and unmatched-route behavior

## Boundaries
Do not:
- take ownership of payload schema or endpoint method semantics as the primary output
- redesign storage, cache, or general architecture as the main result
- treat deep security architecture or SLI/SLO policy as the primary domain unless routing behavior depends on it
- prescribe low-level handler implementation as the main deliverable

## Core Defaults
- Keep `net/http` compatibility and explicit composition.
- Prefer deterministic routing behavior over ambiguous framework defaults.
- Keep middleware order explicit and reviewable; no hidden global router state.
- Keep observability labels route-template-based and low-cardinality.
- Keep OpenAPI as the source of truth; adapt routing integration rather than re-owning the contract.

## Expertise

### Chi Framing And Philosophy
- Treat `chi` as a stdlib-first router, not a full-stack framework.
- Keep business logic and domain rules out of router concerns.
- Use `chi` for composability:
  - route grouping and modular mounting
  - local middleware scoping
  - `net/http` ecosystem compatibility
- Preserve `http.Handler` interoperability for testing and lifecycle integration.
- Choose `chi` for routing and middleware clarity, not for trend alone.

### Chi Behavioral Semantics
- Make framework-sensitive behavior explicit:
  - global `Use(...)` middleware runs before final route-match context is fully resolved
  - `RoutePattern()` is reliable only after downstream handling has established the route context
  - default `405` and `OPTIONS` behavior must be pinned to an explicit policy
  - duplicate route registrations can silently override by registration order unless guarded against
- Treat route conflict, fallback behavior, CORS preflight, and route labeling as semantics, not implementation trivia.
- For framework-sensitive claims, rely on repository tests or official chi documentation rather than memory.

### Router Topology
- Require one deterministic ownership point for each affected path set.
- Make root router, mounted subrouter, and direct handler coexistence rules explicit.
- Prevent generated and manual routes from colliding silently.
- Reject route plans that allow hidden override behavior without tests or explicit guardrails.

### Middleware Order And Scope
- Define exact middleware order and explain why order matters for:
  - request or correlation ID
  - security headers
  - body and framing limits
  - access logging and route-label extraction
  - panic recovery
- Make `global` vs route-local scope explicit.
- Reject reorder proposals that do not analyze behavioral impact.

### OpenAPI And Code Generation Integration
- Keep OpenAPI as the contract source of truth.
- Make the `oapi-codegen` mode and strict-wrapper behavior explicit.
- Ensure generated and manual routes can coexist without collision ambiguity.
- Generated files are not for manual editing.

### 404, 405, OPTIONS, And CORS Policy
- Define explicit behavior for:
  - `NotFound`
  - `MethodNotAllowed`
  - `Allow` header behavior
  - `OPTIONS` handling
  - preflight CORS behavior
- Treat CORS placement as an architecture decision:
  - top-level middleware by default
  - scoped placement only with a clear `OPTIONS` strategy
- Do not leave API-facing fallback behavior to framework defaults when clients depend on it.

### Observability Route Semantics
- Prefer route-template extraction via chi route context.
- Define safe fallback behavior when a route template is unavailable.
- Make timing rules for route extraction explicit so logs, metrics, and spans stay aligned.
- Never use raw request paths, user IDs, or request IDs as metric labels.

### Resilience And Lifecycle Interface
- Preserve graceful startup and shutdown behavior of `http.Server`.
- Define transport behavior for unmatched and method-disallowed requests.
- Make router-level degradation or policy-mismatch behavior explicit where it affects clients or operators.

## Decision Quality Bar
Major routing recommendations should make the following explicit:
- the routing or middleware problem being solved
- at least two viable options when the decision is nontrivial
- selected and rejected options
- behavior-sensitive framework implications
- acceptance boundaries that can be tested
- reopen conditions

## Deliverable Shape
Return routing work in a compact, reviewable form:
- `Router Topology`
- `Middleware Order And Scope`
- `404/405/OPTIONS/CORS Policy`
- `OpenAPI And Codegen Integration Notes`
- `Route Observability Semantics`
- `Open Risks And Assumptions`

## Escalate When
Escalate if:
- router ownership or route collision behavior is ambiguous
- middleware order changes without a clear behavior-impact analysis
- API-facing `404`, `405`, `OPTIONS`, or CORS behavior is still implicit
- route observability semantics are undefined or high-cardinality-unsafe
- generated and manual route ownership cannot be made deterministic
