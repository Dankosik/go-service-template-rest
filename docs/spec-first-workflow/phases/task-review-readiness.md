# Task Review / Readiness

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only ledger-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent plan/readiness review.
- Structured or orchestrated work has a completed implementation ledger.
- Implementation is high-impact, broad, delegated, hard to reverse, or otherwise difficult for the planner to falsify.
- A repaired ledger needs confirmation that a prior blocker is closed.

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

## Review Questions

### Coverage And Traceability

- Does every in-scope accepted obligation map to an executable task and adequate proof or an evidence-backed no-implementation disposition, and does every task trace back to an accepted obligation and own an outcome rather than an activity?
- Does the ledger preserve accepted examples or scenarios that define behavior or proof while excluding rationale, rejected alternatives, non-normative examples, and future ideas from implementation scope? Does each no-implementation disposition cite current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus a proving surface or objective recheck condition?

### Task Ownership And Boundaries

- Are `Global constraints`, `Owner/surface/resources`, generated-source order, cleanup, non-duplicated task deltas, and exact handoffs clear without duplicating task prose? Are mutable or exclusive databases, ports, environments, migration targets, destructive fixtures, locks, generated pairs, and proof resources structurally visible rather than hidden in prose?
- Does any task span separable ownership, review, failure/recovery, rollback, or proof domains that can each end in a valid provable state? Split that oversized outcome; do not split coupled work or use file count, estimated minutes, or desired Worker count as a sizing rule.
- Are source anchors narrow enough and execution-critical constraints carried into tasks so a fresh implementer need not reconstruct them from broad documents or chat? Are discovery boundaries used only when unavoidable, with bounded inspection and a deterministic placement rule or canonical source that resolves the file choice?
- Does any `Reopen if` entry hide a known current decision, evidence, or mandatory-input gap, or invent a generic future trigger where no objective invalidation condition exists?

### Waves And Dependencies

- Are dependency edges true execution or proof gates rather than document order or review preference, and do independent roots remain unchained?
- When a real multi-task wave exists, does current evidence establish disjoint writable and mutable-resource ownership, preserve canonical/generated and migration/rollout coupling, and avoid an interface or assumption changed by one member and consumed by another? Sequential tasks need no wave record.

### Proof And External Inputs

- Does each proof name its claim, check, and expected observable, and can that observable establish the claim, including failure and negative paths where relevant?
- Would implementation have to choose behavior, design, test strategy, or rollout policy?
- Is every external input required by the next task or wave available now? Later unavailable inputs need an owner and checkpoint; they block only when they can invalidate the next result or a final-completion claim.

### Cold Completion

- Is the completion condition successful and observable, not merely “record blocker” or “run commands”?
- Can a fresh agent execute the next task or real parallel wave and its proof from cited sources and current inputs without chat history or choosing behavior, ownership, proof strategy, or concurrency?
- Is the ledger smaller and clearer than the work it coordinates?

Do not require fields or phase files that cannot change execution or evidence.

## Stop Rule

For an internal checkpoint, return findings to the owning root; it repairs planning defects and re-reviews under the shared convergence contract in the same root session without a user handoff. Missing decisions reopen their upstream owner. For an explicitly user-requested standalone review, return findings and stop read-only.
