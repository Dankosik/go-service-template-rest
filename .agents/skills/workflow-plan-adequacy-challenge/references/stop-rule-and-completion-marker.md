# Stop Rule And Completion Marker

## When To Load
Load this when a phase-local plan has a vague completion marker, no stop rule, unclear phase boundary, or a next action that starts the next phase without an upfront direct/local waiver.

## Authoritative Inputs
- `AGENTS.md`: for non-trivial work, one session owns one named phase unless an upfront waiver was recorded; when the completion marker is met, update artifacts, mark boundary state, record next-session start, and stop.
- `docs/spec-first-workflow.md`: `workflow-plans/<phase>.md` owns the phase-local completion marker, stop rule, next action, and local blockers.

## Good Findings
- `Gap`: `workflow-plans/technical-design.md` says "finish design and then plan implementation" without a stop rule.
  `Why It Matters`: The session can drift from design into planning even though non-trivial phases are session-bounded.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_stop_or_completion_rule`.
  `Exact Orchestrator Addition`: Add `Completion marker: required core design artifacts approved and master artifact status updated`; add `Stop rule: stop after technical-design handoff; do not create plan.md or tasks.md in this session`.
- `Gap`: Completion marker says "research done" but fan-in and challenge status are absent.
  `Why It Matters`: Research may end before the orchestrator compares claims or records whether challenge is required.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_stop_or_completion_rule`.
  `Exact Orchestrator Addition`: Add `Completion marker: lanes returned or local research complete; orchestrator synthesized comparable claims; pre-spec challenge run or explicitly waived with rationale`.
- `Gap`: Stop rule exists, but next action says "start coding" from planning while implementation readiness is missing.
  `Why It Matters`: The phase boundary points at implementation without the readiness gate.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_readiness_status`.
  `Exact Orchestrator Addition`: Add `Stop rule: do not hand off to implementation until implementation readiness is PASS, eligible CONCERNS, or eligible WAIVED`.

## Bad Findings
- "Add every future implementation step to the stop rule." Bad because implementation steps belong in `plan.md` or `tasks.md`, not phase-control routing.
- "Approve the next phase once the stop rule is written." Bad because this skill does not approve handoff.
- "Make completion marker 'all docs done'." Bad because it is not task-specific or verifiable.

## Blocker Classification Examples
- `blocks_phase_handoff`: no stop rule for a non-trivial phase, or completion marker would allow the next phase to start in the same session.
- `blocks_specific_lane`: one lane lacks its own return condition, but phase-wide stop rule is otherwise clear.
- `non_blocking_but_record`: stop rule is adequate, but next action could be sharper for resume reliability.

## Exact Orchestrator Additions
- `workflow-plans/<phase>.md`: `Completion marker: <specific artifact or synthesis state that ends this phase>; Stop rule: <what this session must not start>; Next action: <one next routing action>; Blockers: <none or named blockers>; Session boundary handling: update master and stop when marker is met`.
- `workflow-plan.md`: `Session boundary reached: yes|no; Ready for next session: yes|no; Next session starts with: <phase plus first action>`.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Asana project closure template](https://asana.com/templates/project-closure) for completing activities, recording acceptance, archiving records, and documenting next steps.
- [Asana progress report template](https://asana.com/templates/progress-report) for connecting current status, risks, and next action items.
- [Scrum.org Definition of Ready discussion](https://www.scrum.org/resources/blog/ready-or-not-demystifying-definition-ready-scrum) for treating readiness criteria as contextual aids rather than a generic blocking checklist.
