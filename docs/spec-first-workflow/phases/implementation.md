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
acceptance units. Do not bind one Lead when sibling units still need scheduling.

## Execution

The Acceptance-Unit Lead owns the fixed unit through integration, claim-matched
proof, required independent review, and one accepted or blocked result. Load
the current task packet and consumed outputs; repair only within that boundary.
A new postcondition, responsibility, behavior, or proof oracle reopens Planning
instead of expanding the task. A discovered mutable owner or exclusive lock
updates the live frontier without reopening Outcome.

Fixture, transport-control, and test-runner defects remain repairs of this unit
while accepted behavior, oracle, and proof boundary hold. Reopen Test Design
only when its method is invalid, and the mechanism owner when product design is
invalid; do not turn an implementation defect into a new phase chain. For novel
controls, consume Test Design's [feasibility witness](test-design.md#feasibility)
and exercise the first runnable path before building the rest of the harness.

The Lead implements a coherent unit directly by default. Delegate a subset
only when it enables independent concurrent progress, supplies a missing
capability, or keeps the remaining context reliably bounded. A serial handoff
that duplicates the Lead's context needs one of those reasons. Use the existing
brief; no separate delegation assessment is required.

Fan out execution lanes when two or more strict subsets have disjoint writable
responsibility, no shared exclusive lock, stable accepted interfaces,
independently checkable focused proof, and a result the Lead can integrate
without delegating a missing decision. Dispatch every independent lane before
waiting. Integrate returned lane results serially under the Lead. When the Lead
cannot reliably hold the whole edit surface, fan out lanes and keep one review.

An identified lane that produces no material result at a stall signal is a
carrier failure. Replace it or finish that subset directly. Do not wait
indefinitely.

Execution lanes are not acceptance units. Workers do not accept, transition, or
review the parent unit. Lanes do not spawn; a brief that still needs partition
returns to the Lead. For a lane's technical or proof gap, the Lead applies
[Parent-Owned Recovery](../shared/transition.md#parent-owned-recovery) and
resumes the same unit after the required input closes.

## Candidate Freeze And Proof

After focused proof passes, freeze one candidate identity. Dispatch independent
review and remaining read-only or non-mutating validation concurrently. Do not
mutate the candidate while those lanes are active. Acceptance waits for every
mandatory result; the lanes do not wait for one another.

Workers run only the focused proof named by their brief. The Lead does not
rerun an equivalent package aggregate on an unchanged integrated candidate.
Surface-aware aggregate proof runs once as `make verify` on the integrated
delivery tree unless the unit's own acceptance claim still needs a leaf that
`make verify` marked not applicable.

The same Lead owns every candidate-caused repair. [Review](../shared/review.md)
selects bounded delta recheck versus a fresh reviewer. During orchestrated
execution, return [Acceptance Result
V1](../interfaces/acceptance-result-v1.md) instead of writing the ledger.

If the packet cannot reach its Accept-when without still-planned companion
work, reopen Planning rather than silently absorbing the companion or shipping
a layer.

Load only methods exposed by the changed surface. Apply [Validation
Routing](../../validation-routing.md), the [Evidence
Contract](../shared/evidence-contract.md), and [Implementation
Review](implementation-review.md). Load the [Planning Ledger
Contract](planning/ledger-contract.md) only when a persisted ledger exists, and
[remote preflight](../shared/deployment-proof-preflight.md) only before a
matching external action.

Done when current evidence and review establish the unit's postcondition on the
real path. Otherwise return the exact unverified claim and owner. Reopen through
[Transition](../shared/transition.md) only when an accepted input is invalid.
