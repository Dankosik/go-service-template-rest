# Evidence Contract

Canonical sufficiency and evidence semantics for local completion, verification,
and external readiness claims.

## Local Completion

For ordinary Go development, local completion requires the agreed behavior and
tests to be implemented, the relevant deliverable to build, relevant unit tests
to pass, and no known concrete defect in the changed outcome. Resolve any final
review selected by [Review](review.md). These are sufficient local acceptance
criteria, not a guarantee that every production behavior has been observed.
Choose existing commands through [Validation Routing](../../validation-routing.md).
Instruction-only work instead uses static consistency review and the matching
existing structural check; it does not require building or running the service.

Unit tests must meaningfully check changed behavior in the existing style and
actually execute or have valid reusable results. Written tests alone, a selector
matching none of the required tests, skipped required cases, and execution
errors are not passes. Build affected deliverable entry points and required
variants, not every template profile. Reuse adequate existing coverage; do not
invent an exhaustive scenario matrix or re-prove unchanged behavior without a
relevant dependency change or concrete defect.

Once this criterion and any explicit additions pass, report local completion
and stop. Disclose unrun optional integration, container, provider, end-to-end,
race, or performance scenarios only when their limitations are material. They
are not blockers to invent or repair; do not claim they passed. Local completion is neither a
CI result, production observation, deployment, release, nor authority for an
external action. When the request includes such an outcome, retain it as a
separate outstanding obligation until actually satisfied.

## Required And Optional Proof

A product behavior requirement, a possible test method, a local acceptance
criterion, and a CI/release gate are different things. Additional mandatory
verification must trace to an explicit requirement of the accepted task or an
existing applicable external gate. That gate applies to its own action; it does
not require a duplicate local run. A file path, generic risk label, skill,
reviewer preference, selected command, or agent-authored packet does not expand
local acceptance. Loading a domain method supplies expertise, not new gates.

Preserve genuine user-established requirements, including explicitly requested
migration execution, runtime observation, green CI, or release. Reconcile old
agent-added checks with this policy in existing task state; do not treat their
presence or prior artifact acceptance as evidence of user intent. An empty
additional-check list means none, not work to discover more. No new approval
round is needed to omit agent-invented extras.

Do not create temporary applications, smoke runners, compose stacks, remote
environments, provider sandboxes, permanent workflows, or validation matrices
solely to establish local completion. Small unit fixtures, existing test doubles,
and in-process `httptest` remain normal test authoring. Use existing suitable
tools for a concrete diagnostic question; do not promote the diagnostic into a
new completion requirement. New verification infrastructure needs an explicitly
accepted task requirement, not a broad security, integrity, or concurrency label.

If an optional check lacks Docker, a provider, access, or usable setup, record
only the material unverified scope and stop that path. Do not repair or provision
its environment, increase privileges, or enable heavy modes to remove the gap.
If a required build, unit test, or explicitly required verification cannot run,
report the exact missing input and incomplete scope truthfully. Continue useful
authorized work, but do not turn environment recovery into an unbounded project.

A reproducible or mechanically supported in-scope defect must be repaired,
including one found by an optional check. A missing optional observation,
speculative risk, style preference, or imaginable extra scenario is not such a
defect. Unrelated historical defects remain observations unless the accepted
outcome spans them. Never hide a real defect behind passing unit tests.

## Claim Scope

For a specific runtime or verification claim, evidence must be current, match
its scope, and reject the claimed wrong behavior or missing wiring. A unit-test
result establishes its tested boundary, not a live provider or production path.
Status, file presence, proposed commands, and implementation summaries alone do
not establish observed behavior. Report only what was exercised; accepting
local development does not require widening that empirical claim.

Before adopting a hard constraint, trace its source, scope, and units. Distinguish
requester policy, observed runtime settings, diagnostic limits, and agent
assumptions. Prior artifact acceptance does not establish that attribution.
Preserve actual safety and authority requirements.

## Design Proof

Specification closes behavior and its nearest feasible falsifier; Technical
Design supports the mechanism, invariants, and feasible proof. The executor
chooses concrete cases, fixtures, assertions, and commands while writing code.
Neither a test-case matrix nor completed product tests are pre-Implementation
inputs. Unwritten tests and unavailable optional infrastructure do not block
Planning or implementation from accepted behavior and design. A proposed live
falsifier is not automatically a mandatory live acceptance scenario.
Evidence needed to resolve an actual mechanism choice remains with Technical
Design; do not claim measured existing behavior from design rationale alone.

## Execution Evidence

Attach commit or tree identity only across a checkout or integration boundary;
the current bounded diff is enough for local work. Retain actual commands,
inputs, environment, result, and exercised scope, using [Evidence Result
V1](../interfaces/evidence-result-v1.md) for final verification claims.
Implementation handoffs remain `Implemented` without a proof receipt.
[Implementation](../phases/implementation.md#feedback-during-coding) owns coding
feedback and the single assembled final-validation boundary.

Reuse passing scoped evidence when covered code, relevant dependencies, command,
inputs, and environment are unchanged. A new commit, unrelated documentation
edit, task handoff, or phase transition alone does not invalidate it. Check the
relevant delta and retain original candidate and scope; never relabel an old
result as a new run. When relevant equivalence is uncertain, rerun that affected
check, not unrelated proof.

The delivery owner selects one non-overlapping plan for the local criterion and
explicit additions. One result may support several claims. Neither `make plan`,
`make verify`, nor `ALLOW_FULL=1 make check` is an automatic completion gate.
When expanded verification is explicitly required, select its matching canonical
leaves or aggregate without duplicating covered checks. Heavy execution still
needs its existing authorization and guards.

After a failed or interrupted run, retain its plan and completed scoped results.
Reconcile against the repair, running failed, invalidated, and not-yet-run
required checks only. Do not repeat the same failed attempt without changed
code, environment, or a discriminating diagnostic hypothesis. A focused repair
does not restart the whole plan, review, or task ledger.

Valid consolidated scoped evidence can establish local Completion without an
aggregate receipt. An explicitly required exact-candidate aggregate, CI check,
fresh runtime observation, or release gate still needs its actual result.
Whole-candidate receipts retain the tool's exact base, candidate, plan, input,
and environment rules under Validation Routing; do not synthesize them from
partial results. The Orchestrator records acceptance without repeating proof.

Missing required final proof means `implementation complete; verification incomplete`
for that scope. Missing optional proof does not prevent local `Accepted`.
When an external outcome remains requested, report local completion separately
without declaring the whole request complete. Verification reports evidence
and material gaps; repair and genuinely required recovery stay with their owners.
