---
name: go-chi
description: "Chi transport: Use for router composition, middleware, OpenAPI wiring, fallbacks, CORS, labels, or routing review. Own chi decisions; Skip client API semantics, system topology, or general Go defects."
---

# Go Chi

The router is a **composition**: one route tree where middleware order is semantics, not decoration.

`route tree -> middleware order and scope -> handler boundary -> fallbacks -> labels -> proof`

What runs before what decides what is protected, limited, and observed: authentication before body consumption, limits before expensive work, recovery around everything that can panic. Every fallback — not found, method not allowed, preflight, panic — is an explicit route node with an owner, and route labels stay a bounded set so telemetry cardinality remains a decision rather than an accident.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct the affected route nodes from the composition root, generated and manual routes, middleware, fallbacks, and bounded labels; place them on one route tree, then reason about scope and order.

Most of that tree already has an owner here: `internal/infra/http/router.go` builds the chain, the root mount, and the fallback policy, and `internal/infra/http/middleware_access_log.go` owns route identity. Read the affected owner before proposing a node.

## Choose The Branch

The branch decides what you return; both branches read from the same [reference selector](references/index.md), loading one entry by default and another only for an independent pressure.

- **Decision** — select when transport policy is absent or changing. Complete when shared Decision dispositions cover every affected route node with its position in the tree, forced consequence, and focused proof.
- **Review** — select when changed chi code must conform to accepted transport policy. Complete when the shared finding envelope accounts for every affected node; name any outside boundary or proof blocker with the smallest safe correction and proof. Missing policy returns to the named transport Decision owner.

Hand resource or status semantics to `go-api-contract` and system topology to `go-system-architecture`.
