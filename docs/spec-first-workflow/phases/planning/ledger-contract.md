# Planning Ledger Contract

Use only after Planning selects a persisted [Task Ledger
V1](../../interfaces/task-ledger-v1.md).

## Unit Boundaries

Each task leaves repository and any deployment/migration state internally
consistent and provable without unfinished companion work. Group canonical
source, generated output, wiring, tests/fixtures, documentation, and replacement
cleanup required by one postcondition.

Prefer one end-to-end behavior through its real entry point. Split only for a
distinct postcondition, owner, consumed dependency, rollout/migration sequence,
independently valid handoff, or separately acceptable proof boundary. File
count, desired agent count, and estimated time do not create a unit.

Group adjacent tasks into one non-overlapping acceptance unit only when none can
be accepted independently and one Lead must integrate/prove the combined state.

## Dependencies And Readiness

Record `Depends on` only when a task consumes another task's concrete output or
must cross its safety/proof gate; name that output or condition. The next unit
must be executable from the ledger, cited sources, and current repository
without chat history or a new behavior, mechanism, ownership, proof, rollout,
authority, or concurrency decision.

Currently ready units may run concurrently only when dependencies, files,
mutable resources, interfaces, and assumptions are genuinely independent.
Otherwise run serially; no persisted wave protocol is required.

## Acceptance Transition

After the fixed unit satisfies postcondition, mapped proof, and required review,
the Acceptance-Unit Lead writes exactly one [Task Ledger
V1](../../interfaces/task-ledger-v1.md) `Accepted` or `Blocked` result.
Check every member task in the same edit. A delegated result, review return,
candidate handoff, or attempted action is not ledger state.

Keep `status: ready` while another unit or owner-held recovery is executable;
use `blocked` only when none remains. After the final accepted unit, verify the
global Completion condition before `done`.

## Stop Rule

The ledger is ready when every accepted obligation has one unit disposition,
each boundary is independently valid, every dependency names a consumed output
or gate, and each `Done when` can falsify its postcondition.
