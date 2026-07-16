---
name: go-implementation-ownership
description: "Implementation ownership: Use when accepted decisions need Go package, file, dependency, cleanup, and proof placement, or when changed Go may violate those accepted boundaries. Own implementation placement and conformance; Skip when system topology, domain policy, Go semantics, or local readability is primary."
---

# Go Implementation Ownership

Load the [shared specialist contract](../specialist-contract.md). Keep source of truth, package/file responsibility, dependency direction, generated/manual authority, sequence, cleanup, and proof ownership coherent.

## Choose The Branch

- **Decision** — select when accepted behavior exists but implementation placement is absent or changing. Stop on any unresolved prerequisite; otherwise complete when every source, file, dependency, cleanup, and proof owner is assigned with forced consequences and blockers.
- **Review** — select when changed Go must conform to accepted placement. Load the [review selector](references/review/index.md) for one violated boundary. Complete when every affected ownership seam is dispositioned as a finding or no finding with the smallest correction and focused proof; missing placement stays in the decision branch.

Hand runtime topology to `go-system-architecture` and behavior policy to its domain skill.
