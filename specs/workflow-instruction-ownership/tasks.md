# Workflow Instruction Ownership Tasks

Ledger status: completed

## Goal Contract

Goal objective: Complete the workflow instruction ownership cleanup by executing this ledger through mirror sync, targeted duplication proof, and final validation.
Goal scope: canonical workflow docs, canonical workflow skills, generated skill mirrors, this task-local bundle, and validation evidence named here.
Completion condition: all required tasks are checked, mirror sync is current, targeted duplication checks pass, required validation passes or any external blocker is recorded, and this ledger plus `spec.md` contain closeout evidence.
Completion evidence: `rtk make skills-sync`, `rtk make skills-check`, `rtk make agents-check`, `rtk make guardrails-check`, targeted grep checks, and `rtk git diff --check`.
Blocked-stop condition: if a required command cannot run, a required validation fails for a reason outside this change, or cleanup exposes a new workflow policy decision, stop with affected tasks unchecked, record `Blocked:` with evidence, and do not claim completion.

Read before coding:

- `specs/workflow-instruction-ownership/spec.md` because it is the canonical decision record for this cleanup.
- `AGENTS.md` because it owns repository authority.
- `docs/spec-first-workflow.md` because it is the workflow router.
- `docs/spec-first-workflow/shared/artifact-model.md` and `docs/spec-first-workflow/shared/subagents-and-handoff.md` because they own shared mechanics.
- `docs/spec-first-workflow/phases/planning.md` and `docs/spec-first-workflow/phases/task-review-readiness.md` because they decide the planning/readiness split.

Read before relevant tasks:

- `.agents/skills/planning-session/SKILL.md` and `.agents/skills/planning-and-task-breakdown/SKILL.md` before T003.
- Workflow/session skills under `.agents/skills/` before T004.
- `scripts/dev/sync-skills.sh` and `Makefile` before T005 if mirror behavior is unclear.

Do not change:

- Runtime Go code, API behavior, migrations, generated service artifacts, OpenAPI, SQLC, Docker, deployment behavior, closed historical task bundles, root historical `tasks.md`, or `.codex/agents`.

Task-local implementation quality bar:

- Edit canonical sources first; regenerate mirrors instead of manually editing mirror directories.
- Keep `AGENTS.md` compact and move mechanics to phase/shared docs by reference.
- Keep phase docs as phase-local execution authority.
- Keep skills as session/method wrappers that cite owning docs for repo policy, exact authorization text, handoff rendering, and readiness approval.
- Preserve the planning/readiness boundary: planning authors draft ledgers; `task-review-readiness.md` approves implementation readiness.
- Do not remove all reminders of a hard rule when a loaded skill needs a pointer to the canonical source.

Progress log: update each task checkbox and evidence after proof runs.
Resume rule: on resume, read git status, this ledger, and `spec.md`; continue at the first unchecked task whose dependencies are satisfied.

## Implementation Handoff

Task ledger review: PASS
Implementation readiness: PASS
Consumes: approved `spec.md`, user-approved implementation plan, current workflow docs, and this task ledger.
Technical-design-review consumed: not expected with rationale in `spec.md`.
Design fan-out status: not expected with rationale in `spec.md`.
Subagent gates consumed: local_only with rationale in `spec.md`.
Ledger-review fan-out: local_only.
Ledger-review fan-out rationale: the user supplied an approved, file-specific plan; task slices are docs/skills-only, deterministic, and validated by grep plus mirror checks. The active subagent tool exists but requires explicit delegation authorization, which the implementation prompt did not provide.
Proof: required commands and targeted grep checks in T006.
Reopen target: specification if a new policy ownership decision is needed; planning if task slices or proof are insufficient.

## Tasks

Legacy cleanup audit:

