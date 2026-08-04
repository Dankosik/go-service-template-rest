---
name: go-domain-invariant
description: "Domain invariants: Use for business terms, transitions, acceptance, violations, replay, effect order, or review. Own domain policy; Skip transport, data/cache mechanics, security, or test structure."
---

# Go Domain Invariant

A business rule is an **invariant**: a statement about state and transitions that stays true under every accepting path, replay, and version mix — or it is a wish, not a rule.

`accepted terms -> states and transitions -> acceptance conditions -> rejection surfaces -> effect order -> replay -> proof`

State every invariant in accepted business terms together with its false cases: which input, sequence, or replay would violate it, and which surface rejects that attempt. Effect order belongs to the domain — what may be observed before acceptance commits, and what a duplicate or out-of-order arrival does to meaning — rather than to whichever transport happens to deliver it.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected invariants and transitions from accepted behavior, current accepting paths, state/effect owners, rejection surfaces, replay, and mixed-version constraints.

## Choose The Branch

- **Decision** — select when business policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every invariant and transition with rejection, effect boundary, forced consequence, and proof obligation explicit.
- **Review** — select when changed Go must preserve accepted domain policy. Load the [review selector](references/review/index.md) for the affected accepting path. Follow every affected accepting path into the shared finding envelope, naming any outside boundary or proof blocker with falsifying proof. Missing policy returns to the named Domain Decision owner.

This skill owns which rule must hold and what its violation means. `go-data-architecture` owns where truth lives and how a constraint and its migration are shaped, `go-api-contract` owns how the outcome reaches a client, and `go-distributed` owns durable coordination. Choosing a database constraint over an application convention is a domain decision made here; writing it is theirs.
