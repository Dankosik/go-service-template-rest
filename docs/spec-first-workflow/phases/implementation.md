# Implementation

Use when accepted implementation work is ready and authorized. Implement the
planned tasks and their tests; validate the assembled result at the end.

## Carrier

Use one root-local Acceptance-Unit Lead for a single fixed unit. Use the Ledger
Orchestrator when several units, isolated handoffs, or durable scheduling need
one routing owner. The Orchestrator schedules; Leads implement their assigned
units. The [Planning Ledger Contract](planning/ledger-contract.md) owns
implementation progress, dependency unlocking, and final acceptance.

## Execution

Load the current task packet and consumed outputs. Choose and write the task's
tests alongside production code from accepted behavior and existing test
patterns. Reuse sufficient tests; add cases for changed behavior that existing
tests would miss. Select fixtures, assertions, and the smallest proving layer
locally, without a preliminary matrix, approval, or separate test-planning task.
For a non-obvious test technique, consult the matching method within this task.
Reopen only genuinely unresolved product behavior or architecture; a missing or
incorrect test case stays with the executor.

Implement directly when
handoff costs more than it saves. Run independent tasks and useful subtask lanes
in parallel when accepted interfaces are stable, writable owners and exclusive
locks are disjoint, and capacity permits. Integrate mutations serially. Apply
[Agent Harness](../../agent-harness.md) when delegating or replacing a lane.

A finished implementation immediately unlocks downstream implementation that
consumes its available code or agreed contract. Do not insert test, lint, build,
smoke, review, self-review, or acceptance stages between tasks or subtask lanes.
Do not start watch-mode checks for every edit. Writing tests, regenerating
required sources, and reading callers remain implementation work; their
existence does not establish passing behavior.

Do not run checks during ledger implementation, including optional diagnostics,
quick compiles, or checks hidden inside another task or delegated brief. Read
source and tool errors to repair implementation. Source generation is code
production; invoke its generation target without adding drift checks or tests.
If an unavailable input prevents coding, record that implementation blocker
and continue independent work. Do not turn the blocker into an early validation
stage. Missing final-validation infrastructure does not block coding.

Establish shared contracts and generated sources before dependent lanes consume
them. When implementation reveals invalid accepted behavior or architecture,
resolve that smallest decision owner. Repair test cases and oracles locally
when the expected product behavior remains unchanged. Callers, error handling,
cleanup, command locators, fixture relationships, and runner repairs for the
same accepted result stay with the executor. A new write surface updates locks
and scheduling; it does not itself reopen Planning.

Join or stop lanes using the candidate, return `Implemented` through
[Acceptance Result V1](../interfaces/acceptance-result-v1.md), and release the
implementation scope. Do not wait for proof, a reviewer, or other independent
units before returning it. `Implemented` supplies unverified implementation for
local dependent work; it is neither accepted behavior nor release authority.
The same owner remains available for defects found in final validation.

## Final Validation

Start only when every planned code task is Implemented and assembled, with no
remaining implementation blocker or writer. Finishing one task, one wave, or
all currently runnable tasks does not satisfy this condition. Do not recast a
ledger task as a standalone delivery to start verification early.

The delivery owner selects one
non-overlapping proof plan through [Validation Routing](../../validation-routing.md)
and the [Evidence Contract](../shared/evidence-contract.md). Run required
packet checks here, combining claims covered by the same command. Use one
surface-aware `make verify`; add only required proof that it does not cover.
Apply [Review](../shared/review.md) only at this final boundary, never per task.
Keep the assembled candidate unchanged while checks or review consume it. Join
or stop those readers before repair, then rerun only invalidated evidence.

Before an expensive environment run, compile the relevant test surface and
check cheap prerequisites and fixture relationships against the contract.
Reuse sufficient evidence and the prepared environment while inputs remain
valid. Do not restart PostgreSQL, Vespa, or equivalent services for a code or
fixture edit that needs no reset. Classify a failure as product, test/oracle,
or environment before choosing the repair. Rerun only invalidated proof.

Use existing validation locks; do not hold them while editing or waiting.
Within established scope and heavy-run authority, lock availability schedules
retries without a fresh CPU permit or time-window negotiation. Coordinate anew
only when authority, resource scope, or budget changes. Never run heavy checks
concurrently or bypass effect authority.

Final acceptance requires all mandatory claims, resolved blocking findings, and
any final review selected by shared Review. Missing proof means `implementation
complete; verification incomplete`. Continue available repairs and independent
work; do not claim delivery, integration, or production readiness prematurely.

## Progress

In the existing status, name implemented subresults, the next concrete result,
and any current delay: implementation, product repair, test repair, environment,
resource wait, usage limit, or external dependency. Distinguish implementation
from verified behavior. No extra report or ledger level is required.

Use the requested 8–10 hour delivery target to expose likely overruns and
change a stalled approach early. It is a planning target, not measured speed or
permission to omit accepted scope or final proof. Report measured waiting
intervals only when existing logs establish their start and end.