| Source | Surface | Status | Evidence | Retention owner/reason/exit |
| --- | --- | --- | --- | --- |
| `spec.md` D6-D8 | Duplicate skill authority and generated mirrors | refactored | Targeted grep checks passed; `rtk make skills-check` passed | N/A |
| `spec.md` Out of scope | Closed `specs/workflow-simplification/` bundle | retained | Existing closed status; no edits planned | Historical workflow context only; no exit needed |
| `spec.md` Out of scope | Root historical `tasks.md` | retained | Existing file says historical ledger; no edits planned | Historical linter ledger only; no exit needed |

- [x] T001 Patch authority, router, and shared docs.
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/spec-first-workflow/shared/artifact-model.md`, `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
  Source: `spec.md` D1-D4.
  Depends on: none.
  Proof-first waiver: docs-only ownership cleanup; targeted diff and grep proof are more useful than a failing test.
  Proof: targeted reads confirm `AGENTS.md` stays hard authority, router stays navigation-only, artifact model owns recording/layout, and subagents shared doc owns exact authorization/handoff mechanics.
  Stop/reopen condition: reopen specification if a new authority layer or policy owner is needed.
  Evidence:
  - Command/read: targeted reads and final diff review of `AGENTS.md`, `docs/spec-first-workflow.md`, `shared/artifact-model.md`, and `shared/subagents-and-handoff.md`.
  - Result: passed.
  - Key output/ref: `subagents-and-handoff.md` now states it is the canonical source for the exact authorization line; `artifact-model.md` points hard decision rules and triggers back to `AGENTS.md`.
  - Changed proof files: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/spec-first-workflow/shared/artifact-model.md`, `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
  - Residual blocker/narrower claim: none.

- [x] T002 Patch phase docs for phase-local ownership.
  Files: `docs/spec-first-workflow/phases/planning.md`, `docs/spec-first-workflow/phases/task-review-readiness.md`, and only other phase docs whose duplicate wording blocks the ownership split.
  Source: `spec.md` D5-D7.
  Depends on: T001.
  Proof-first waiver: docs-only ownership cleanup.
  Proof: targeted reads confirm planning stops at draft/repaired `tasks.md`, while task-review/readiness remains the canonical implementation-readiness gate.
  Stop/reopen condition: reopen specification if phase ownership changes beyond the accepted model.
  Evidence:
  - Command/read: `rtk rg -n 'This file is the canonical home for task-ledger review|Task ledger review: pending_task_review|Implementation readiness: pending_task_review' docs/spec-first-workflow/phases/planning.md docs/spec-first-workflow/phases/task-review-readiness.md`.
  - Result: passed.
  - Key output/ref: `planning.md` leaves both readiness fields as `pending_task_review`; `task-review-readiness.md` declares itself the canonical task-ledger review and implementation-readiness approval owner.
  - Changed proof files: `docs/spec-first-workflow/phases/planning.md`, `docs/spec-first-workflow/phases/task-review-readiness.md`.
  - Residual blocker/narrower claim: none.

- [x] T003 Fix planning skill readiness ownership.
  Files: `.agents/skills/planning-session/SKILL.md`, `.agents/skills/planning-and-task-breakdown/SKILL.md`.
  Source: `spec.md` D6-D7.
  Depends on: T001, T002.
  Proof-first waiver: skill text cleanup; grep proof is the relevant regression check.
  Proof: targeted grep confirms planning skills do not claim they approve task-ledger review or implementation readiness and route completed ledgers to `task-review-readiness`.
  Stop/reopen condition: reopen planning if the task-review phase cannot render implementation handoff from the revised wording.
  Evidence:
  - Command/read: `rtk rg -n 'Task ledger review: PASS|Implementation readiness: PASS|Task ledger review: CONCERNS|Implementation readiness: CONCERNS|Task ledger review: FAIL|Implementation readiness: FAIL|Implementation readiness: WAIVED|Task ledger review: WAIVED' .agents/skills/planning-session .agents/skills/planning-and-task-breakdown || true`.
  - Result: passed; no matches.
  - Key output/ref: planning skills consistently say completed ledgers route to `task-review/readiness` and leave readiness as `pending_task_review`.
  - Changed proof files: `.agents/skills/planning-session/SKILL.md`, `.agents/skills/planning-and-task-breakdown/SKILL.md`, and planning-session references.
  - Residual blocker/narrower claim: none.

