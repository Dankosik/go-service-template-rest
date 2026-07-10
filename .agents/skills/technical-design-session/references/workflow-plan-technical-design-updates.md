# Workflow Plan Technical Design Updates

## Behavior Change Thesis
When loaded for workflow-control updates after a design pass, this file makes the model record master and phase-local status, artifact readiness, blockers, and reopen routing instead of leaving state in chat, duplicating design content in workflow files, or letting the two workflow files disagree.

## When To Load
Load after writing or repairing design artifacts, or whenever technical design is blocked and workflow control must record the reopen target.

## Decision Rubric
- Update `workflow-plan.md` with current design checkpoint, typed phase/artifact/gate state, contract-design checkpoint state when applicable, blockers, reopen conditions, `session_boundary`, `handoff_readiness`, and `Next session starts with`.
- Update the active design phase-control file, either `workflow-plans/system-integration-design.md` or `workflow-plans/go-code-ownership-design.md`, only when `ROUTING-PHASE-CONTROL` requires it; then record pass type, canonical `phase_state`, completion marker, typed artifact state, local stop rule, blockers, parallelizable follow-up if any, and next-checkpoint or technical-design-review handoff state. Otherwise keep compact state in the master or design entrypoint.
- `artifact_expectation=not_expected|conditional` and any separate `waiver_disposition=waived` need a short trigger or eligibility rationale; a bare label is not enough for resume.
- Keep workflow files routing-only; link to design artifacts rather than copying component maps, sequence detail, or ownership tables into them.
- If master and phase-local workflow files disagree, repair or block before claiming the session is complete.
- If a triggered conditional artifact becomes expected but remains draft, absent, or stale, set technical-design `phase_state=blocked|active`; do not call the handoff review-ready.
- In repair passes, record the repaired artifact and leave unrelated artifact statuses untouched.

## Imitate
```markdown
`workflow-plan.md`
Current phase: go-code-ownership-design
phase_state: complete
Required design artifacts: artifact_expectation=expected, artifact_state=approved, record_validity=current
Conditional artifacts: `design/data-model.md` and `design/contracts/` resolved to artifact_expectation=expected, artifact_state=approved, record_validity=current; `rollout.md` resolved to artifact_expectation=not_expected, artifact_state=absent
Negative expectation rationale: `rollout.md` is not expected because no migration, mixed-version, deploy-order, or failback choreography is in scope.
Blockers: none
session_boundary: reached
handoff_readiness: ready
Next session starts with: technical design review
```

Copy this shape: the master owns cross-phase routing and next-session readiness.

```markdown
`workflow-plans/go-code-ownership-design.md`
Pass type: repair
Repaired artifact: `design/sequence.md`, artifact_state=approved, record_validity=current
Still blocked: `design/contracts/` has artifact_expectation=expected, artifact_state=draft
Completion marker: not met
Stop rule: do not begin technical design review or planning until contracts design is approved.
```

Copy this shape: the phase file records local repair state without pretending the whole bundle is ready.

```markdown
Technical design blocked.
Blocker: `spec.md` does not choose event durability semantics.
Reopen target: specification.
phase_state: blocked
session_boundary: reached
Technical-design-review procedural_gate_state: blocked
Technical-design-review review_verdict: pending
Technical-design-review record_validity: current
handoff_readiness: ready
Next session starts with: specification.
```

Copy this shape: a blocked handoff names the missing upstream decision.

## Reject
```markdown
Updated the design files; workflow state is obvious from the diff.
```

Failure: resume state is left in chat and inference.

```markdown
`workflow-plan.md`: next session starts with planning.
`workflow-plans/go-code-ownership-design.md`: `design/go-code-ownership.md` still pending.
```

Failure: master and phase-local control disagree.

```markdown
`workflow-plans/system-integration-design.md`: component map details...
```

Failure: phase control becomes a second design artifact.

## Agent Traps
- Marking required design approved while hiding triggered conditional artifacts under a generic `design/: approved` line.
- Recording `rollout.md: not expected` in one workflow file but omitting it from the other.
- Forgetting `session_boundary`, `handoff_readiness`, or `Next session starts with` because the final message says it.
- Clearing blockers in workflow control without repairing the design artifact or routing upstream.
