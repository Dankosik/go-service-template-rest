# Context-First Artifact Audit Fixes Tasks

## Implementation Handoff

Consumes: approved `spec.md`, `design/`, and this task ledger.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: none.
Reopen target: planning if edits require broad new artifact templates or change `AGENTS.md` authority.

## Tasks

- [x] T001 [Docs] Add master `workflow-plan.md` next-session context bundle guidance in `docs/spec-first-workflow.md`. Depends on: none. Proof: targeted diff review and `git diff --check`.
- [x] T002 [Docs] Make post-code implementation, review, and validation phase-control shapes explicit in `docs/spec-first-workflow.md`. Depends on: T001. Proof: targeted diff review and `git diff --check`.
- [x] T003 [Skills] Add optional research-backed decision provenance guidance to `.agents/skills/spec-document-designer/SKILL.md`. Depends on: T001. Proof: `make skills-check`.
- [x] T004 [Skills] Add compact `tasks.md` handoff header guidance to `.agents/skills/planning-and-task-breakdown/SKILL.md`. Depends on: T001. Proof: `make skills-check`.
- [x] T005 [Skills] Add explicit negative artifact status rationale guidance to planning and technical-design session skills and references. Depends on: T001. Proof: `make skills-check`.
- [x] T006 [Skills] Add compact review finding disposition shape to planning phase-control references. Depends on: T002. Proof: `make skills-check`.
- [x] T007 [Validation] Run `git diff --check`, `make agents-check`, and `make skills-check`; update task-local closeout state with actual results. Depends on: T001-T006. Proof: `git diff --check`, `make agents-check`, and `make skills-check` passed in this session.
