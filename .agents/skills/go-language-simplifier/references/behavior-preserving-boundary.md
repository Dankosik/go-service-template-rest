# Behavior-Preserving Boundary

Behavior Change Thesis: this file makes the model check the observable
distinctions a cleanup collapses before judging it simpler, so it stops
approving a dedupe that gives branches with different caller-visible outcomes one
path — and stops reporting a behavior change as a readability win.

## When To Load
Load this when a diff merges branches, error paths, response paths, or cleanup
sequences, or when the stated goal is deduplication with no narrower pressure.

## Decision Rubric
- A simplification that changes what a caller, operator, or test can observe is
  not a simplification. Report it as the behavior change it is; the semantics
  belong to `go-idiomatic`, not to this lane.
- Ask which distinction the merged code used to make: status, retry, error
  identity, audit or notification, nil versus empty, who may mutate a shared
  slice or map after the call, cleanup order, which error wins. Repetition whose
  branches carry different contracts is not debt.
- Error identity is a caller-visible contract, and the gate covers less of it
  than it appears to. `errorlint` fails `==`, `switch`, and type assertions on
  errors; `wrapcheck` fails an unwrapped external error. None of them sees a
  `map[error]T` lookup — verified against this repository's `.golangci.yml` — so
  a "table instead of a chain" refactor silently stops matching wrapped errors.
- On Go 1.26.5, `fmt.Errorf` accepts more than one `%w` and `errors.Join` wraps a
  set. A cleanup that must carry a cleanup failure alongside the operation
  failure can keep both identities inspectable; dropping one to `%v` is a loss to
  justify, not a default.
- Side effects that must follow a durable boundary stay visible at the call site.
  A loop over steps is a good shape when the steps are peers and a poor one when
  their order is the contract.

## Reject
A shared writer that answers cancellation, validation, conflict, and internal
failure with one status: the response path is shorter and the caller and the
operator have both lost a distinction they act on.

## Validation Shape
Prove the distinction that was collapsed, not the path that still works: one case
per surviving behavior class, `errors.Is`/`errors.As` still matching where a
caller relies on it, and cleanup or audit still ordered against the durable
boundary. Green broad tests that never separated the branches are not evidence.
