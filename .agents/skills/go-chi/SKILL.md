---
name: go-chi
description: "Chi transport: Use when router composition, middleware, OpenAPI wiring, fallbacks, CORS, or route labels need a decision, or when changed chi routing needs conformance review. Own chi transport decisions and review; Skip when client-visible API semantics, system topology, or a general Go defect is primary."
---

# Go Chi

Load the [shared specialist contract](../specialist-contract.md). Reconstruct the affected route nodes from the composition root, generated and manual routes, middleware, fallbacks, and bounded labels; place them on one route tree, then reason about scope and order.

## Choose The Branch

- **Decision** — select when transport policy is absent or changing. Load the [decision selector](references/decision/index.md) only for a pressure that can change the result. Complete when shared Decision dispositions cover every affected route node, forced consequence, and focused proof.
- **Review** — select when changed chi code must conform to accepted transport policy. Load the [review selector](references/review/index.md) for the changed runtime judgment. Complete when the shared finding envelope accounts for every affected node; name any outside boundary or proof blocker with the smallest safe correction and proof. Missing policy ends this run with a named transport Decision handoff; conformance Review begins separately after the rule is accepted.

Hand resource or status semantics to `go-api-contract` and system topology to `go-system-architecture`.
