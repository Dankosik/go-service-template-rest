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
  - Depends on: <ID and consumed output or gate | none; annotate acceptance or a named external effect when later than implementation>
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

Tasks retain their outcome boundaries; checkboxes record implementation, while
Completion records final validation. Return one replaceable
[Acceptance Result V1](acceptance-result-v1.md) per task, and a final result for
Completion after assembly:

```text
Implemented: <unit>; verification: pending final validation; candidate: <bounded diff, tree, or commit>
Accepted: Completion; evidence: <claim-matched result>; candidate: <identity>
Blocked: <unit>; unverified: <claim>; evidence: <narrower result>; next owner: <owner and condition>; candidate: <identity or none>
```

Replace that result in place. Git owns superseded candidates, prior review
receipts, and repair history.

Check a task after its Implemented result is recorded and integrated. A Blocked implementation stays unchecked. An implemented task
may remain checked while final validation is blocked; it supplies no acceptance
claim. Clear Execution when its writers stop. Mark the goal done only after
final Accepted proves Completion. Keep a later acceptance or effect gate visible
even when its implementation is checked.

Keep task status, dependencies, unit membership, receipts, completion, and
blockers in the index.
