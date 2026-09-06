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

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
Trace every changed route, mount, middleware, fallback, or route label. When
comparing interacting nodes, covering alternate serving paths, or handing off
a Decision or Review, record each affected node as:

`RouteNode{node, owner, parent_scope, before_after_order, handler_boundary, fallback, label_source, proof}`

Include those records in the existing artifact or required result. When
rejecting a proposed node, identify its accepted replacement. A single local
node can keep its position, owner, and order proof in code or the existing task
artifact without a separate record.

Start at `internal/infra/http/router.go`. Follow generated and manual
composition until each affected request reaches an operation handler or the
router fallback. `internal/infra/http/middleware_access_log.go` owns route
identity. No affected node or alternate serving path may remain implicit.

Reject a second router that bypasses the existing hardened or generated path.
Proof names a route-tree walk for topology and an explicit order
oracle: cite the current chain-order proof when order is unchanged, or use a
before/after recorder when order changes. A status-only test proves neither.

Complete when every changed node has one owner and position, no parallel
serving path remains, and the proof fails for the
wrong topology, order, fallback, or label source. Load the [reference
selector](references/index.md) only for the changed node pressure.
