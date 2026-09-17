# Planning Ledger Contract

Use only after Planning selects a persisted [Task Ledger
V1](../../interfaces/task-ledger-v1.md).

## Task And Acceptance Boundaries

A ledger task retains one coherent repository outcome and its final acceptance
criteria. Keep layers of that outcome together; use subtask lanes for useful
parallel implementation. Separate independently consumable outcomes or targets
with different external gates. Do not create more artifacts merely to show
partial progress: name implemented subresults in the current task status.

Implementation completion and verified acceptance are separate events.
`Implemented` means the planned code, tests, and cleanup are present and its
writers have joined. It does not mean any check ran or passed. Task boundaries
do not create validation or review gates. Global Completion owns the final
assembled result under the ordinary local criterion in
[AGENTS.md](../../../../AGENTS.md#validation-budget). Load the
[Evidence Contract](../../shared/evidence-contract.md) for explicit additions,
reuse, claim scope, or unavailable infrastructure. It consolidates that
criterion and explicit task additions, not every runtime scenario an agent
could propose.

## Ready Frontier

Place each dependency at the first action that actually consumes it:
implementation, final acceptance, or a named external effect. [Task Packet
V1](../../interfaces/task-packet-v1.md) owns annotations. An agreed contract can
support independent consumer implementation while a provider runtime gates only
an explicitly required live integration or external effect. Missing live infrastructure does not hold code
that can be implemented from closed decisions.

A task is ready to implement when its required decisions and consumed code or
contracts are available, writable owners and exclusive locks are free, and its
implementation authority is present. A landed `Implemented` output satisfies a
code dependency; no passing receipt or per-task acceptance is required. A
genuine acceptance requirement gates acceptance, not the start of coding;
reconcile annotations with AGENTS.md and, when needed, the Evidence Contract
before final validation. Local unverified code never satisfies a
production or external-effect gate.

Dispatch all ready independent work before waiting, within capacity. Parallel
tasks and subtask lanes require disjoint writers and exclusive locks. Keep
overlapping mutation serial. Do not wait for a whole wave or for review and
validation of a finished task before starting newly ready work.

When only a later acceptance or external gate is pending, continue the local
work supported by accepted contracts. Return `Implemented` after that work is
complete and keep the later gate in the ledger; do not keep a Lead idle waiting
for a provider. Return `Blocked` only when the remaining implementation needs
an unavailable input or authority. Resume from that input without restarting
unaffected work.

Reuse the same Lead for a related ready task after its previous code is
integrated and its lanes have stopped. Do not wait for verification or
acceptance. Re-read the new packet and current inputs; reuse context only while
it remains reliable. Reassign later repairs serially so a Lead has one active
writable scope. The Orchestrator owns scheduling and Execution locators.

After each result or integration, release finished scopes, recompute the
frontier, and start newly ready work immediately. Prefer work that supplies
missing code or contracts to other tasks. Preparation must not occupy locks or
capacity needed by its prerequisite. Capacity is a ceiling, not a fan-out target.
Reserve enough capacity to integrate active results and resolve blockers.

A discovered write overlap stops only conflicting writers. An invalidated
contract stops only work that consumes it. Continue unrelated tasks; do not
restart the ledger or reopen Planning for a mechanical locator or lock update.

## Acceptance Transition

During orchestrated execution, only the Orchestrator writes canonical ledger
state. Leads return [Acceptance Result V1](../../interfaces/acceptance-result-v1.md).
Verify result and candidate identity when receiving an isolated handoff; this
is integration bookkeeping, not a code-review or test gate.

For `Implemented`, integrate serially into the local development candidate,
record implementation completion, release scopes, and refill the frontier.
Implementation progress does not authorize pushing, deploying, or landing code into
an auto-deploy branch. Preserve the external-effect boundary.

Do not silently resolve semantic conflicts during integration. Return the
conflicting delta to its Lead for repair against the assembled tree; unrelated
work continues. Invalidate affected evidence without starting a per-task test
or review cycle. For `Blocked`, retain partial code and the exact missing
implementation input, reconcile writers, and route available recovery.

Keep one replaceable result per task. Git owns prior candidates and repair
history. Checkboxes track implementation completion; they do not claim passing
behavior. Keep `status: ready` while implementation, final validation, or an
owner-held repair can proceed. Use `blocked` only when no such authorized work
can obtain a genuinely required input, evidence, capability, or authority.
An optional validation environment is not such a dependency. Apply
[Parent-Owned Recovery](../../shared/transition.md#parent-owned-recovery).

Only after every planned code task is Implemented and assembled, with no
unfinished code or active writer, assign the existing delivery owner one final
validation boundary covering global Completion and all task claims. A blocked
task or an exhausted ready frontier does not permit partial final validation.
Do not split the ledger or dispatch per-task verification assignments to bypass
this boundary. Pending deployment or release actions do not delay the start of local
final validation; execute them only after their required evidence and authority
exist, and retain their proof as a Completion gate when those actions are
actually included in the request. Optional environments do not create a gate.
That owner runs the consolidated proof through the [Evidence
Contract](../../shared/evidence-contract.md) and any final review selected by
[Review](../../shared/review.md), and returns a Completion result. Missing or
failed required proof leaves its scope incomplete even when every task is
checked. Passing local criteria permit local `Accepted` without optional
runtime observations; explicitly requested external outcomes remain outstanding.
Return defects to their implementation owners and rerun only invalidated proof;
do not restart task-by-task acceptance. The Orchestrator records the final
verdict without repeating validation or review.

Mark `done` only after final `Accepted` establishes Completion. Then apply
[Cleanup](../../shared/cleanup.md). A root-local Lead may validate a standalone delivery after implementing it;
all tasks inside a ledger share one final validation stage.

## Stop Rule

Planning is ready when each outcome has its scope, consumed inputs, writable
owners, final observable outcomes, and dependency timing. Concrete tests are
executor-owned implementation choices, not readiness inputs. Name a future live
prerequisite as a final gate only when the accepted task explicitly requires
that execution or external effect. Empty additional-proof lists need no
expansion. Preserve actual user-owned acceptance and external-effect requirements.
