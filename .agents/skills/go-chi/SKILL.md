---
name: go-chi
description: "Chi route composition. Use when a change adds, mounts, moves, or reviews routes, middleware, fallbacks, CORS, or bounded route identity."
metadata:
  invocation: model
  kind: method
---

# Go Chi

A chi server is one **route tree**. Middleware order and scope are observable
semantics.

Apply the [shared specialist contract](../../contracts/specialist-contract.md).
For every changed route, mount, middleware, fallback, or route label, build:

`RouteNode{node, owner, parent_scope, before_after_order, handler_boundary, fallback, label_source, proof}`

Emit each affected `RouteNode` inside the returned Decision or Review result.
When rejecting a proposed node, disposition it and emit the accepted replacement
record. A prose conclusion that does not fill every field is incomplete.

Start at `internal/infra/http/router.go`. Follow generated and manual
composition until each affected request reaches an operation handler or the
router fallback. `internal/infra/http/middleware_access_log.go` owns route
identity. No affected node or alternate serving path may remain implicit.

Reject a second router that bypasses the existing hardened or generated path.
The `proof` field names a route-tree walk for topology and an explicit order
oracle: cite the current chain-order proof when order is unchanged, or use a
before/after recorder when order changes. A status-only test proves neither.

Complete when every changed node appears in the returned records with one owner
and position, no parallel serving path remains, and the proof fails for the
wrong topology, order, fallback, or label source. Load the [reference
selector](references/index.md) only for the changed node pressure.
