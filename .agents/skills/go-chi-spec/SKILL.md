---
name: go-chi-spec
description: "Use when chi transport routing must be decided before coding; Own router composition, route ownership, middleware ordering, OpenAPI wiring, fallback behavior, CORS/OPTIONS handling, and route-template labels; Skip when the primary decision is client-visible API semantics, system topology, or implementation review."
---

# Go Chi Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this chi transport boundary.

## Outcome And Boundary

Define testable chi/`net/http` composition: root and subtree ownership, `Route`/`Mount`/`Group` shape, middleware scope/order, generated/manual registration, route conflict controls, unmatched and method fallback, `HEAD`, `OPTIONS` and CORS handling, and bounded route-template labels.

Do not choose resource/payload or endpoint method/status policy, system/service topology, persistence, security policy, reliability budgets, SLI/SLOs, or handler internals. OpenAPI owns the accepted public contract; chi owns deterministic transport wiring that realizes it.

## Transport Core

1. Give every path prefix and subtree one composition owner; make generated/manual boundaries explicit and reject wildcard, duplicate, shadowed, or registration-order ownership.
2. Specify middleware outer-to-inner order and exact global/group/route scope. Account for panic recovery, request mutation, auth, body limits, CORS, logging, and when final route identity becomes available; use no hidden global router state.
3. Keep generated OpenAPI output authoritative and unedited. Decide mount prefix and `BaseURL` once, and place manual routes outside the generated contract surface.
4. Define deterministic API `NotFound`, `MethodNotAllowed` with `Allow`, implicit or explicit `HEAD`, unmatched and preflight `OPTIONS`, and CORS middleware behavior. Do not duplicate CORS and hand-written preflight ownership.
5. Derive log/metric/trace route identity from the resolved template with a bounded fallback label; never label raw paths, IDs, or other high-cardinality values.
6. Treat chi-version-sensitive matching, middleware timing, fallback, CORS, and generated wiring as proof obligations verified from repository code/tests or the installed version, not memory.

## Symptom-Driven References

| Pressure | Load | Required effect |
| --- | --- | --- |
| Root/subrouter shape, prefix ownership, generated/manual coexistence, conflicts, or wildcards | [router-topology-patterns.md](references/router-topology-patterns.md) | Choose one path owner and route-inventory proof. |
| Global/scoped middleware, exact order, context mutation, recovery, limits, logging, or generated wrappers | [middleware-layering-patterns.md](references/middleware-layering-patterns.md) | Specify scope and outer-to-inner behavior. |
| `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, preflight, fallback JSON, or CORS placement | [notfound-methodnotallowed-options-cors.md](references/notfound-methodnotallowed-options-cors.md) | Select explicit fallback/preflight mechanics and header proof. |
| Generated strict handlers, mount prefix, `BaseURL`, generated/manual ownership, or source authority | [openapi-oapi-codegen-integration.md](references/openapi-oapi-codegen-integration.md) | Prevent generated edits, double prefixes, and ownership collision. |
| Logs, metrics, traces, span names, `RoutePattern()`, `Match`/`Find`, or fallback identity | [route-template-observability.md](references/route-template-observability.md) | Use low-cardinality labels after route resolution. |
| Topology, fallback, order, preflight, route coverage, or labels need proof | [router-validation-test-patterns.md](references/router-validation-test-patterns.md) | Produce a risk-focused transport test matrix. |

## Return And Stop

Return only triggered decisions: route/prefix inventory and owner; composition; middleware order/scope; generated/manual authority; 404/405/HEAD/OPTIONS/CORS mechanics; route-label semantics; assumptions; and focused tests for matches, conflicts, order, headers, preflight, generated coverage, and bounded labels.

Stop when path ownership, accepted fallback/CORS policy, middleware behavior, route-label rules, or generated/manual authority is unresolved; name the policy owner instead of inventing it. Reject implicit fallback/CORS defaults, raw-path labels, collision by registration order, unexplained middleware reordering, and transport work that chooses API or system policy.
