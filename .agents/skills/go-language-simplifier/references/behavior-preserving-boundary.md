# Behavior-Preserving Boundary

## Load When
Load this when a diff merges branches, error paths, response paths, or cleanup
sequences, or when the stated goal is deduplication with no narrower pressure.

## Decide
- A simplification that changes what a caller, operator, or test can observe is
  not a simplification. Report it as the behavior change it is; the semantics
  belong to `go-idiomatic`, not to this lane.
- Ask which distinction the merged code used to make: status, retry, error
  identity, audit or notification, nil versus empty, who may mutate a shared
  slice or map after the call, cleanup order, which error wins. Repetition whose
  branches carry different contracts is not debt.
- Error identity is a caller-visible contract. A `map[error]T` lookup uses key
  equality, so replacing `errors.Is` checks with a lookup can stop matching
  wrapped errors. Inspect the target's `.golangci.yml` before attributing
  coverage to a linter; its presence does not prove this semantic distinction.
- Resolve supported multi-error APIs through [Go Modern
  Version](../../go-modern-version/SKILL.md). A cleanup that must carry its own
  failure alongside the operation failure preserves both required inspectable
  identities; dropping one to `%v` needs an accepted semantic reason.
- Side effects that must follow a durable boundary stay visible at the call site.
  A loop over steps is a good shape when the steps are peers and a poor one when
  their order is the contract.

## Reject
A shared writer that answers cancellation, validation, conflict, and internal
failure with one status: the response path is shorter and the caller and the
operator have both lost a distinction they act on.

## Prove
Prove the distinction that was collapsed, not the path that still works: one case
per surviving behavior class, `errors.Is`/`errors.As` still matching where a
caller relies on it, and cleanup or audit still ordered against the durable
boundary. Green broad tests that never separated the branches are not evidence.
