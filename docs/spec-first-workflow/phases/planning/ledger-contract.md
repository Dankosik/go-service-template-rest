# Planning Ledger Contract

Use only after Planning selects a persisted [Task Ledger
V1](../../interfaces/task-ledger-v1.md).

## Task And Acceptance Boundaries

A ledger task is the smallest independently acceptable repository outcome, not
the smallest independently implementable piece of work.

A result is independently acceptable only when a later owner can consume it, or
the repository can ship it, without any still-planned companion work for that
same outcome. A green local test on a partial layer does not make the layer
independently acceptable.

Create a separate ledger task when the result can land in a valid repository
state without that companion work, is independently shippable or consumable,
provides a stable output to another owner or later task, or has a materially
distinct acceptance oracle or protected-risk boundary.

Keep layers of one Outcome in one unit. Split distinct Outcomes even when they
share a request, package, or convenient worktree. When those pressures conflict,
Outcome, oracle, and protected-risk boundary win.

Do not split a task because parts can be implemented, investigated, tested, or
repaired independently. Those parts are execution lanes under one
Acceptance-Unit Lead when they share one postcondition and one integrated
acceptance verdict.

One ledger task is one acceptance unit. File count, diff size, desired agent
count, and elapsed time are diagnostic signals, not boundaries.

## Ready Frontier

A unit is ready when every declared dependency output or gate is accepted, no
active unit overlaps its [Task Packet](../../interfaces/task-packet-v1.md)
mutable owners or exclusive locks, no active unit may change an accepted
interface or assumption it consumes, required authority, environment, and
focused proof are available, and it can reach one acceptance verdict without
another unfinished unit.

Dispatch the complete ready frontier before waiting, within current capacity.
A unit may be ready and still wait: start it concurrently only when owners and
locks are free and isolation cost is smaller than the unit's work; otherwise
start it serially on the current checkout when that checkout is free.
Overlapping owners or locks stay serial.

After every result, integration, blocker, or accepted transition, re-read
canonical ledger state, release completed locks, recompute the frontier, and
immediately dispatch newly ready units. Continue waiting only when no additional
unit can start. Do not wait for an earlier frontier to drain.

Capacity is a ceiling, not a fan-out target. Do not persist waves.

## Acceptance Transition

During orchestrated execution, only the Orchestrator writes canonical ledger
state. Each Lead returns one immutable [Acceptance Result
V1](../../interfaces/acceptance-result-v1.md). The Orchestrator validates unit
and candidate identity, lands the candidate serially without semantic edits,
checks that landing did not materially change the reviewed bytes or
preconditions, records the Lead-owned verdict without re-adjudicating it, then
releases locks and refills the frontier. When several results are waiting, land
one, refill, then land the next. Do not drain the landing mailbox before
dispatching newly unlocked work.

If the reviewed bytes and accepted inputs remain equivalent after landing, reuse
the review and focused evidence. Conflict resolution, rebasing, generated
output, dependency movement, or manual editing that changes candidate semantics
invalidates that review.

When no Orchestrator exists, a root-local Lead may write its own fixed unit
result.

Replace that result in place. Git owns superseded candidates, prior review
receipts, and repair history. A delegated result, review return, candidate
handoff, or attempted action is not ledger state.

Keep `status: ready` while another unit or owner-held recovery is executable;
an agent-owned technical, review, proof, or Planning repair with available
authority is owner-held recovery even when the current unit result is `Blocked`.
Use `blocked` only when no ready unit or owner-held recovery remains because a
required user/external input or authority is unavailable. A conflicting
`status: blocked` reopens Planning; it is not a user confirmation question.
After the final accepted unit, verify the global Completion condition before
`done`. When the integrated candidate contains two or more accepted units,
apply required [Review](../../shared/review.md) for that candidate before
`done`.

## Stop Rule

The ledger is ready when every accepted obligation has one unit disposition,
each task packet admits one independently acceptable verdict, every dependency
names a consumed output or gate, each packet names mutable owners and exclusive
locks, and each `Accept when` can falsify its postcondition.
