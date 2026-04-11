# Non-Blocking Vs Blocking Findings

## Behavior Change Thesis
When loaded for symptom finding severity is ambiguous, this file makes the model classify by execution and handoff impact instead of likely mistake making every imperfection block the phase or every missing gate a nit.

## When To Load
Load this when the same issue could be over-classified as a phase blocker or under-classified as a note. Use it to calibrate severity and recommended action, not to add generic checklist findings.

## Decision Rubric
- `blocks_phase_handoff`: the current phase cannot honestly be marked complete or ready for next session until repaired, waived, or accepted as risk.
- `blocks_specific_lane`: one lane, generated phase-control file, blocker, or artifact route needs repair, while the whole phase can continue around it.
- `non_blocking_but_record`: a short rationale or accepted-risk note improves resume reliability but does not change current routing.
- Classification should follow what could fail, not how many fields are missing.

## Imitate
### Missing active phase link
`Gap`: Master has no active phase file link after generating `workflow-plans/specification.md`.

Why to copy: later sessions may inspect the wrong phase plan or assume no phase-local routing exists.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: Add `Phase workflow plans: specification active at workflow-plans/specification.md; technical-design pending; planning pending`.

### One unclear lane
`Gap`: One fan-out lane lacks a single chosen skill, but other lanes are clear and local research can proceed while it is repaired.

Why to copy: the issue affects one lane rather than the whole phase.

Use:
- `Classification`: `blocks_specific_lane`
- `Recommended Action`: `clarify_lane_ownership`
- `Exact Orchestrator Addition`: Add `Lane C: reliability-agent; owned question: timeout and retry policy; skill: go-reliability-spec; dependency: waits for API lane only if contract semantics change`.

### Optional artifact rationale
`Gap`: Master says `rollout.md: not expected` but lacks a short rationale.

Why to copy: the note improves resume reliability without changing current routing.

Use:
- `Classification`: `non_blocking_but_record`
- `Recommended Action`: `record_skip_or_accepted_risk`
- `Exact Orchestrator Addition`: Add `rollout.md: not expected; rationale: no deploy sequencing, migration, mixed-version, or rollback choreography change`.

## Reject
- "Everything should block until perfect." The challenge filters for execution-quality and handoff-safety impact.
- "This is only a nit" when missing readiness would let implementation start. That hides a gate violation.
- "No findings; approved." The challenger can report no surviving adequacy gaps but cannot approve.

## Agent Traps
- Do not use `blocks_phase_handoff` for a lane-local repair that does not affect phase completion.
- Do not downgrade missing readiness, missing active phase, or missing stop rule just because the intended route is obvious from chat.
- Do not add non-blocking notes that cannot name a resume or routing improvement.
