---
name: go-domain-invariant
description: "Domain invariants. Use when business acceptance, rejection, transitions, replay meaning, or effect order changes what states and moves are legal."
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

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
From every changed accepting path through its false case and replay, build
`InvariantRecord{rule, owner, accepting_paths, transitions, false_case,
rejection, effect_order, replay, mixed_version, proof}`. A rule is incomplete
until every accepting path and invalid move has a disposition.

## Choose The Branch

- **Decision** — load one matching [decision reference](references/decision/index.md)
  and cover every invariant and transition with rejection, effect boundary,
  forced consequence, and proof obligation.
- **Review** — load one matching [review reference](references/review/index.md)
  and follow every affected accepting path into the finding envelope with
  falsifying proof.

Complete when every invariant is falsifiable in accepted business terms, every
invalid move has one deterministic rejection surface, and proof would fail if
an alternate accepting path bypassed the rule.
