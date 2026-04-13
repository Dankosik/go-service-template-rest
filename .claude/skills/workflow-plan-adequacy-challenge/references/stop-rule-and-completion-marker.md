# Stop Rule And Completion Marker

## Behavior Change Thesis
When loaded for symptom weak completion or stop rules, this file makes the model protect the phase boundary with concrete handoff routing instead of likely mistake letting the session continue into the next phase.

## When To Load
Load this when a phase-local plan has a vague completion marker, no stop rule, unclear phase boundary, or a next action that starts the next phase without an upfront direct/local waiver.

## Decision Rubric
- Completion marker says what proves this phase is done; stop rule says what this session must not start.
- Block handoff when a non-trivial phase can flow directly into the next phase without an eligible waiver.
- Block handoff when the completion marker skips required fan-in, challenge reconciliation, artifact approval, or readiness status for the current phase.
- Record without blocking when the stop rule is adequate but the next action could be sharper for resume reliability.

## Imitate
### Technical design drifts into planning
`Gap`: `workflow-plans/technical-design.md` says "finish design and then plan implementation" without a stop rule.

Why to copy: the session can drift from design into planning even though non-trivial phases are session-bounded.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_stop_or_completion_rule`
- `Exact Orchestrator Addition`: Add `Completion marker: required core design artifacts approved and master artifact status updated`; add `Stop rule: stop after technical-design handoff; do not create tasks.md or optional plan.md in this session`.

### Research completion omits fan-in
`Gap`: Completion marker says "research done" but fan-in and challenge status are absent.

Why to copy: research may end before the orchestrator compares claims or records whether challenge is required.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_stop_or_completion_rule`
- `Exact Orchestrator Addition`: Add `Completion marker: lanes returned or local research complete; orchestrator synthesized comparable claims; pre-spec challenge run or explicitly waived with rationale`.

### Planning points to implementation too early
`Gap`: Stop rule exists, but next action says "start coding" from planning while implementation readiness is missing.

Why to copy: the phase boundary points at implementation without the readiness gate.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_readiness_status`
- `Exact Orchestrator Addition`: Add `Stop rule: do not hand off to implementation until implementation readiness is PASS, eligible CONCERNS, or eligible WAIVED`.

## Reject
- "Add every future implementation step to the stop rule." Implementation steps belong in `tasks.md` or optional `plan.md`.
- "Approve the next phase once the stop rule is written." This skill does not approve handoff.
- "Make completion marker 'all docs done'." It is not task-specific or verifiable.

## Agent Traps
- Do not confuse "next action" with permission to begin the next phase in the same session.
- Do not let a completion marker hide missing challenge/fan-in/readiness work.
- Do not ask for a long phase doctrine when one concrete stop rule would repair the control gap.
