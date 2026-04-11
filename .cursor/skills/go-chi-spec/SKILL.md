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
- define `NotFound`, `MethodNotAllowed`, `HEAD` when routing policy depends on it, `OPTIONS`, and CORS policy
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

## Reference Loading
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default. Load multiple only when the task clearly spans independent decision pressures, such as generated-route ownership plus route-label telemetry.

Pick the narrowest matching reference by symptom:

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| [references/router-topology-patterns.md](references/router-topology-patterns.md) | root router shape, `Route`/`Mount`/`Group` choice, top-level prefix ownership, generated/manual coexistence, route conflict or wildcard concern | makes the model choose one path owner with route-inventory proof instead of likely mistake `let modules, registration order, or broad wildcards decide ownership` |
| [references/middleware-layering-patterns.md](references/middleware-layering-patterns.md) | global vs scoped middleware, exact execution order, request context mutation, panic recovery, body limits, logging, generated middleware order | makes the model choose explicit scope and outer-to-inner stack semantics instead of likely mistake `make auth/CORS/logging global or assume route identity is available before routing finishes` |
| [references/notfound-methodnotallowed-options-cors.md](references/notfound-methodnotallowed-options-cors.md) | `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, CORS preflight, custom fallback JSON, scoped CORS placement | makes the model choose explicit fallback and preflight policy with header proof instead of likely mistake `trust framework defaults or duplicate CORS and hand-written OPTIONS behavior` |
| [references/openapi-oapi-codegen-integration.md](references/openapi-oapi-codegen-integration.md) | OpenAPI-generated chi handlers, `oapi-codegen` strict server wiring, `BaseURL`/mount prefix choice, generated/manual route boundaries, generated-code ownership | makes the model choose one generated API owner and config/wrapper changes instead of likely mistake `edit generated files, double-prefix routes, or add manual handlers inside the generated contract surface` |
| [references/route-template-observability.md](references/route-template-observability.md) | metrics, traces, logs, span names, route labels, `RoutePattern()`, `Match`/`Find`, raw-path labels, fallback route identity | makes the model choose bounded route-template labels after route resolution instead of likely mistake `use raw URL paths or incomplete pre-handler route patterns` |
| [references/router-validation-test-patterns.md](references/router-validation-test-patterns.md) | proof obligations for router topology, fallback, middleware order, CORS preflight, OpenAPI route coverage, observability labels | makes the model choose a small test/proof matrix tied to the routing risk instead of likely mistake `write generic happy-path route tests or prose-only validation` |

## Design Method
- Start from the affected route surfaces and list the routing decisions that are still implicit.
- Load the narrowest relevant reference only when it changes a routing decision; do not load references for general chi knowledge.
- Make selected and rejected options explicit for nontrivial routing choices.
- Treat framework-sensitive behavior as testable policy. Require repository proof or installed-version verification instead of relying on memory.
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
Return routing work in a compact, reviewable form. Include only sections that carry task-relevant decisions; omit headings whose domain is out of scope unless the omission is itself a handoff or risk:
- `Router Topology`
- `Middleware Order And Scope`
- `404/405/HEAD/OPTIONS/CORS Policy`
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
