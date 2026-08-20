# Planning Ledger Contract

Use `tasks.md` only when Implementation has multiple acceptance units,
dependencies, or a real actor/session boundary.

## Ledger Shape

Keep the ledger centered on postconditions and proof:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <unavailable required input and owner; omit when none>
Global constraints: <only constraints shared by several tasks; omit when none>

- [ ] T1: <verifiable postcondition>
  - Depends on: <task ID and consumed output or gate; or none>
  - Constraints/references: <important accepted constraints and narrow sources>
  - Done when: <claim; command/check; expected observable>
  - Owner/surface/resources: <only when non-obvious or concurrency-sensitive>
  - External gate: <required non-ledger input and owner; omit when none>
  - Reopen if: <objective invalidation condition and owner; omit when none>
```

Add only fields that change execution. Ordinary owners, paths, and repository
structure remain discoverable from current code. Record a writable boundary,
exclusive resource, generated owner, handoff shape, or exact order only when it
is non-obvious or two ready units could otherwise conflict.

When one entry is too large for its executor, move only that entry's outcome,
constraints, references, optional owner/surface/resources, and proof to
`tasks/<ID>-<slug>.md`. Keep status, checkboxes, dependencies, acceptance units,
receipts, completion, and blockers in `tasks.md`.

## Task Boundaries

Each task must leave the repository and any deployment or migration state it
creates internally consistent and provable without unfinished companion work.
Group canonical source, generated output, tests, fixtures, required wiring,
documentation, and replacement cleanup needed for that postcondition.

Prefer an end-to-end behavior through its real entry point. Split only for a
distinct postcondition, owner, required dependency, rollout or migration
sequence, independently valid handoff, materially different proof boundary, or
a separate result that can be accepted on its own. File count, desired agent
count, and estimated time do not create a task boundary.

## Acceptance Units

Each implementation task is one acceptance unit by default. Group adjacent
tasks only when none can be accepted independently and one Lead must integrate
and prove them together:

```markdown
## Acceptance units
- A1: T2, T3 — <why only the combined state is valid and provable>
```

Membership cannot overlap. A Lead may implement a unit directly or create any
useful delegated subtree; internal delegation creates no acceptance state.

## Dependencies And Concurrency

Record `Depends on` only when a task consumes another task's concrete output or
must cross its safety or proof gate. Name the consumed output or condition.
Shared labels, packages, broad final checks, or possible merge effort do not by
themselves create an edge.

The Ledger Orchestrator may run currently ready units concurrently when their
accepted dependencies, files, mutable resources, interfaces, and assumptions
are genuinely independent. No persisted wave or execution map is required. If
independence cannot be established cheaply, run them serially or improve the
task split.

## Execution Readiness

A fresh Lead must be able to execute the next unit from the ledger, cited
sources, and current repository without chat history or inventing behavior,
mechanism, ownership, proof strategy, rollout policy, or authority. State the
postcondition as the caller or operator observes it; keep important constraints
close to that task and point to real code, tests, contracts, or accepted
artifacts instead of copying them.

Known required external input belongs in `External gate`; when it is already
unavailable and blocks all work, use `Blocked stop`. A future invalidation
condition belongs in `Reopen if`, not in an implementation procedure.

## Implementation Transitions

The Acceptance-Unit Lead records one transition after the fixed unit satisfies
its postcondition, mapped proof, and any triggered fresh review:

```markdown
  - Accepted: <unit or task IDs>; evidence: <command or source and result>; candidate: <current bounded diff, commit, or tree>
```

Use `current bounded diff` in the same checkout and an immutable commit or tree
only across a checkout or integration boundary. Check every member task in the
same edit; after the final unit, set `status: done` and verify `Completion`.

When required proof or authority is unavailable, leave the unit unchecked and
keep one replaceable line:

```markdown
  - Blocked: <unit or task IDs>; unverified: <claim>; evidence: <narrower evidence>; next owner: <owner and condition>; candidate: <bounded diff, commit, tree, or none>
```

Keep `status: ready` while another unit or agent-owned recovery is executable;
use `status: blocked` only when neither remains. A delegated result, reviewer
return, candidate handoff, or attempted action is not ledger state. Replace the
blocker with the accepted receipt after closure rather than accumulating an
attempt log.

## Stop Rule

The ledger is ready only when every accepted implementation obligation has one
task disposition, every task is independently valid at its boundary, the next
unit can start from closed inputs, every dependency names a real consumed
output or gate, and each `Done when` can falsify its postcondition. Detail that
does not change execution stays in its canonical code, contract, test, or
artifact owner.
