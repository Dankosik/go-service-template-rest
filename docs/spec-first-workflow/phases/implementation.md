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

The Orchestrator never implements unit work. The Lead does not schedule sibling
acceptance units. Do not replace the Orchestrator with a Lead while sibling
units still need scheduling. The Orchestrator may reassign a finished Lead
under the [Planning Ledger Contract](planning/ledger-contract.md#ready-frontier).

## Execution

The Acceptance-Unit Lead owns the fixed unit through integration, claim-matched
proof, required independent review, and one accepted or blocked result. Load
the current task packet and consumed outputs; repair only within that boundary.
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
that duplicates the Lead's context needs one of those reasons. Use the existing
brief; no separate delegation assessment is required.

When changed generated contracts or shared interfaces feed several lanes,
one owner first establishes the canonical source, regenerates any outputs, and
checks the smallest connected production path through the affected seams.
Use compilation, or a smoke check when compilation cannot expose the mismatch;
reuse current evidence when it already establishes that path. Stabilize these
inputs before expanding dependent implementation and test matrices. This is
intra-unit iteration, not another task or review; it does not establish behavior
acceptance.

Delegate parallel execution lanes when their expected independent progress
outweighs dispatch and integration cost. Eligible subsets have disjoint writable
responsibility, no shared exclusive lock, stable accepted interfaces,
independently checkable focused proof, and a result the Lead can integrate
without delegating a missing decision. Dispatch selected independent lanes
before waiting. Integrate returned lane results serially under the Lead. When
the Lead cannot reliably hold the whole edit surface, fan out lanes while
keeping acceptance and any required review at the unit boundary.

Use [Agent Harness](../../agent-harness.md#delegation-interface)'s stall criteria
and recovery route before replacing a lane or absorbing its work.

Executors diagnose and repair code and focused-check failures within the
accepted contract, ownership, proof boundary, and agreed resource budget without
returning each attempt to the Lead. A failed compile or changed local candidate
does not require a new execution agreement. Return a contract conflict,
inadequate oracle, unresolved material risk, or stalled diagnosis with evidence
and a proposal. The Lead resolves the gap and retains acceptance responsibility.

Use the existing validation lock for resource scheduling. Where independent
repositories share a constrained host without a common lock, agree one bounded
execution window with the resource owner in the existing brief. Sequential
focused repairs may continue within that window; coordinate again when it
expires or the resource scope or budget changes. Acquire and release the
applicable command lock for each run, including failed runs; do not hold it
while editing or awaiting review. A work window never bypasses locks, candidate
freeze, heavy-check authority, or the agreed resource limits.

Execution lanes are not acceptance units. Workers do not accept, transition, or
review the parent unit. Apply shared [Nested
Execution](../../agent-harness.md#nested-execution) through the selected adapter;
when native limits prevent descendants, return the subset to the nearest
capable parent. For a lane's technical or proof gap, the Lead applies
[Parent-Owned Recovery](../shared/transition.md#parent-owned-recovery) and
resumes the same unit after the required input closes.

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

Executors run the task's focused proof. The Lead assesses its sufficiency,
reuses current evidence, and runs any required unit Integrated check before
acceptance. The Orchestrator records that verdict without repeating the checks
or review. [Evidence Contract](../shared/evidence-contract.md) assigns the final
delivery aggregate; unit acceptance alone does not trigger it.

The same Lead owns every candidate-caused repair. [Review](../shared/review.md)
selects bounded delta recheck versus a fresh reviewer. During orchestrated
execution, return [Acceptance Result
V1](../interfaces/acceptance-result-v1.md) instead of writing the ledger.

If reaching Accept-when reveals an undeclared companion dependency, reopen
Planning rather than absorbing the companion or shipping a layer. For explicitly
permitted preparation behind a pending gate, follow the [Ready
Frontier](planning/ledger-contract.md#ready-frontier) stop and resume rules.

Load only methods exposed by the changed surface. Apply [Validation
Routing](../../validation-routing.md), the [Evidence
Contract](../shared/evidence-contract.md), and [Implementation
Review](implementation-review.md). Load the [Planning Ledger
Contract](planning/ledger-contract.md) only when a persisted ledger exists, and
[remote preflight](../shared/deployment-proof-preflight.md) only before a
matching external action.

Done when current evidence, Lead self-review, and any required independent
review establish the unit's postcondition on the real path. Otherwise return
the exact unverified claim and owner. Reopen through
[Transition](../shared/transition.md) only when an accepted input is invalid.
