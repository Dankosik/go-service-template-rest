---
name: go-domain-invariant
description: "Domain invariants: Use when business terms, transitions, acceptance, violation, replay, or effect-order policy needs a decision, or when changed Go may violate accepted business rules. Own domain policy and conformance; Skip when transport, data/cache mechanics, security enforcement, or test structure is primary."
---

# Go Domain Invariant

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected invariants and transitions from accepted behavior, current accepting paths, state/effect owners, rejection surfaces, replay, and mixed-version constraints; state each in accepted terms with its false cases.

## Choose The Branch

- **Decision** — select when business policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every invariant and transition with rejection, effect boundary, forced consequence, and proof obligation explicit.
- **Review** — select when changed Go must preserve accepted domain policy. Load the [review selector](references/review/index.md) for the affected accepting path. Follow every affected accepting path into the shared finding envelope, naming any outside boundary or proof blocker with falsifying proof. Missing policy returns to the named Domain Decision owner.

Hand API representation to `go-api-contract`, data mechanics to `go-data-architecture`, and durable coordination to `go-distributed`.
