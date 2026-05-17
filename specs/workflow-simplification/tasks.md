# Workflow Simplification Tasks

## Implementation Handoff

Consumes: approved `spec.md`, `workflow-plans/planning.md` design-skip rationale, and this task ledger.
Implementation readiness: `CONCERNS`.
First task: `T001`.
Accepted concerns:
- Docs and skills must land coherently in one pass; prove consistency with targeted review plus `rtk make agents-check` and `rtk make skills-check`.
- The simplified paths must not weaken full-orchestrated escalation triggers; prove preserved invariants remain explicit in authority docs and challenge skills.
- `lean local` must not become an unstructured shortcut; prove lean docs/skills require `spec.md`, `tasks.md`, inline `Risk Challenge`, and fresh proof for bounded non-trivial implementation.
- Historical task bundles must stay valid; do not rewrite existing completed bundles.
Reopen target: specification if implementation needs a new artifact name, a different approval model, or a compatibility rule not approved in `spec.md`.

## Tasks

- [x] T001 [Phase 1] Update `AGENTS.md` so the direct / lean local / full orchestrated trigger matrix and artifact-depth rules are prominent while preserving orchestrator authority, read-only subagents, canonical `spec.md`, task-ledger-gated implementation when needed, and fresh validation evidence.
  Depends on: none.
  Proof: targeted diff read confirms direct path, lean local, and full orchestrated rules are near the top; `lightweight local` is preserved only as a compatibility alias; high-risk escalation triggers remain explicit; no invariant is weakened.

- [x] T002 [Phase 1] Rewrite `docs/spec-first-workflow.md` around trigger-based artifact depth: direct path, lean local, and full orchestrated; make `workflow-plans/<phase>.md`, split `design/`, `test-plan.md`, `rollout.md`, and review/validation phase files conditional by trigger.
  Depends on: `T001`.
  Proof: targeted diff read confirms the detailed doc matches `AGENTS.md`, keeps old full bundles valid, and moves lean local rules into primary routing instead of a late exception appendix.

- [x] T003 [Phase 1] Add the lean artifact shapes to `docs/spec-first-workflow.md`: delta-style `Behavior / Contract Delta`, compact design, inline `Risk Challenge`, and lean `tasks.md` as the main execution surface.
  Depends on: `T001`, `T002`.
  Proof: targeted diff read confirms lean local requires `spec.md`, `tasks.md`, `Risk Challenge`, and proof obligations; direct path remains the only routine no-bundle path.

- [x] T004 [Phase 1] Update `docs/subagent-contract.md` and `docs/subagent-brief-template.md` so lanes are triggered by unresolved owning questions, preserve the read-only advisory boundary, and include the shorter challenge/review brief variant without making subagents default ceremony.
  Depends on: `T001`, `T002`, `T003`.
  Proof: targeted diff read confirms evidence and handoff classifications remain, and the short variant does not relax read-only or evidence requirements.

- [x] T005 [Phase 2] Update core workflow/session skills to prefer the trigger matrix and simplified artifact model: `.agents/skills/workflow-planning-session/SKILL.md`, `.agents/skills/research-session/SKILL.md`, `.agents/skills/specification-session/SKILL.md`, `.agents/skills/technical-design-session/SKILL.md`, `.agents/skills/planning-session/SKILL.md`, and `.agents/skills/validation-closeout-session/SKILL.md`.
  Depends on: `T001`, `T002`, `T003`.
  Proof: targeted skill diff read confirms direct/lean/full routing is consistent with the docs and each skill still stops at its phase boundary.

- [x] T006 [Phase 2] Update workflow helper, challenge, design, and planning skills to support inline `Risk Challenge`, local checklist waivers, delta-style lean specs, and merged lean design context: `.agents/skills/workflow-status/SKILL.md`, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md`, `.agents/skills/spec-clarification-challenge/SKILL.md`, `.agents/skills/spec-document-designer/SKILL.md`, `.agents/skills/go-design-spec/SKILL.md`, and `.agents/skills/planning-and-task-breakdown/SKILL.md`.
  Depends on: `T001`, `T002`, `T003`, `T004`.
  Proof: targeted skill diff read confirms formal challenge remains required for triggered high-risk/full-orchestrated work, and lean local `Risk Challenge`/merged-design paths require explicit rationale rather than silent omission.

- [x] T007 [Phase 3] Run a consistency sweep over workflow docs and skills for stale mandatory-artifact language, duplicate status ownership, and references that still make full orchestrated files look default for bounded work.
  Depends on: `T001`, `T002`, `T003`, `T004`, `T005`, `T006`.
  Proof: targeted `rtk rg` sweeps plus manual diff read show old mandatory wording is either removed, trigger-scoped, or intentionally preserved for full orchestrated work.

- [x] T008 [Phase 3] Validate and close the implementation: run `rtk make agents-check`, `rtk make skills-check`, and `rtk git diff --check`; then update this task ledger, `workflow-plan.md`, and `spec.md` validation/outcome only with fresh evidence.
  Depends on: `T007`.
  Proof: the three commands pass, task checkboxes match completed work, and closeout artifacts name the exact evidence.
