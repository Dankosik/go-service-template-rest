# Task Ledger V1

A persisted `tasks.md` contains only execution-changing state:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <unavailable required input and owner; omit when none>
Global constraints: <shared execution constraints; omit when none>

- [ ] T1: <verifiable postcondition>
  - Depends on: <task ID and consumed output or gate; or none>
  - Constraints/references: <narrow accepted sources>
  - Done when: <claim; command/check; expected observable>
  - Owner/surface/resources: <only when non-obvious or conflict-sensitive>
  - External gate: <required non-ledger input and owner; omit when none>
  - Reopen if: <objective invalidation condition and owner; omit when none>
```

When adjacent tasks cannot be accepted independently, add one non-overlapping
acceptance-unit row naming the task IDs and why only the combined state is valid.
After a fixed unit closes, record one replaceable result:

```text
Accepted: <unit>; evidence: <claim-matched result>; candidate: <bounded diff, tree, or commit>
Blocked: <unit>; unverified: <claim>; evidence: <narrower result>; next owner: <owner and condition>; candidate: <identity or none>
```

Keep task status, dependencies, unit membership, receipts, completion, and
blockers in the index even when one task's details move to `tasks/<ID>-<slug>.md`.
