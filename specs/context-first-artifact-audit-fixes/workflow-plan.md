# Context-First Artifact Audit Fixes Workflow Plan

## Routing

Execution shape: lightweight local.

Rationale: the user approved fixing all findings from the completed read-only artifact-format audit. The change is docs/skills only, scoped to the workflow artifact format and handoff guidance already inspected in the audit. Same-session collapse is allowed for this pass because the findings are concrete, no runtime Go behavior changes are involved, and the user asked to proceed step by step with fixes.

Current phase: validation.
Phase status: complete.
Session boundary reached: yes.
Ready for next session: no.
Next session starts with: N/A.

## Next Session Context Bundle

- `spec.md`: approved scope, findings, and non-goals for this docs/skills pass.
- `design/overview.md`: design entrypoint and artifact index.
- `design/component-map.md`: exact docs and skill surfaces expected to change.
- `design/sequence.md`: implementation order for the docs/skills edits.
- `design/ownership-map.md`: authority split that edits must preserve.
- `tasks.md`: executable task ledger and proof expectations.

## Artifact Status

- `workflow-plan.md`: approved for this lightweight local pass.
- `workflow-plans/planning.md`: approved; records the same-session planning waiver and implementation readiness.
- `spec.md`: approved.
- `design/`: approved.
- `tasks.md`: approved.
- `plan.md`: not expected; the task is one docs/skills checkpoint and does not need supplemental strategy.
- `test-plan.md`: not expected; validation obligations fit in `tasks.md`.
- `rollout.md`: not expected; no deployment, migration, or mixed-version rollout behavior changes.
- `research/*.md`: not expected; the audit findings are preserved in `spec.md` and this bundle, and no separate reusable research note is needed.
- Post-code phase-control files: not expected; this one-session docs/skills pass can use `tasks.md` plus this master workflow plan.

## Gates And Risks

Workflow plan adequacy challenge: waived for this lightweight local pass. Rationale: the task is docs/skills only, the audit findings are already concrete, and this environment only spawns subagents on explicit user request. Proof obligation: keep changes limited to the approved artifact-format guidance and run docs/skills validation checks before closeout.

Implementation readiness: PASS.

Accepted risks: none beyond the waiver above.
Blockers: none.
Reopen targets: reopen planning if the edits require new workflow artifacts or change the artifact authority model beyond `AGENTS.md` and `docs/spec-first-workflow.md`.

## Closeout

Validation status: complete.

Fresh proof:

- `make skills-sync` completed to update skill mirrors.
- `git diff --check` passed before closeout notes.
- `make agents-check` passed.
- `make skills-check` passed.

Remaining blockers: none.
