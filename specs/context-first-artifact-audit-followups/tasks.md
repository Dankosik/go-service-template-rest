# Context-First Artifact Audit Followups Tasks

## Implementation Handoff

Consumes: approved `spec.md`, `design/`, and this task ledger.
Implementation readiness: PASS, with lightweight-local waiver recorded in `spec.md`.
First task: T001.
Accepted concerns: none beyond the recorded adequacy-challenge waiver.
Reopen target: planning if the edits require a new artifact type, broad universal template, or authority-model change.

## Tasks

- [x] T001 [Docs] Update `docs/spec-first-workflow.md` to make `Next Session Context Bundle` always present, centralize uncertainty labels, require status/rationale in planning-bound `design/overview.md`, clarify status vocabulary, allow readable multi-line `tasks.md` items, and note that historical bundles are examples rather than templates. Depends on: none. Proof: targeted diff review and `git diff --check`.
- [x] T002 [Skills] Update cross-phase session guidance in `.agents/skills/workflow-planning-session/SKILL.md`, `.agents/skills/research-session/SKILL.md`, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md`, and `.agents/skills/workflow-status/SKILL.md` for always-present context bundles and phase-versus-routing state. Depends on: T001. Proof: `make skills-check`.
- [x] T003 [Skills] Update specification guidance in `.agents/skills/spec-document-designer/SKILL.md` and `.agents/skills/specification-session/SKILL.md` to mirror uncertainty-label and context-bundle rules. Depends on: T001. Proof: `make skills-check`.
- [x] T004 [Skills] Update technical-design guidance in `.agents/skills/go-design-spec/SKILL.md`, `.agents/skills/go-design-spec/references/design-bundle-assembly.md`, `.agents/skills/go-design-spec/references/design-readiness-and-planning-handoff.md`, `.agents/skills/technical-design-session/SKILL.md`, and `.agents/skills/technical-design-session/references/required-design-artifact-examples.md` for planning-bound overview index status/rationale. Depends on: T001. Proof: `make skills-check`.
- [x] T005 [Skills] Update planning guidance in `.agents/skills/planning-and-task-breakdown/SKILL.md`, `.agents/skills/planning-session/SKILL.md`, and `.agents/skills/planning-session/references/workflow-plan-update-examples.md` for multi-line task items, always-present context bundles, and status vocabulary. Depends on: T001. Proof: `make skills-check`.
- [x] T006 [Skills] Update validation closeout guidance in `.agents/skills/validation-closeout-session/SKILL.md` and `.agents/skills/validation-closeout-session/references/workflow-plan-completion-vs-reopen.md` for always-present context bundles and phase-vs-task/routing state wording. Depends on: T001. Proof: `make skills-check`.
- [x] T007 [Validation] Run `git diff --check`, `make agents-check`, and `make skills-check`; run sync if a repository check reports stale generated or mirrored skill surfaces. Depends on: T001-T006. Proof: `git diff --check` passed before and after closeout; `make agents-check` passed; initial `make skills-check` reported stale mirrors; `make skills-sync` completed; rerun and final `make skills-check` passed; trailing-whitespace check for new task-local files passed.
