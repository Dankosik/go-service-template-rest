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

## Reference Files
Load only the files needed for the routing design question.

- `references/router-topology-patterns.md`: path ownership, root router vs subrouter shape, manual/generated route coexistence, mount/group/route tradeoffs, route conflict and shadowing controls.
- `references/middleware-layering-patterns.md`: global vs route-local middleware scope, order-sensitive stacks, request context mutation, panic recovery, logging, body limits, and generated middleware order compatibility.
- `references/notfound-methodnotallowed-options-cors.md`: `NotFound`, `MethodNotAllowed`, `Allow` headers, `OPTIONS`, CORS preflight, and top-level vs scoped CORS placement.
- `references/openapi-oapi-codegen-integration.md`: OpenAPI source-of-truth routing, `oapi-codegen` chi server or strict server wiring, generated/manual route boundaries, generated-code ownership, and codegen compatibility settings.
- `references/route-template-observability.md`: route-template labels, `RoutePattern()` timing, `Find`/`Match` tradeoffs, raw-path rejection, and safe fallback labels.
- `references/router-validation-test-patterns.md`: route table validation, conflict and fallback tests, middleware-order probes, CORS preflight tests, OpenAPI route coverage, and observability-label assertions.

## Design Method
- Start from the affected route surfaces and list the routing decisions that are still implicit.
- Load the relevant reference files and reuse their options, rejected alternatives, examples, and acceptance boundaries.
- Make selected and rejected options explicit for nontrivial routing choices.
- Treat framework-sensitive behavior as testable policy. Cite the reference source or require repository proof instead of relying on memory.
- Keep the output focused on chi routing and transport composition. Hand off payload schema, persistence, security architecture, broad reliability policy, and SLI/SLO ownership unless routing behavior directly depends on them.

## Decision Quality Bar
Major routing recommendations should make the following explicit:
- the routing or middleware problem being solved
- selected and rejected options when the decision is nontrivial
- behavior-sensitive chi or `net/http` implications
- generated vs manual route ownership when OpenAPI is involved
- acceptance boundaries that can be tested
- adjacent handoffs and reopen conditions

## Reject Conditions
Reject designs that:
- rely on implicit fallback or CORS defaults for client-visible API behavior
- use raw request paths, user IDs, request IDs, or other high-cardinality values as metrics labels
- allow generated and manual route ownership to collide without a validation hook
- change middleware order without explaining the behavior impact
- turn routing work into payload schema, storage, security architecture, or broad reliability design

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
