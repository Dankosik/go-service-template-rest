# Task Ledger V1

A persisted `tasks.md` contains only execution-changing state:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <unavailable required input and owner; omit when none>
Global constraints: <shared execution constraints; omit when none>

## Tasks

- [ ] T<ID>: <one independently acceptable postcondition>
  - Depends on: <ID and consumed output or gate | none>
  - Provides: <stable output or final outcome>
  - Packet: tasks/<ID>-<slug>.md
  - Execution: <native task or agent locator; omit until dispatched or when unavailable>
  - Resume: <remaining action and indispensable verified context; omit when recoverable from native state>
```

Every persisted task has one [Task Packet V1](task-packet-v1.md). Keep detailed
sources, boundaries, proof, writable surfaces, exclusive locks, and reopen
conditions in that packet, not in the index. `Global constraints` references
accepted owners instead of copying their full constraints.

Execution identifies the unit's current Lead; record the returned identity
after dispatch and replace it if the Lead changes. Read live execution status
from that native owner. Keep Resume only across an interruption when native
state cannot recover the next action; refresh or remove it when work resumes.
These fields are routing context, not acceptance evidence.

Each task is its own acceptance unit. Do not add a compound acceptance-unit row
to recover from over-splitting; reopen Planning and merge the packets. After a
fixed unit closes, record one replaceable result from the Lead-owned
[Acceptance Result V1](acceptance-result-v1.md):

```text
Accepted: <unit>; evidence: <claim-matched result>; candidate: <bounded diff, tree, or commit>
Blocked: <unit>; unverified: <claim>; evidence: <narrower result>; next owner: <owner and condition>; candidate: <identity or none>
```

Replace that result in place. Git owns superseded candidates, prior review
receipts, and repair history.

Check a task only after its Accepted result is recorded. Blocked leaves it
unchecked and cannot satisfy successful Completion. Clear Execution and Resume
when acceptance leaves no continuation for that unit.

Keep task status, dependencies, unit membership, receipts, completion, and
blockers in the index.
