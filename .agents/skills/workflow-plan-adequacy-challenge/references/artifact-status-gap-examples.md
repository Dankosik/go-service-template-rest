# Artifact Status Gap Examples

## Behavior Change Thesis
When loaded for symptom artifact expectations or statuses are unclear, this file makes the model request status and rationale repair instead of likely mistake demanding just-in-case artifacts or copying artifact content into workflow control.

## When To Load
Load this when artifact expectations or statuses are missing, stale, too vague, inconsistent across control files, or not proportional to the task's execution shape.

## Decision Rubric
- Block handoff when a required artifact is missing, falsely marked approved, or omitted from readiness-sensitive routing.
- Record without blocking when an optional artifact is correctly absent but the non-trigger rationale would prevent later guesswork.
- Require `artifact_expectation`, `artifact_state`, `record_validity`, and separate `waiver_disposition`; compound display phrases are not new-write authority.
- Workflow control owns artifact status and routing only. The artifact itself owns decisions, design, strategy, task ledgers, and proof details.

## Imitate
### Expected task ledger has no status
`Gap`: `tasks.md` is expected in the master, but status is absent and planning handoff says implementation may start.

Why to copy: non-trivial implementation could start without the executable task ledger this repo requires by default.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_artifact_status`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, add `tasks.md: artifact_expectation=expected, artifact_state=absent, record_validity=current`; record `phase_state=blocked`, `session_boundary=reached`, `handoff_readiness=ready`, and `Next session starts with: planning` for the actionable repair. Implementation remains unauthorized until planning and task review complete.

### Optional rollout file looks required
`Gap`: `rollout.md` is listed as missing, but no trigger explains why rollout detail is expected.

Why to copy: the finding pushes proportionality, not a paperwork reflex.

Use:
- `Classification`: `non_blocking_but_record`
- `Recommended Action`: `clarify_artifact_status`
- `Exact Orchestrator Addition`: Add `rollout.md: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none; rationale: no migration, mixed-version, delivery sequencing, or rollback choreography change`.

### Master and phase disagree on design approval
`Gap`: `design/` is marked approved in the master, but `workflow-plans/go-code-ownership-design.md` still says `design/ownership-map.md` is draft.

Why to copy: planning could start while a required design artifact is still incomplete.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_artifact_status`
- `Exact Orchestrator Addition`: Align both files to `design/: artifact_expectation=expected, artifact_state=draft, record_validity=current; blocker: design/ownership-map.md is not approved`; keep the design content in `design/`, not in workflow control.

### Separate design lacks review procedure/verdict state
`Gap`: `design/overview.md` or split `design/` is approved, but neither the master nor phase-local control records technical-design-review `procedural_gate_state` and `review_verdict` before planning.

Why to copy: planning could start from an unreviewed design bundle, which violates the mandatory pre-planning gate.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_artifact_status`
- `Exact Orchestrator Addition`: Add `technical-design-review: procedural_gate_state=blocked, review_verdict=pending, record_validity=current; blocker: run the mandatory gate before planning`; if already reviewed, record `procedural_gate_state=complete`, `review_verdict=PASS|CONCERNS|FAIL`, current validity, and accepted design risks/proof obligations or reopen target.

## Reject
- "Add the whole `tasks.md` checklist to `workflow-plan.md`." The master tracks status and routing, not executable task state.
- "Create `test-plan.md` just to be complete." Conditional artifacts need real triggers.
- "The plan is good once all statuses are green." The challenger does not approve the plan.

## Agent Traps
- Do not equate "missing" with "must create"; first ask whether the artifact is expected for this execution shape.
- Do not mark a missing `tasks.md` as non-blocking for non-trivial implementation unless an eligible waiver exists.
- Do not make the artifact-status repair larger than one or two routing/status lines.
