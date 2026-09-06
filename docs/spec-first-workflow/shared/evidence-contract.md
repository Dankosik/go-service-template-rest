# Evidence Contract

Canonical proof semantics for correctness, readiness, and completion claims.

Evidence of existing or completed behavior qualifies only when it is current,
matches the claim's scope, and would fail if the claimed behavior or required
production wiring were absent or wrong. Prefer one exercise of the observable
path. Split proof qualifies only
when its owner and wiring checks together establish that path. Status, file or
symbol presence, unrelated checks, and implementation summaries do not qualify
alone.

Before adopting a hard constraint, trace its source, scope, and units. Distinguish
requester policy, observed runtime settings, diagnostic limits, and agent
assumptions. Prior artifact acceptance does not establish that attribution.
Correct a misattributed constraint through its decision owner before designing
or buying proof around it; preserve actual safety and authority requirements.

## Design Proof

Interpret domain proof requirements against the claim and active phase.
Specification closes observable behavior and its nearest feasible falsifier;
Technical Design supports the selected mechanism, invariant enforcement, and
proof feasibility. The implementing agent chooses concrete test cases,
fixtures, assertions, and commands as it writes the task's code. Neither a
test-case matrix nor completed product tests are pre-Implementation inputs.

Claims about existing behavior, including deliberately unchanged behavior,
still require current evidence. Proposed commands and design rationale do not
establish runtime behavior or completed acceptance. Evidence needed to choose
an otherwise unresolved system mechanism remains with Technical Design.
Unwritten test cases and unavailable test infrastructure do not block Planning
or implementation from accepted behavior and design. The executor supplies the
tests; Implementation owns permitted coding feedback and final harness execution.

## Execution Evidence

Attach commit or tree identity only across a checkout or integration
boundary; the current bounded diff is enough for local work. Distinguish scoped
evidence from a whole-candidate validation receipt.

Reuse a passing scoped result when its covered code, relevant dependencies,
command, inputs, and environment are unchanged for the claim. A new commit
or tree identity alone does not invalidate it. Check the relevant delta and
retain the original candidate and scope; do not relabel it as a run on the new
tree. When equivalence is uncertain, rerun the affected check.

Whole-candidate aggregate receipts retain the validation tool's exact base,
candidate, plan, input, and environment requirements under [Validation
Routing](../../validation-routing.md). Scoped reuse does not create an aggregate
receipt or prove a fresh runtime observation. Do not rerun unchanged evidence
as ceremony.

At final validation, choose one minimal proof plan across all claims; one
receipt may support several. Avoid duplicating a leaf in the initial aggregate
plan or adding an aggregate whose extra surfaces are irrelevant. Start with one surface-aware
`make verify` and only uncovered required proof. A full-repository gate is never
automatic. `make plan` explains the route; it is not a gate.

Reuse existing proof when it would fail on the changed observable. Add a test or
fixture only for otherwise-unproved changed behavior; unrelated historical
coverage remains outside the unit.

During ledger implementation, [Implementation](../phases/implementation.md#feedback-during-coding)
owns static diagnostics and bounded behavioral feedback. Retain an actual
scenario result with its command, inputs, environment, and exercised scope;
evaluate scoped reuse at final validation under this contract. It creates no
per-task receipt or acceptance stage. Workers and Leads return `Implemented`,
not verified delivery. All required claims and final review remain with the
whole assembled ledger; an early scenario never substitutes for an explicitly
required final aggregate or fresh runtime observation.

The integrated delivery owner owns all required proof after assembly, including
each packet's Final validation claims. Deduplicate overlapping claims. After
an interrupted or failed aggregate, preserve its selected plan and completed
step results. Reconcile the plan against the repaired candidate, adding newly
required claims and commands. Reuse only scoped results whose dependencies,
inputs, and environment remain valid; run failed, invalidated, and not-yet-run
checks directly through their canonical targets. Complete only when the whole
reconciled plan and uncovered required claims have passing evidence.

Consolidated scoped evidence can establish local Completion without another
whole `make verify` run. It is not a passing aggregate receipt. An explicitly
required exact-candidate aggregate, CI check, runtime observation, or release
gate still must run; do not synthesize its receipt from partial results. The
Orchestrator records progress and final acceptance without repeating
proof. For a single-unit delivery, the Lead is also the delivery owner.
Repair executors run focused checks within that owner's final-validation scope
under [Implementation](../phases/implementation.md#repair-within-final-validation).
A reviewer runs only a missing or adversarial falsifier for its final independent
question. Run `ALLOW_FULL=1 make check` only when the claim spans its full
boundary. Heavy targets require `ALLOW_HEAVY=1` or CI.

Return [Evidence Result V1](../interfaces/evidence-result-v1.md) for final
validation claims; implementation handoffs carry no proof result. When required
final proof cannot run, report `implementation complete; verification incomplete`;
do not accept Completion or claim outcome completion or readiness.

Verification reports evidence and gaps. Repair, unknown-cause diagnosis, and
missing rollout policy return to their owning methods; rerun only evidence
invalidated by the resulting change.
