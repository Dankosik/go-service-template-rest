---
name: go-domain-invariant
description: "Domain invariants: Use when business terms, transitions, acceptance, violation, replay, or effect-order policy needs a decision, or when changed Go may violate accepted business rules. Own domain policy and conformance; Skip when transport, data/cache mechanics, security enforcement, or test structure is primary."
---

# Go Domain Invariant

Load the [shared specialist contract](../specialist-contract.md). Keep terms, invariant ownership, legal transitions, acceptance/rejection meaning, effect boundaries, false cases, and mixed-version behavior coherent.

## Choose The Branch

- **Decision** — select when business policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when every invariant, transition, violation outcome, forced consequence, proof obligation, and blocker is explicit.
- **Review** — select when changed Go must preserve accepted domain policy. Load the [review selector](references/review/index.md) for the affected accepting path. Complete when every affected transition and side effect is dispositioned as a finding or no finding with falsifying proof; missing policy stays in the decision branch.

Hand API representation to `go-api-contract`, data mechanics to `go-data-architecture`, and durable coordination to `go-distributed`.
