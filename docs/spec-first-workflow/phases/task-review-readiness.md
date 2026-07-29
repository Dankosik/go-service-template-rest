# Task Review / Readiness

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only ledger-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent plan/readiness review.
- The shared independent-review trigger applies to a completed implementation ledger.
- A prior triggered `FAIL` was repaired and needs focused confirmation.

## Inputs

- Current fixed `tasks.md` candidate or diff.
- Ready spec and required design/test/rollout artifacts.
- Repository source/command evidence needed to check ownership and proof feasibility.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: the next task or real parallel wave is executable from closed inputs with adequate proof and no hidden decision that can invalidate it.
- `CONCERNS`: a bounded later risk or proof obligation may move after its owner, checkpoint, observable, and reopen condition are recorded and it cannot invalidate the next accepted result.
- `FAIL`: the ledger or an upstream decision must be repaired first.

## Review Method

Independently walk the next executable task or real parallel wave through its proof using cited sources and current inputs. Report hidden decisions, unavailable required inputs, unsafe concurrency, unowned deltas, or unprovable claims that can invalidate that work. Inspect later tasks only for a decision or dependency that can invalidate the next accepted result.

## Falsification Questions

Falsify the boundary in Review Method against the [Planning](planning.md) contract:

- Is the completion condition successful and observable, not merely “record blocker” or “run commands”?
- Can a fresh agent execute the next task or real parallel wave and prove its claimed postcondition from cited sources and current inputs without chat history or choosing behavior, mechanism, ownership, proof strategy, rollout policy, or concurrency?
- Can the named check and expected observable establish the claimed postcondition, including failure and negative paths where relevant?
- For a real parallel wave, does current evidence establish disjoint writable and mutable-resource ownership, preserve canonical/generated and migration/rollout coupling, and rule out an interface or assumption changed by one member and consumed by another?
- Can any known later decision, dependency, or unavailable input invalidate or make unusable the next accepted result?

## Stop Rule

For an internal checkpoint, return findings to the owning root; it repairs planning defects and re-reviews under the shared convergence contract in the same root session without a user handoff. Missing decisions reopen their upstream owner. For an explicitly user-requested standalone review, return findings and stop read-only.
