# Implementation

Use when one accepted implementation unit is ready and authorized.

## Carrier

Use one root-local Acceptance-Unit Lead when exactly one fixed unit is in
scope, no sibling unit must be scheduled, and no durable multi-session recovery
or isolated candidate handoff is required.

Use the Ledger Orchestrator when several acceptance units may become ready,
concurrent mutable candidates must be coordinated, dependency unlocking must
continue after individual completions, execution must survive session or actor
changes, or canonical ledger transitions require one routing owner.

The Orchestrator never implements unit work. The Lead does not schedule sibling
acceptance units. Do not bind the Orchestrator for a single unit. Do not bind
one Lead when sibling units still need scheduling.

## Execution

The Acceptance-Unit Lead owns the fixed unit through integration, claim-matched
proof, required independent review, and one accepted or blocked result. Load
the current task packet and consumed outputs; repair only within that boundary.
A new postcondition, responsibility, behavior, or proof oracle reopens Planning
instead of expanding the task. A discovered mutable owner or exclusive lock
updates the live frontier without reopening Outcome.

Choose the smallest reliable execution topology. Implement directly when the
unit has one coherent mutable owner, delegation would duplicate context, or
handoff and integration would cost as much as doing the subset directly.

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
review the parent unit.

## Candidate Freeze And Proof

After focused proof passes, freeze one candidate identity. Dispatch independent
review and remaining read-only or non-mutating validation concurrently. Do not
mutate the candidate while those lanes are active. Acceptance waits for every
mandatory result; the lanes do not wait for one another.

Workers run only the focused proof named by their brief. The Lead reruns
claim-matched proof on the integrated candidate. Repository-wide aggregate proof
runs once on the integrated delivery tree unless the unit's own acceptance claim
requires that aggregate.

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
