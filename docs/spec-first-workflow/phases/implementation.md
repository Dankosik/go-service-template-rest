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

For a non-trivial fixture, find an existing valid fixture or producer while
writing the test. Otherwise derive the required entities, relationships, and
state transitions from the authoritative contract before expanding scenarios.
Keep that construction in test code, not a separate preparation artifact.
Execution follows the feedback and final-validation boundaries below.

Implement directly when handoff costs more than it saves. Run independent tasks
and useful subtask lanes in parallel when accepted interfaces are stable, writable owners and exclusive
locks are disjoint, and capacity permits. Integrate mutations serially. Apply
[Agent Harness](../../agent-harness.md) when delegating or replacing a lane.

A finished implementation immediately unlocks downstream implementation that
consumes its available code or agreed contract. Task boundaries do not create
check, review, self-review, or acceptance stages. Writing tests, generating
required sources, and reading callers remain implementation work.

### Feedback During Coding

#### Static Diagnostics

After a coherent edit changes shared types, signatures, imports, generated
contracts, or their callers, use bounded compile-only or type diagnostics to
catch mechanical incompatibilities. Include affected production and test code
and required build variants; gather independent diagnostics even if one fails.
Choose commands that execute neither application/test code nor startup hooks,
services, containers, or live probes. Run within existing resource and execution
authority, with no watch loop or heavy checks; repeat only after a relevant
change or new diagnostic hypothesis. Lint, scans, runtime builds, and aggregate
validation still wait for the assembled ledger. Behavioral feedback has the
narrower trigger below.

This feedback repairs code; it is not a task-transition gate or acceptance
receipt. An unavailable diagnostic tool does not hold a handoff or independent
coding. Fix observed in-scope mechanical defects before calling that code
Implemented; record an unavailable implementation input and continue unrelated
work. Missing final-validation infrastructure does not block coding.

#### Behavioral Feedback

Use one bounded behavioral scenario during coding when a named unverified
assumption at an implemented boundary would otherwise propagate into substantial
dependent work, and the scenario costs less than that likely rework. State the
assumption, dependent work, and discriminating result in existing task state.
Task completion, a generic risk label, or a desire for a green check is not
this trigger.

Choose the smallest scenario that exercises the actual boundary with an
independent expected result. Use an available local fixture and existing
resource authority, validation locks, and a bounded runtime and cleanup.
Keep its consumed code and inputs stable while it runs; unrelated writers may
continue. This allowance does not include aggregate suites, full matrices,
watch loops, review, live targets, or provisioning paid resources.

Stop after resolving that assumption; repeat only after a relevant repair or
new discriminating hypothesis. On failure, use [Validation Repair](implementation/validation-repair.md).
If the scenario cannot run cheaply within existing authority, retain the
material uncertainty and continue code supported by accepted contracts. It does not become mandatory
final proof merely because it was considered during coding. A demonstrated
defect is repaired before returning the affected code as Implemented; a missing
probe alone creates no handoff gate. Reopen invalid accepted behavior or
architecture through its owner.

Retain the actual command, result, and exercised scope in existing execution
state for possible final reuse under the Evidence Contract. This feedback
grants no task acceptance, aggregate receipt, or release authority. Required
final validation and review still cover the whole assembled result.

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

The delivery owner selects one non-overlapping plan for the ordinary local
criterion in [AGENTS.md](../../../AGENTS.md#validation-budget) and any explicit
additions. Ordinary Go delivery uses [Go Validation](../../validation/go.md):
matching build and relevant unit tests, not an automatic `make verify`. Use
[Validation Routing](../../validation-routing.md) only for mixed surfaces or a
specialized command branch. Load the [Evidence Contract](../shared/evidence-contract.md)
when judging explicit additions, reuse, claim scope, or unavailable
infrastructure. Reconcile packet requirements with their actual source;
agent-added runtime scenarios do not become mandatory by accumulation. Combine
claims covered by the same command.
Apply [Review](../shared/review.md) only at this final boundary, never per task.
Keep the assembled candidate unchanged while checks or review consume it. Join
or stop those readers before repair, then rerun only invalidated evidence.

Within the selected validation plan, collect cheap mechanical failures before
expensive execution. Cover affected production/test projects, required build
variants, generated clients, and accepted intermediate release versions;
one failing project must not hide independent diagnostics.

For a required runtime/integration scenario or runner fixture, load
[Validation Scenarios](implementation/validation-scenarios.md) before execution.
On a check failure, load [Validation Repair](implementation/validation-repair.md).
Before isolated or remote execution, load [Execution Inputs](implementation/execution-inputs.md).
Before a potentially long check or resource wait, load [Progress](implementation/progress.md).
These branches do not add checks or change the assembled validation boundary.

Final local acceptance requires the AGENTS.md local criterion, explicit task
additions, resolved blocking findings, and any final review selected by shared
Review. Stop once that boundary passes. Missing optional runtime proof does not
block local `Accepted`; report only material limitations.
Missing required proof remains `implementation complete; verification incomplete`
for that scope. Continue available authorized repairs without indefinite
environment recovery. A requested CI, release, deployment, or runtime result
remains outstanding until actually obtained; never claim it from local tests.
