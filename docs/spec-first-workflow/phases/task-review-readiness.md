# Task Review / Readiness

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only ledger-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent plan/readiness review.
- Structured or orchestrated work has a completed implementation ledger.
- Implementation is high-impact, broad, delegated, hard to reverse, or otherwise difficult for the planner to falsify.
- A repaired ledger needs confirmation that a prior blocker is closed.

## Inputs

- Exact `tasks.md` revision.
- Ready spec and required design/test/rollout artifacts.
- Repository source/command evidence needed to check ownership and proof feasibility.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: every mandatory task and proof through current completion is executable from closed inputs in safe planned waves, with no hidden execution work, current-phase defect, unowned question, or uncovered affected lens.
- `CONCERNS`: a bounded risk or downstream proof obligation still needs explicit owner disposition and fresh review; it does not permit implementation.
- `FAIL`: the ledger or an upstream decision must be repaired first.

## Review Questions

- Does every in-scope accepted obligation map to an executable task and adequate proof or an evidence-backed no-implementation disposition, and does every task trace back to an accepted obligation and own an outcome rather than an activity?
- Does the ledger preserve accepted examples or scenarios that define behavior or proof while excluding rationale, rejected alternatives, non-normative examples, and future ideas from implementation scope? Does each no-implementation disposition cite current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus a proving surface or objective recheck condition?
- Are dependency edges true execution or proof gates rather than document order or review preference, and do independent roots remain unchained? Are `Global constraints`, `Owner/surface/resources`, generated-source order, cleanup, non-duplicated task deltas, and exact handoffs clear without duplicating task prose? Are mutable or exclusive databases, ports, environments, migration targets, destructive fixtures, locks, generated pairs, and proof resources structurally visible rather than hidden in prose?
- Does every task appear exactly once in the earliest safe planned wave? For each multi-task wave, does current evidence positively establish disjoint writable or discovery boundaries, no canonical/generated or migration/rollout coupling, no shared mutable or non-concurrent proof resource, and no interface, invariant, or assumption changed by one member and consumed by another? Absence of dependency edges alone is not proof; unknown independence requires a one-task wave.
- Does any task span separable ownership, review, failure/recovery, rollback, or proof domains that can each end in a valid provable state? Split that oversized outcome; do not split coupled work or use file count, estimated minutes, or desired Worker count as a sizing rule.
- Does each proof name its claim, check, and expected observable, and can that observable establish the claim, including failure and negative paths where relevant?
- Are source anchors narrow enough and execution-critical constraints carried into tasks so a fresh implementer need not reconstruct them from broad documents or chat? Are discovery boundaries used only when unavoidable, with bounded inspection and a deterministic placement rule or canonical source that resolves the file choice?
- Does any `Reopen if` entry hide a known current decision, evidence, or mandatory-input gap, or invent a generic future trigger where no objective invalidation condition exists?
- Is the completion condition successful and observable, not merely “record blocker” or “run commands”?
- Would implementation have to choose behavior, design, test strategy, or rollout policy?
- Cold completion: can a fresh agent execute every mandatory path and planned wave from each dependency root through final validation from cited sources and currently available inputs, without chat history or choosing behavior, schema, values, ownership, proof strategy, or the initial concurrency schedule?
- Is every known external input on a mandatory path available now? If not, is its dependent task and claim excluded from current completion and routed separately? A ledger cannot receive `PASS subject to gates` when a gate can block mandatory completion.
- Is any task both mandatory for completion and permitted to remain blocked? If so, return `FAIL` and reopen the accepted-outcome owner; the reviewer does not narrow scope.
- Is the ledger smaller and clearer than the work it coordinates?

Do not require fields or phase files that cannot change execution or evidence.

## Stop Rule

For an internal checkpoint, return findings to the owning root; it repairs planning defects and re-reviews under the shared convergence contract in the same root session without a user handoff. Missing decisions reopen their upstream owner. For an explicitly user-requested standalone review, return findings and stop read-only.
