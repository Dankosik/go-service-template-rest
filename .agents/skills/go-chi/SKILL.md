---
name: go-chi
description: "Chi transport: Use for routers, middleware, OpenAPI wiring, fallbacks, CORS, labels, or routing review. Own chi composition; Skip API meaning, topology, or general Go."
metadata:
  invocation: model
  kind: method
---

# Go Chi

The router is one composition where middleware order is semantics:

`route tree -> middleware order and scope -> handler boundary -> fallbacks -> labels -> proof`

Apply the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct
the affected nodes from generated and manual routes, middleware, fallbacks, and
bounded labels; account for every node and its position in the composed tree.

`internal/infra/http/router.go` owns the chain, root mount, and fallbacks;
`internal/infra/http/middleware_access_log.go` owns route identity. Read the
affected owner before proposing a node. Load the [reference
selector](references/index.md) when route-tree, middleware, fallback, or
request-path label behavior changes.

Hand resource or status semantics to `go-api-contract` and system topology to
`go-system-architecture`.
