---
name: go-chi
description: "Chi transport: Use for routers, middleware, OpenAPI wiring, fallbacks, CORS, labels, or routing review. Own chi composition; Skip API meaning, topology, or general Go."
---

# Go Chi

The router is a **composition**: one route tree where middleware order is semantics, not decoration.

`route tree -> middleware order and scope -> handler boundary -> fallbacks -> labels -> proof`

Order decides what is protected, limited, and observed. Every fallback is an
owned route node, and route labels remain bounded.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct the affected route nodes from the composition root, generated and manual routes, middleware, fallbacks, and bounded labels; place them on one route tree, then reason about scope and order.

`internal/infra/http/router.go` owns the chain, root mount, and fallbacks;
`internal/infra/http/middleware_access_log.go` owns route identity. Read the
affected owner before proposing a node.

## Choose The Branch

Load the [reference selector](references/index.md) for middleware or route-tree
changes, fallbacks, or request-path-derived telemetry labels.

- **Decision** — select when transport policy is absent or changing. Complete when shared Decision dispositions cover every affected route node with its position in the tree, forced consequence, and focused proof.
- **Review** — select when changed chi code must conform to accepted transport policy. Complete when the shared finding envelope accounts for every affected node.

Hand resource or status semantics to `go-api-contract` and system topology to `go-system-architecture`.
