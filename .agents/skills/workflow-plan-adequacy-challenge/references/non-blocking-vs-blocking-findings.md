# Non-Blocking Vs Blocking Findings

## When To Load
Load this when the same issue could be over-classified as a phase blocker or under-classified as a note. Use it to calibrate severity and the recommended action, not to add generic checklist findings.

## Authoritative Inputs
- `AGENTS.md`: blocking adequacy findings must be reconciled before phase-complete handoff; non-blocking findings may be recorded or explicitly accepted.
- `docs/spec-first-workflow.md`: findings must say what is insufficient, why it matters, what could fail, whether it blocks handoff or is recordable, and exactly what the orchestrator should add or clarify.

## Good Findings
- `Gap`: Master has no active phase file link after generating `workflow-plans/specification.md`.
  `Why It Matters`: Later sessions may inspect the wrong phase plan or assume no phase-local routing exists.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `add_missing_routing`.
  `Exact Orchestrator Addition`: Add `Phase workflow plans: specification active at workflow-plans/specification.md; technical-design pending; planning pending`.
- `Gap`: One fan-out lane lacks a single chosen skill, but other lanes are clear and local research can proceed while it is repaired.
  `Why It Matters`: That lane could violate the one-skill-per-pass rule or produce unfocused research.
  `Classification`: `blocks_specific_lane`.
  `Recommended Action`: `clarify_lane_ownership`.
  `Exact Orchestrator Addition`: Add `Lane C: reliability-agent; owned question: timeout and retry policy; skill: go-reliability-spec; dependency: waits for API lane only if contract semantics change`.
- `Gap`: Master says `rollout.md: not expected` but lacks a short rationale.
  `Why It Matters`: The next session might wonder whether delivery risk was forgotten.
  `Classification`: `non_blocking_but_record`.
  `Recommended Action`: `record_skip_or_accepted_risk`.
  `Exact Orchestrator Addition`: Add `rollout.md: not expected; rationale: no deploy sequencing, migration, mixed-version, or rollback choreography change`.

## Bad Findings
- "Everything should block until perfect." Bad because the challenge filters for execution-quality and handoff-safety impact.
- "This is only a nit" when missing readiness would let implementation start. Bad because understated severity hides a gate violation.
- "No findings; approved." Bad because the challenger can report no surviving adequacy gaps but cannot approve.

## Blocker Classification Examples
- `blocks_phase_handoff`: missing current phase, missing active phase file, missing stop rule, missing readiness status in planning, or artifact status that would let the next phase start incorrectly.
- `blocks_specific_lane`: a single research/review lane has missing role, owned question, one-skill choice, dependency, or generated phase-control status while the rest of the phase can continue.
- `non_blocking_but_record`: a rationale, accepted risk, optional-artifact non-trigger, or wording cleanup improves resume reliability but does not change current routing.

## Exact Orchestrator Additions
- For `blocks_phase_handoff`: add or repair the field in both master and active phase files, then leave `Ready for next session: no` until reconciled.
- For `blocks_specific_lane`: add the missing lane field and mark only that lane or generated file blocked.
- For `non_blocking_but_record`: add one short rationale line or accepted-risk note; do not expand workflow control into spec, design, plan, or task content.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Atlassian project risk management](https://www.atlassian.com/work-management/project-management/project-risk-management) for connecting risks to escalation paths and assigned next steps.
- [Atlassian RACI chart](https://www.atlassian.com/work-management/project-management/raci-chart) for role clarity and accountability without replacing the execution process.
- [Asana status report template](https://asana.com/templates/status-report) for separating project health, blockers, and next steps.
