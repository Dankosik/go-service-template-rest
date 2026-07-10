---
name: go-chi-review
description: "Review Go chi routing changes for router ownership, chi-specific middleware and mount semantics, HTTP fallback policy, route observability labels, and OpenAPI/generated-route integration. Use whenever a Go diff touches github.com/go-chi/chi/v5 routers, middleware order or scope, NotFound/MethodNotAllowed/OPTIONS/CORS handling, route labeling, or manual wiring around generated handlers, even if the request is framed as a generic code review."
---

# Go Chi Review

## Trigger, Scope, And Boundary

Review changed `github.com/go-chi/chi/v5` router topology, registration, mounts, middleware scope/order, fallbacks, HTTP capability, route labels, generated/manual ownership, and transport startup/readiness behavior for concrete merge-risk runtime or contract regressions.

Stay chi-specific and review-only. Do not redesign service architecture, payload/business semantics, DB/cache, or deep security/reliability policy unless routing cannot be corrected locally; hand the decisive question to its owner.

## Chi Invariants

1. Router construction is deterministic and startup-safe: global `Use` precedes routes on a mux; mounts/subtrees have one non-nil owner; static/param/wildcard/generated/manual overlaps are deliberate.
2. `Use`, `With`, `Group`, `Route`, and `Mount` preserve intended middleware order and exact scope; route-dependent logic reads final route context only after routing.
3. Alternate method/path probing uses fresh `chi.NewRouteContext()` state; custom `Match`/`Find`/`Allow`/`OPTIONS` logic never mutates the live request context.
4. `404`, `405`, `Allow`, `HEAD`, `OPTIONS`, CORS, and preflight behavior advertise only capabilities the router actually serves; chi does not imply `HEAD` from `GET` without explicit support.
5. Metrics, traces, logs, and spans share bounded route-template identity; raw paths, IDs, wildcard captures, and user input never become cardinality labels.
6. OpenAPI/generated handlers and manual routes keep one source and path owner; generated files remain no-touch derived output and surrounding middleware/fallback policy stays consistent.
7. Validation exercises router construction and concrete method/path behavior, including startup panic, fallback, route context, and bounded-label cases—not just happy-path handlers.

## Symptom-Driven Reference Selector

Load at most one reference by default; use more only for independent pressures. State which runtime judgment it changes.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Late `Use`, construction order, `Route`/`Mount`, wildcard/subtree ownership, duplicate owner, nil mount. | [chi-router-registration-hazards.md](references/chi-router-registration-hazards.md) | Treat startup and subtree ownership as runtime defects instead of style or generic duplicates. |
| Middleware order/scope changes across `Use`, `With`, `Group`, `Route`, or `Mount`. | [middleware-order-and-scope.md](references/middleware-order-and-scope.md) | Prove exact coverage/order instead of assuming nested refactors preserve behavior. |
| `RouteContext`, `RoutePattern`, `Match`, `Find`, custom `Allow`/`OPTIONS`, or method probing. | [route-context-and-match-probing.md](references/route-context-and-match-probing.md) | Read route identity after resolution and probe with isolated contexts. |
| `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, CORS, preflight, or fallback wrappers. | [http-fallback-head-options-cors.md](references/http-fallback-head-options-cors.md) | Verify actual router capability and contract instead of inferred/hardcoded method support. |
| OpenAPI/generated chi handlers, generated/manual overlap, subtree wrappers, or generated no-touch files. | [generated-and-manual-route-drift.md](references/generated-and-manual-route-drift.md) | Preserve one source/route owner and policy parity instead of shadow routes or generated edits. |
| Metrics/traces/logs/span names, `http.route`, route extraction, or unmatched labels. | [route-observability-labels.md](references/route-observability-labels.md) | Use shared bounded route-template labels instead of raw or inconsistent identities. |

## Evidence And Shared Finding Envelope

Inspect router construction, registration sequence, subtree ownership, middleware stack, live/probe contexts, method capability, fallback handlers, generated authority, telemetry timing/labels, and constructor/request tests. Each finding adds the chi defect, runtime or contract-visible impact, smallest safe correction, governing chi/contract evidence, and focused constructor or `httptest` validation.

Use the [shared review finding envelope](../../../docs/subagent-contract.md#shared-review-finding-envelope). `critical` is a confirmed merge-unsafe routing/startup failure; `high` is strong evidence of major route or HTTP-policy drift. Use `Reference` only for concrete contract, design, generated-source, or chi behavior evidence.

## Success, Escalation, And Stop Conditions

Success means findings are chi-runtime-specific, exact-source anchored, merge-risk ordered, and prove route ownership, startup, middleware, HTTP fallback/capability, generated parity, or bounded observability behavior.

Escalate non-local router ownership/topology or middleware strategy to `go-chi-spec`; client-visible method/status/fallback/CORS semantics to API contract design; telemetry contract to observability design; startup/shutdown/fallback policy to reliability design; and broader seam drift to integrated design. Stop rather than smuggle those decisions into a local route fix.
