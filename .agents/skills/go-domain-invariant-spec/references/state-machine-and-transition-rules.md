# State Machine And Transition Rules

## When To Load
Load this when a feature has lifecycle states, phase boundaries, workflow ownership, terminal states, invalid transitions, stuck-state behavior, or transition guards.

Use current repo product/spec artifacts as behavior truth. For this repository, the workflow lifecycle in `AGENTS.md` and `docs/spec-first-workflow.md` is the primary example domain. External sources calibrate how to make transitions explicit.

## State Model Pattern
For each lifecycle, write:

```text
Lifecycle owner:
States:
Initial state:
Terminal states:
Allowed transitions:
Forbidden transitions:
Transition trigger:
Preconditions:
Postconditions:
Idempotent repeat behavior:
Timeout or stuck-state behavior:
Violation outcome:
Traceability:
```

Keep business states separate from technical implementation statuses. A state such as `specification_blocked` can be domain-meaningful in this repo workflow; a state such as `file_written` is usually just implementation progress unless it changes allowed business behavior.

## Example Invariant Statements
- `SessionPhaseBoundary`: for non-trivial work, a session advances only the current named phase unless an upfront direct-path or lightweight-local waiver was recorded before crossing the boundary.
- `PlanningBeforeImplementation`: implementation cannot start until the implementation plan is explicit and the readiness gate is not `FAIL`.
- `PostCodeArtifactConsumption`: implementation and validation phases consume existing workflow/process artifacts and must not invent new planning artifacts mid-code.
- `SpecClarificationGate`: non-trivial `spec.md` approval requires the clarification challenge to be reconciled, explicitly deferred, accepted as risk, or routed to a blocker.
- `ReadinessStateMeaning`: `PASS`, `CONCERNS`, `FAIL`, and `WAIVED` are distinct planning outcomes; treating all of them as "good enough to code" violates workflow semantics.

## Good And Bad State Transition Specs
Good transition spec:

```text
Lifecycle: non_trivial_task
Owner: orchestrator workflow contract
State: planning
Trigger: `plan.md` and expected `tasks.md` are ready for implementation handoff
Preconditions:
- `spec.md` is approved or eligible for explicit direct-path waiver
- required `design/` artifacts are approved or design-skip rationale exists
- validation/proof path is explicit
Allowed next states:
- implementation when readiness = PASS
- implementation_with_accepted_risks when readiness = CONCERNS and risks/proof obligations are named
- upstream_reopen when readiness = FAIL
- implementation when readiness = WAIVED and scope/rationale are eligible
Forbidden next states:
- implementation when readiness = FAIL
- implementation when an unresolved high-impact open question could change correctness or ownership
Violation outcome: block implementation and route to the named earlier phase
```

Bad transition spec:

```text
When planning is mostly done, start coding.
```

Why it fails: "mostly done" does not say which gate was reached, which questions remain, or which failure path applies.

Good mini transition table:

| From | Trigger | Guard | To | Violation outcome |
| --- | --- | --- | --- | --- |
| `specification` | `spec.md` candidate complete | clarification gate reconciled or eligible waiver recorded | `technical_design` | keep `spec.md` draft or blocked |
| `technical_design` | design bundle complete | required artifacts approved or skip rationale exists | `planning` | reopen specification or design |
| `planning` | readiness check complete | `PASS` or accepted `CONCERNS` or eligible `WAIVED` | `implementation` | route to earlier phase |
| `implementation` | proof exposes missing required `tasks.md` | none | `planning_reopen` | stop coding; do not invent task ledger mid-code |

## Edge-Case Prompts
- Is this a true lifecycle or just a one-step command?
- Which states are terminal, and can any terminal state be reopened?
- What transition is legal on repeated delivery of the same command or event?
- What transition is legal when the process times out or external confirmation is ambiguous?
- Which actor owns the transition: user, system, subagent, orchestrator, or external dependency?
- Does a skipped or waived gate need a state of its own, or only a recorded transition guard?
- What forbidden transition would cause silent data, workflow, or authority drift?

## Downstream Handoff Notes
- API handoff: after transitions are stable, define which transitions are externally visible and which are internal workflow semantics.
- Data handoff: identify the source of truth for current state and, when needed, transition history.
- Reliability handoff: model timeout, retry, stuck-state, and reconciliation paths as domain states or domain outcomes before choosing operational machinery.
- QA handoff: every transition guard needs positive, negative, duplicate/replay, and invalid-transition tests when risk justifies it.
- Observability handoff: if the state is operationally meaningful, name low-cardinality labels and audit events after the domain vocabulary is stable.

## Exa Source Links
- [ddd-crew Aggregate Design Canvas](https://github.com/ddd-crew/aggregate-design-canvas) for state transitions as part of aggregate design.
- [Vendure Order Lifecycle and State Machine](https://docs.vendure.io/current/core/core-concepts/orders/) for explicit allowed next states and custom transition hooks.
- [State Machines in Microservices Workflows](https://www.nilus.be/blog/state_machines_in_microservices_workflows/) for workflow owners, guards, idempotency, and reconciliation in distributed lifecycles.
- [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf) for entity lifecycle and aggregate boundary vocabulary.
