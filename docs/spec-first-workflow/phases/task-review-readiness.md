# Task Review / Readiness

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only ledger-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent plan/readiness review.
- The shared independent-review trigger applies to a completed implementation ledger.
- A prior triggered `FAIL` was repaired and needs focused confirmation.

## Inputs

- Current fixed `tasks.md` candidate. For a focused re-review, also provide the material candidate diff and retained unaffected dispositions.
- Ready spec and required design/test/rollout artifacts.
- Repository source/command evidence needed to check ownership and proof feasibility.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: the next task or real parallel wave is executable from closed inputs with adequate proof and no hidden decision that can invalidate it.
- `CONCERNS`: a bounded later risk or proof obligation may move after its owner, checkpoint, observable, and reopen condition are recorded and it cannot invalidate the next accepted result.
- `FAIL`: executor simulation of the next task or wave reaches an unrecorded behavior, mechanism, placement, ownership, test/proof strategy, rollout, or concurrency choice; a mandatory input or gate is unavailable; a write/resource/interface conflict makes the wave unsafe; or the named check cannot establish the claimed postcondition.

## Review Method

Independently dry-run the next executable task or real parallel wave from task selection through acceptance using only the fixed ledger, cited sources, and current repository evidence. Resolve every prerequisite and handoff, locate the writable surface through the named owner or discovery rule, follow required canonical ordering, and trace the postcondition through the named check and its oracle to the real path. The simulation is carrier-neutral: it validates the accepted task contract and current execution prerequisites; Implementation chooses the execution carrier. For a wave, simulate every pair from the same accepted base and attempt to falsify writable-surface, mutable-resource, interface, and assumption independence. At simulated acceptance, test the resulting repository and any deployment or migration state against Planning's split-boundary rule; unfinished coupled companion work fails the task boundary.

Falsify Planning's link-check by tracing each changed contract or authority in the simulated task or wave to its current producers, consumers, derived outputs, proof carriers, and replacement surfaces. A required companion must be inside the task's recorded surface or discovery rule, behind a valid dependency or handoff whose intermediate state satisfies Planning's split rule, or supported by a proved-unchanged disposition; otherwise return `FAIL` at the first missing boundary and name the upstream owner.

For each evidence-valid task path or wave member, report the earliest point where execution would need an unrecorded choice or unavailable input, with its exact anchor and owning repair or reopen boundary. Stop that path at the blocker, continue the remaining bounded simulation where evidence is still valid, and consolidate the independent material blockers reached. The reviewer does not reslice tasks, author replacement wording, or choose the missing decision. Inspect later tasks only for a decision or dependency capable of invalidating the next accepted result.

## Falsification Questions

Falsify the boundary in Review Method against the [Planning](planning.md) contract:

- Is the completion condition successful and observable, not merely “record blocker” or “run commands”?
- Can a fresh agent execute the next task or real parallel wave and prove its claimed postcondition from cited sources and current inputs without chat history or choosing behavior, mechanism, ownership, proof strategy, rollout policy, or concurrency?
- Can the named check and expected observable establish the claimed postcondition, including failure and negative paths where relevant?
- For a real parallel wave, does current evidence establish disjoint writable and mutable-resource ownership, preserve canonical/generated and migration/rollout coupling, and rule out an interface or assumption changed by one member and consumed by another?
- Can any known later decision, dependency, or unavailable input invalidate or make unusable the next accepted result?

## Stop Rule

For an internal checkpoint, return findings to the owning root; it repairs planning defects and re-reviews under the shared convergence contract in the same root session without a user handoff. Missing decisions reopen their upstream owner. For an explicitly user-requested standalone review, return findings and stop read-only.
