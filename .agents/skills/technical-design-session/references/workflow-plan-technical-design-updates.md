# Workflow Plan Technical Design Updates

## Behavior Change Thesis
When loaded for workflow-control updates after a design pass, this file makes the model record master and phase-local status, artifact readiness, blockers, and reopen routing instead of leaving state in chat, duplicating design content in workflow files, or letting the two workflow files disagree.

## When To Load
Load after writing or repairing design artifacts, or whenever technical design is blocked and workflow control must record the reopen target.

## Decision Rubric
- Update `workflow-plan.md` with current design checkpoint, phase status, design artifact statuses, contract-design checkpoint status when plausible, conditional artifact statuses, blockers, reopen conditions, `Session boundary reached`, `Ready for next session`, and `Next session starts with`.
- Update the active design phase-control file, either `workflow-plans/system-integration-design.md` or `workflow-plans/go-code-ownership-design.md`, with pass type, local status, completion marker, artifact statuses, local stop rule, blockers, parallelizable follow-up if any, and next-checkpoint or technical-design-review handoff state.
- Negative artifact statuses such as `not expected`, `conditional`, or `waived` need a short trigger rationale; a bare label is not enough for resume.
- Keep workflow files routing-only; link to design artifacts rather than copying component maps, sequence detail, or ownership tables into them.
- If master and phase-local workflow files disagree, repair or block before claiming the session is complete.
- If a triggered conditional artifact is draft, missing, or stale, mark technical design `blocked` or `in_progress`; do not call the handoff review-ready.
- In repair passes, record the repaired artifact and leave unrelated artifact statuses untouched.

## Imitate
```markdown
`workflow-plan.md`
Current phase: go-code-ownership-design
Phase status: complete
Required design artifacts: approved
Conditional artifacts: `design/data-model.md` approved; `design/contracts/` approved with OpenAPI source-of-truth handoff; `rollout.md` not expected
Negative status rationale: `rollout.md` not expected because no migration, mixed-version, deploy-order, or failback choreography is in scope.
Blockers: none
Session boundary reached: yes
Ready for next session: yes
Next session starts with: technical design review
```

Copy this shape: the master owns cross-phase routing and next-session readiness.

```markdown
`workflow-plans/go-code-ownership-design.md`
Pass type: repair
Repaired artifact: `design/sequence.md`
Still blocked: `design/contracts/` is draft
Completion marker: not met
Stop rule: do not begin technical design review or planning until contracts design is approved.
```

Copy this shape: the phase file records local repair state without pretending the whole bundle is ready.

```markdown
Technical design blocked.
Blocker: `spec.md` does not choose event durability semantics.
Reopen target: specification.
Technical design review readiness: no.
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
- Forgetting `Session boundary reached`, `Ready for next session`, or `Next session starts with` because the final message says it.
- Clearing blockers in workflow control without repairing the design artifact or routing upstream.
