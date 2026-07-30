---
name: go-implementation-ownership
description: "Implementation ownership: Use for Go package/file/dependency, cleanup, proof placement, or boundary review. Own code placement; Skip system topology, domain policy, Go semantics, or local readability."
---

# Go Implementation Ownership

Load the [shared specialist contract](../specialist-contract.md). Reconstruct changed responsibilities and executable paths from accepted behavior, current callers and owners, generated/manual sources, wiring, tests, and cleanup surfaces; assign one owner for each source of truth, package/file responsibility, dependency direction, sequence, and proof surface, exposing ownerless, duplicated, or competing paths.

## Choose The Branch

- **Decision** — select when accepted behavior exists but implementation placement is absent or changing. Stop on any unresolved prerequisite; otherwise complete when shared Decision dispositions cover every responsibility and path with forced consequences explicit.
- **Review** — select when changed Go must conform to accepted placement. Load the [review selector](references/review/index.md) for one violated boundary. Account for every affected owner and competing path through the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and focused proof. Missing placement returns to the named Ownership Decision owner.

Hand runtime topology to `go-system-architecture` and behavior policy to its domain skill.
