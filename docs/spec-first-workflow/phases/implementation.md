# Implementation

Use when one accepted implementation unit is ready and authorized.

## Carrier

Use one root-local Acceptance-Unit Lead when exactly one fixed unit is in
scope, no sibling unit must be scheduled, and no durable multi-session recovery
or isolated candidate handoff is required.

Use the Ledger Orchestrator when several acceptance units may become ready,
concurrent mutable candidates must be coordinated, dependency unlocking must
continue after individual completions, execution must survive session or actor
changes, an isolated candidate handoff needs a separate routing owner, or
canonical ledger transitions require one routing owner. Durable recovery or
isolated handoff can require this carrier even for one unit.

The Orchestrator never implements unit work; the Lead does not schedule sibling
acceptance units. Scheduling, related-Lead reuse, and canonical transitions
belong to the [Planning Ledger Contract](planning/ledger-contract.md) when a
persisted ledger exists.

## Execution

The Acceptance-Unit Lead owns the fixed unit through one accepted or blocked
result. Load the current task packet and consumed outputs; repair only within
that boundary.
Reopen the smallest upstream owner only when evidence invalidates accepted
behavior, architecture, the task boundary, or its proof criterion. Additional
callers, error handling, cleanup, and internal placement needed to satisfy the
same accepted result remain implementation work. A discovered mutable owner or
exclusive lock updates the live frontier without reopening Outcome.

Fixture, transport-control, and test-runner defects remain repairs of this unit
while accepted behavior, scenario, oracle, and proof boundary hold. Mechanical
corrections to test names, paths, or runner options update the existing command
locator without reopening Test Design. Reopen Test Design only when its method
is invalid, and the mechanism owner when product design is invalid. For novel
controls, consume Test Design's [feasibility witness](test-design.md#feasibility)
and exercise the first runnable path before building the rest of the harness.

The Lead implements a coherent unit directly by default. Delegate a subset
only when it enables independent concurrent progress, supplies a missing
capability, or keeps the remaining context reliably bounded. A serial handoff
that duplicates the Lead's context needs one of those reasons. Independent lanes
need stable accepted interfaces, disjoint writable owners and locks, focused
proof, and results the Lead can integrate without a missing decision. Use the
existing brief; no separate delegation assessment is required. Dispatch useful
independent lanes before waiting and integrate their results serially.

When changed generated contracts or shared interfaces feed several lanes,
one owner first establishes the canonical source, regenerates any outputs, and
checks the smallest connected production path through the affected seams.
Use compilation, or a smoke check when compilation cannot expose the mismatch;
reuse current evidence when it already establishes that path. Stabilize these
inputs before expanding dependent implementation and test matrices. This is
intra-unit iteration, not another task or review; it does not establish behavior
acceptance.

Execution lanes are not acceptance units. [Agent
Harness](../../agent-harness.md) owns lane lifetime, nesting, isolation, capacity,
and subtree reconciliation; load it when delegating or replacing a lane.

Executors diagnose and repair code and focused-check failures within the accepted
contract and resource budget without returning each attempt to the Lead. A failed
compile or changed local candidate does not require a new execution agreement.
Return contract conflicts, inadequate oracles, unresolved material risks, or
stalled diagnosis with evidence and a proposal. The Lead applies [Parent-Owned
Recovery](../shared/transition.md#parent-owned-recovery) and resumes the same unit
after the required input closes.

Use the existing validation lock for resource scheduling. Where independent
repositories share a constrained host without a common lock, agree one bounded
execution window with the resource owner in the existing brief. Sequential
focused repairs may continue within that window; coordinate again when it
expires or the resource scope or budget changes. Acquire and release the
applicable command lock for each run, including failed runs; do not hold it
while editing or awaiting review. A work window never bypasses locks, candidate
freeze, heavy-check authority, or the agreed resource limits.

## Candidate Freeze And Proof

When the code is ready, fix the candidate and apply
[Review](../shared/review.md)'s Implementation trigger. Required review may run
alongside focused and other non-mutating checks on that candidate. Wait for a
cheap compile or smoke check first when it would avoid reviewing unusable code;
do not require all focused proof to finish before useful review starts.
Name in-flight checks and their owners in the existing brief and deliver their
results to the reviewer without duplicate execution. Respect validation locks
and resource limits; review overlap does not authorize concurrent heavy checks.
Do not mutate the candidate while its lanes are active. If repair needs an edit,
stop or join lanes using that candidate first, then rerun only invalidated proof and review.
Acceptance waits for all mandatory results on the repaired candidate.

Select proof through [Validation Routing](../../validation-routing.md) and the
[Evidence Contract](../shared/evidence-contract.md), which owns executor, Lead,
and delivery-check responsibilities. Shared Review selects whether independent
review is required; load [Implementation Review](implementation-review.md) only
for that review.

The same Lead owns candidate-caused repair. During orchestrated execution,
return [Acceptance Result V1](../interfaces/acceptance-result-v1.md) instead of
writing the ledger.

If reaching Accept-when reveals an undeclared companion dependency, reopen
Planning rather than absorbing the companion or shipping a layer. For explicitly
permitted preparation behind a pending gate, follow the [Ready
Frontier](planning/ledger-contract.md#ready-frontier) stop and resume rules.

Load other methods only for the changed surface, and [remote
preflight](../shared/deployment-proof-preflight.md) only before a matching
external action.

Done when current evidence, Lead self-review, and any required independent
review establish the unit's postcondition on the real path. Otherwise return
the exact unverified claim and owner. Reopen through
[Transition](../shared/transition.md) only when an accepted input is invalid.
Once these conditions hold, return the result. Expand proof or review only for
a concrete failure, invalidated evidence, or unresolved required claim; optional
improvements do not extend the completed unit.
