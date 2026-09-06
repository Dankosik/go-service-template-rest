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

Implement directly when
handoff costs more than it saves. Run independent tasks and useful subtask lanes
in parallel when accepted interfaces are stable, writable owners and exclusive
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
new discriminating hypothesis, using the repair method below. If the scenario
cannot run cheaply within existing authority, retain the uncertainty for final
validation and continue code supported by accepted contracts. A demonstrated
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

The delivery owner selects one
non-overlapping proof plan through [Validation Routing](../../validation-routing.md)
and the [Evidence Contract](../shared/evidence-contract.md). Run required
packet checks here, combining claims covered by the same command. Start with one
surface-aware `make verify`; add only required proof that it does not cover.
Apply [Review](../shared/review.md) only at this final boundary, never per task.
Keep the assembled candidate unchanged while checks or review consume it. Join
or stop those readers before repair, then rerun only invalidated evidence.

Before an expensive environment run, compile the relevant test surface and
collect cheap failures across the connected production/test projects, required
tags, generated clients, and accepted intermediate release versions. Do not
let one failing project hide independent diagnostics. Resolve those failures
before expensive scenarios; retain sufficient current results under the Evidence
Contract rather than rerunning them as a separate gate.

For a new or changed runner or fixture, execute its smallest complete scenario
through setup, the intended behavior, observation, and cleanup before expanding
the run. Reuse a still-valid result from coding when the Evidence Contract
permits it. Bind each scenario to its own inputs.
Preserve safe root-cause diagnostics and give cleanup a bounded
lifetime independent of a failed or cancelled scenario. Reuse the prepared
environment while its state and inputs remain valid; reset only what a failed
run invalidated.

### Repair Within Final Validation

Classify failure as product, test/oracle, or environment. Assign the causal
repair and its smallest discriminating rerun to the same executor, with the
current candidate, diagnostics, runner, writable scope, and existing resource
authority. The executor may run that focused check after repair; an old
implementation-only brief does not force a code-only handoff here. Keep final
acceptance with the delivery owner. A read-only reviewer never becomes the
repair executor.

When the cause is still uncertain, select the next run for the explanations it
can distinguish. If failure occurred before the intended behavior, check the
nearest broken setup precondition first. After repairing it, return to the
original scenario; successful setup does not establish product behavior.

Repair the defect class at its shared source: compare a broken fixture with
the complete required shape, trace retained callers before deleting a helper,
and retained references before deleting durable data. Gather related failures
in that affected scope instead of repairing only the first reported line.
If another attempt yields the same failure without new discriminating evidence,
apply [Parent-Owned Recovery](../shared/transition.md#parent-owned-recovery)
before rerunning; do not repeat the whole pipeline or increase timeouts blindly.

After an aggregate failure, continue its pending plan under the Evidence
Contract's scoped-reuse rules. A focused repair does not automatically schedule
the whole aggregate again; retain every not-yet-run or newly affected claim.

For isolated or remote execution, use the existing runner and one current
candidate record in task state or its manifest. Resolve source/patch or image
identities, stages, checksums, and host-specific paths there. Verify the actual
execution inputs before consuming a result; do not reconstruct ad hoc transfer
commands or treat a workstation path as a remote path. After repair, update
the replacement record and invalidate only affected evidence. This bookkeeping
needs no new scheduler or registry.

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

Before a potentially long check or resource wait, choose a bounded observation
checkpoint and the result or progress signal expected by then. Use the prior
comparable duration, tool timeout, and current environment when available;
without a baseline, begin with an early checkpoint and adjust from observation.
Keep the command/native locator and checkpoint in existing execution state.
This is an executor-owned monitoring choice, not a new user deadline or permit.

Use native yielding or background execution so the owner can inspect the run
at that checkpoint. Inspect actual stage, logs, process/resource state, or lock
owner; a live process or repeated elapsed-time message alone is not progress.
Continue useful work with a new observation checkpoint when evidence supports
it. If the expected result is absent, diagnose the wait or stalled stage before
another long wait or rerun. Preserve safe cleanup and required proof; exceeding
a checkpoint neither grants acceptance nor makes cancelling an effect safe.

In the existing status, name implemented subresults, the next concrete result,
and any current delay: implementation, product repair, test repair, environment,
resource wait, usage limit, or external dependency. Distinguish implementation
from verified behavior. No extra report or ledger level is required.

Use an explicitly accepted task deadline or execution budget to expose likely
overruns and reconsider a stalled approach. Keep its value in the existing
task-local artifact; do not infer one when none was accepted. A target is not
measured speed or permission to omit accepted scope or final proof. Report
measured waiting intervals only when existing logs establish their start and end.
