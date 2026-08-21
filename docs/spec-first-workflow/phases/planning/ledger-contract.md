# Planning Ledger Contract

Use only after Planning selects a persisted [Task Ledger
V1](../../interfaces/task-ledger-v1.md).

## Task Boundaries

A task is the smallest repository-valid candidate that one fresh reviewer can
accept or reject with one verdict. It changes one primary responsibility and
produces one concrete postcondition or stable handoff.

Split when any part can be implemented, proved, repaired, or accepted
independently. Keep parts together only when splitting would leave an invalid
intermediate state, separate canonical source from required generated output,
or separate a change from the focused proof needed to accept it.

File count, diff size, desired agent count, and elapsed time are diagnostic
signals, not boundaries by themselves. When they expose multiple owners,
postconditions, repair boundaries, or proof oracles, split on those semantic
boundaries.

Each task is its own acceptance unit by default. A compound acceptance unit is
allowed only when no member can be accepted separately in a valid repository
state; record the exact inseparability reason. Prefer a later integration task
over a broad end-to-end implementation unit.

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
an agent-owned technical, review, proof, or Planning repair with available
authority is owner-held recovery even when the current unit result is `Blocked`.
Use `blocked` only when no ready unit or owner-held recovery remains because a
required user/external input or authority is unavailable. A conflicting
`status: blocked` reopens Planning; it is not a user confirmation question.
After the final accepted unit, verify the global Completion condition before
`done`.

## Stop Rule

The ledger is ready when every accepted obligation has one unit disposition,
each task packet admits one verdict, every dependency names a consumed output or
gate, and each `Accept when` can falsify its postcondition.
