---
name: go-chi
description: "Chi transport: Use when router composition, middleware, OpenAPI wiring, fallbacks, CORS, or route labels need a decision, or when changed chi routing needs conformance review. Own chi transport decisions and review; Skip when client-visible API semantics, system topology, or a general Go defect is primary."
---

# Go Chi

Load the [shared specialist contract](../specialist-contract.md). Keep route ownership, middleware scope, fallback behavior, generated/manual authority, and bounded template labels coherent.

## Choose The Branch

- **Decision** — select when transport policy is absent or changing. Load the [decision selector](references/decision/index.md) only for a pressure that can change the result. Complete when the routing decision, forced consequences, focused proof, and blockers are explicit.
- **Review** — select when changed chi code must conform to accepted transport policy. Load the [review selector](references/review/index.md) for the changed runtime judgment. Complete when every in-scope surface is dispositioned as a finding or no finding, with the smallest safe correction and proof; unresolved policy stays in this skill's decision branch.

Hand resource or status semantics to `go-api-contract` and system topology to `go-system-architecture`.