- [x] T004 Deduplicate workflow session skills.
  Files: `.agents/skills/workflow-planning-session/SKILL.md`, `.agents/skills/research-session/SKILL.md`, `.agents/skills/specification-session/SKILL.md`, `.agents/skills/specification-review-session/SKILL.md`, `.agents/skills/technical-design-session/SKILL.md`, `.agents/skills/validation-closeout-session/SKILL.md`, `.agents/skills/workflow-status/SKILL.md`.
  Source: `spec.md` D4-D6.
  Depends on: T001, T002.
  Proof-first waiver: skill text cleanup; grep proof is the relevant regression check.
  Proof: targeted grep confirms session skills reference `shared/subagents-and-handoff.md` instead of duplicating the full exact authorization prompt and do not become second authorities for phase gates.
  Stop/reopen condition: reopen specification if a skill truly needs to own a rule currently assigned to docs.
  Evidence:
  - Command/read: `rtk rg -n 'Subagent authorization: I explicitly request and authorize read-only subagents' .agents/skills .claude/skills .cursor/skills .gemini/skills .github/skills .opencode/skills || true`.
  - Result: passed; no skill or mirror matches.
  - Key output/ref: `docs/spec-first-workflow/shared/subagents-and-handoff.md` remains the only canonical home for the exact authorization text; session skills reference owning docs instead of restating full prompt mechanics.
  - Changed proof files: workflow/session skills under `.agents/skills/`, including `validation-closeout-session`.
  - Residual blocker/narrower claim: none.

- [x] T005 Regenerate and check mirrors.
  Files: generated skill mirrors under `.claude/skills`, `.gemini/skills`, `.github/skills`, `.cursor/skills`, and `.opencode/skills`.
  Source: `spec.md` D8.
  Depends on: T003, T004.
  Proof-first waiver: generated mirror sync.
  Proof: `rtk make skills-sync` then `rtk make skills-check`.
  Stop/reopen condition: if sync produces unexpected non-skill changes, stop and inspect before continuing.
  Evidence:
  - Command/read: `rtk make skills-sync`; `rtk make skills-check`.
  - Result: passed.
  - Key output/ref: `skills sync complete (non-destructive)` and `skills check complete (non-destructive)`.
  - Changed proof files: generated skill mirrors under `.claude/skills`, `.gemini/skills`, `.github/skills`, `.cursor/skills`, and `.opencode/skills`.
  - Residual blocker/narrower claim: none.

- [x] T006 Run final validation and close out this bundle.
  Files: `specs/workflow-instruction-ownership/spec.md`, `specs/workflow-instruction-ownership/tasks.md`.
  Source: `spec.md` Validation.
  Depends on: T001, T002, T003, T004, T005.
  Proof-first waiver: final validation task.
  Proof: targeted grep checks, `rtk make skills-check`, `rtk make agents-check`, and `rtk git diff --check`.
  Stop/reopen condition: failed validation reopens the owning task unless the failure is unrelated pre-existing drift.
  Evidence:
  - Command/read: `rtk make skills-check`; `rtk make agents-check`; `rtk make guardrails-check`; targeted grep checks for authorization and planning readiness ownership; `rtk git diff --check`.
  - Result: passed.
  - Key output/ref: `skills check complete (non-destructive)`, `agents check complete`, `required repository guardrails check passed`, authorization grep returned no skill/mirror matches, planning readiness verdict grep returned no planning-skill matches, and `rtk git diff --check` exited clean.
  - Changed proof files: `specs/workflow-instruction-ownership/spec.md`, `specs/workflow-instruction-ownership/tasks.md`.
  - Residual blocker/narrower claim: none.
