# Task Ledger V1

A persisted `tasks.md` contains only execution-changing state:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <unavailable required input and owner; omit when none>
Global constraints: <shared execution constraints; omit when none>

## Tasks

- [ ] T<ID>: <one reviewable postcondition>
  - Depends on: <ID and consumed output or gate | none>
  - Provides: <stable output or final outcome>
  - Packet: tasks/<ID>-<slug>.md
```

Every persisted task has one [Task Packet V1](task-packet-v1.md). Keep detailed
sources, boundaries, proof, writable surfaces, and reopen conditions in that
packet, not in the index. `Global constraints` references accepted owners instead
of copying their full constraints.

Each task is its own acceptance unit by default. When adjacent tasks cannot be
accepted independently, add one non-overlapping acceptance-unit row naming the
task IDs and the exact reason only their combined state is valid. After a fixed
unit closes, record one replaceable result:

```text
Accepted: <unit>; evidence: <claim-matched result>; candidate: <bounded diff, tree, or commit>
Blocked: <unit>; unverified: <claim>; evidence: <narrower result>; next owner: <owner and condition>; candidate: <identity or none>
```

Keep task status, dependencies, unit membership, receipts, completion, and
blockers in the index.
