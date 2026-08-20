---
name: go-domain-invariant
description: "Domain rules: Use for business transitions, violations, replay, or effect order. Own invariants; Skip transport, data, security, and tests."
metadata:
  invocation: model
  kind: method
---

# Go Domain Invariant

A business rule is an **invariant**: a statement about state and transitions that stays true under every accepting path, replay, and version mix — or it is a wish, not a rule.

`accepted terms -> states and transitions -> acceptance conditions -> rejection surfaces -> effect order -> replay -> proof`

State each invariant in accepted business terms with the input, sequence, or
replay that falsifies it and the surface that rejects the attempt. The domain
owns effect order, duplicate meaning, and out-of-order meaning.

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct affected invariants and transitions from accepted behavior, current accepting paths, state/effect owners, rejection surfaces, replay, and mixed-version constraints.

## Choose The Branch

- **Decision** — load one matching [decision reference](references/decision/index.md)
  and cover every invariant and transition with rejection, effect boundary,
  forced consequence, and proof obligation.
- **Review** — load one matching [review reference](references/review/index.md)
  and follow every affected accepting path into the finding envelope with
  falsifying proof.

This skill owns which rule must hold and what its violation means. `go-data-architecture` owns where truth lives and how a constraint and its migration are shaped, `go-api-contract` owns how the outcome reaches a client, and `go-distributed` owns durable coordination. Choosing a database constraint over an application convention is a domain decision made here; writing it is theirs.
