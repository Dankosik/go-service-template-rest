# Legacy Code Cleanup Discipline Tasks

## Goal Contract

Goal objective: Complete the approved legacy code cleanup discipline amendment by executing this ledger from `T001` through final validation.

Stopping condition: all required tasks are checked, required proof passes or records a concrete blocker, `tasks.md` evidence is current, and `spec.md` `Validation` / `Outcome` reflects the implementation result.

Read first:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/build-test-and-development-commands.md`
- `specs/legacy-code-cleanup-discipline/spec.md`
- this `tasks.md`

Do not change:
- Go runtime behavior, public APIs, database schema, migrations, deployment policy, or CI job topology.
- The approved lean-local scope, unless a task hits a named reopen condition.
- Generated skill mirrors as primary sources; update `.agents/skills/*` first, then run the repository sync/check flow.
- Unrelated pre-existing worktree changes. Re-read files before editing and work with current content.
- Task-local workflow state outside `spec.md` and this `tasks.md`; no `workflow-plan.md` or separate design artifact is expected for this amendment.

Progress log: after each task, update that task's `Evidence` line with the command, read, or blocker. If blocked, stop and record `Blocked:` under the task with the smallest reopen target.

## Implementation Handoff

Workflow state: planning complete; implementation is the next phase.

Consumes: approved lean-local `spec.md` with compact design and this reviewed task ledger.

Separate technical design depth: not expected. The approved compact design names affected surfaces, ownership/source-of-truth, and sequence/failure behavior. Reopen technical design only if deterministic guardrail/tooling enforcement requires new CI contract, generated-artifact model, or broad tooling semantics.

Subagent gates consumed:
- Specification gate: local-only lean-local gate recorded in `spec.md`, result `PASS`.
- Planning task-ledger review gate: local-only rationale below, result `PASS`.

Ledger-review fan-out rationale:
- The current subagent spawn tool is restricted to explicit user requests for sub-agents, and this planning request did not ask for delegated lane work.
- The implementation-readiness risk is concentrated in one repository-policy seam: source-of-truth order, full affected-surface coverage, mirror regeneration, and proof commands for the cleanup invariant.
- The approved spec already fixes the scope, source-of-truth ownership, non-goals, reopen conditions, and forward-looking proof obligations. A local ledger review can cover traceability, ordering, and proof without inventing new design decisions.

Task ledger review: PASS.

Implementation readiness: PASS.

Accepted concerns: none.

First task: T001.

Proof obligations carried into implementation:
- Changed-file scope audit with `rtk git status --short --branch`.
- Targeted legacy-surface audit that records removed, refactored, retained, or not-applicable status for the old instruction surfaces touched by this amendment.
- Positive and negative text checks for the new invariant and for retired permissive wording where text search is reliable.
- `rtk make skills-sync` and `rtk make skills-check` after source-managed skill edits.
- `rtk bash -n scripts/ci/required-guardrails-check.sh` and `rtk make guardrails-check` if the guardrail script changes.
- `BASE_REF=origin/main HEAD_REF=HEAD rtk make docs-drift-check` when the comparison refs are available.
- `rtk git diff --check`.

Reopen target:
- Reopen planning for missing task coverage, weak proof wording, or ordering gaps that do not change approved scope.
- Reopen technical design if the guardrail/tooling work would require broad static-analysis semantics, a new CI contract, or a generated-artifact model not already approved.
- Reopen specification if implementation would make legacy removal mandatory across public compatibility windows, persisted data migrations, deployment rollouts, generated contracts, or another protected domain.

## Task-Ledger Review Record

Reviewed packet:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/build-test-and-development-commands.md`
- `.agents/skills/planning-and-task-breakdown/SKILL.md`
- `specs/legacy-code-cleanup-discipline/spec.md`
- current `git status`

Coverage result: PASS. Every in-scope surface from the approved spec is represented in tasking, proof, or preserved constraints: repository authority docs, workflow mechanics, subagent guidance docs, source-managed skills, generated skill mirrors, validation command docs, narrow guardrail support, and task-local closeout evidence.

Ordering result: PASS. The ledger updates authority and source-managed surfaces before derived mirrors, runs mirror sync/check after canonical skill edits, and leaves closeout evidence until after proof.

Proof result: PASS. The ledger names targeted text proof, mirror drift proof, guardrail/script proof, docs drift proof, whitespace proof, and changed-file scope proof. Runtime Go proof is conditional because runtime code changes are out of scope.

Open gates: none. The ledger contains no unresolved placeholder marker, unresolved decision gate, implementation-time architecture choice, or hidden cleanup follow-up.

## Tasks

- [x] T001 [Checkpoint 1] Establish the implementation baseline and affected edit set.
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/build-test-and-development-commands.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, `.agents/skills/*`, `scripts/ci/required-guardrails-check.sh`, `specs/legacy-code-cleanup-discipline/spec.md`, `specs/legacy-code-cleanup-discipline/tasks.md`.
  Depends on: none.
  Acceptance: current dirty files are understood before editing; unrelated pre-existing changes are not reverted; the implementation edit set stays within the approved surfaces unless a reopen condition is hit.
  Proof: `rtk git status --short --branch` plus targeted reads/searches of affected surfaces.
  Evidence: Completed. `rtk git status --short --branch` showed pre-existing dirty overlapping target surfaces in authority docs, `.agents/skills`, generated skill mirrors, and `scripts/ci/required-guardrails-check.sh`, plus unrelated `specs/orchestrator-soul-md/`; targeted reads/searches covered the required docs, guardrail script, and canonical skills before edits, and no runtime Go surface was selected.

- [x] T002 [Checkpoint 1] Add the repository invariant and workflow mechanics to authority docs.
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`.
  Depends on: T001.
  Acceptance: `AGENTS.md` states the hard invariant that replaced or unused legacy code must be removed, refactored into the active path, or explicitly retained with owner, reason, proof, and exit condition; `docs/spec-first-workflow.md` explains spec, planning, coding, review, validation, closeout, and reopen mechanics; subagent docs require read-only lanes to inspect and report unexplained surviving legacy surfaces when that is in scope.
  Proof: targeted `rtk rg -n` checks over these files for the cleanup invariant, retention requirements, negative proof, and reopen language.
  Evidence: Completed. Targeted `rtk rg -n "Replaced or unused legacy code|owner, reason, proof|known legacy surfaces|known in-scope legacy surfaces|targeted negative searches|smallest owning phase|unexplained surviving legacy surfaces|Legacy cleanup scope" AGENTS.md docs/spec-first-workflow.md docs/subagent-contract.md docs/subagent-brief-template.md` found the hard invariant, spec/planning/coding/review/validation/reopen mechanics, and subagent legacy-surface reporting requirements.

- [x] T003 [Checkpoint 1] Make validation-command documentation and narrow guardrail support discoverable.
  Files: `docs/build-test-and-development-commands.md`, `scripts/ci/required-guardrails-check.sh`.
  Depends on: T002.
  Acceptance: validation docs identify the task-specific legacy cleanup proof surface, including targeted searches, retained-surface proof, skill mirror checks, docs drift checks, guardrails, and whitespace checks; `modernize-check` remains informational only; the guardrail script asserts the invariant remains present in authority/source-managed surfaces without pretending to detect all unused code.
  Proof: `rtk bash -n scripts/ci/required-guardrails-check.sh`, `rtk make guardrails-check`, and targeted `rtk rg -n` checks over validation docs.
  Evidence: Completed. Targeted `rtk rg -n "modernize-check|task-specific legacy cleanup proof|Legacy cleanup replacement proof|retired identifiers|retained with owner|legacy cleanup invariant|targeted negative searches" docs/build-test-and-development-commands.md scripts/ci/required-guardrails-check.sh` found the informational `modernize-check` boundary, task-specific cleanup proof docs, and narrow invariant guardrails; `rtk bash -n scripts/ci/required-guardrails-check.sh` passed; `rtk make guardrails-check` passed.

- [x] T004 [Checkpoint 2] Update specification, design, and planning source-managed skills to carry cleanup decisions into approved artifacts and ledgers.
  Files: `.agents/skills/spec-document-designer/SKILL.md`, `.agents/skills/go-design-spec/SKILL.md`, `.agents/skills/planning-and-task-breakdown/SKILL.md`.
  Depends on: T002.
  Acceptance: spec guidance requires known legacy surfaces and remove/refactor/retain semantics; design guidance protects source-of-truth, generated-artifact, and retention consequences when separate design depth exists; planning guidance fails or reopens readiness when known in-scope legacy removal is missing from executable tasking or proof.
  Proof: targeted `rtk rg -n` checks over these source-managed skills for legacy-surface naming, retention criteria, cleanup tasking, generated/mirror source-of-truth order, and readiness/reopen language.
  Evidence: Completed. Targeted `rtk rg -n "known legacy surfaces|retained with owner, reason, proof, and exit condition|legacy-surface delta|source-of-truth and generated-artifact consequences|retained legacy surfaces|missing in-scope legacy cleanup is a planning-readiness failure|cleanup audit/removal tasking|generated and mirrored cleanup source-of-truth order" .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md` found legacy-surface naming, retention criteria, cleanup tasking, generated/mirror source-of-truth order, and readiness failure wording.

- [x] T005 [Checkpoint 2] Update coding and review source-managed skills to enforce in-scope cleanup during implementation and review.
  Files: `.agents/skills/go-coder/SKILL.md`, `.agents/skills/go-design-review/SKILL.md`, `.agents/skills/go-language-simplifier-review/SKILL.md`, `.agents/skills/go-qa-review/SKILL.md`.
  Depends on: T004.
  Acceptance: coding guidance treats cleanup made necessary by the approved task as in scope while preserving the non-goal against speculative cleanup; review guidance flags unexplained surviving replaced or unused code as a merge-risk finding when an approved artifact does not justify retention.
  Proof: targeted `rtk rg -n` checks over these skills for in-scope cleanup, non-speculative boundary, unexplained legacy findings, and review proof expectations.
  Evidence: Completed. Targeted `rtk rg -n "Cleanup required by the approved task is in scope|targeted negative checks|unexplained surviving replaced or unused legacy|stale old-path code|retired surface is gone|retained-surface proof" .agents/skills/go-coder/SKILL.md .agents/skills/go-design-review/SKILL.md .agents/skills/go-language-simplifier-review/SKILL.md .agents/skills/go-qa-review/SKILL.md` found in-scope cleanup, non-speculative boundary, unexplained legacy findings, and review proof expectations.

- [x] T006 [Checkpoint 2] Update validation and closeout source-managed skills to require fresh cleanup proof.
  Files: `.agents/skills/go-verification-before-completion/SKILL.md`, `.agents/skills/validation-closeout-session/SKILL.md`.
  Depends on: T004.
  Acceptance: verification guidance maps completion claims to targeted old-surface proof, retained-surface proof, generated/mirror drift proof, and fresh command evidence; closeout guidance updates existing `tasks.md` and `spec.md` with removal/refactor/retention evidence instead of leaving cleanup implicit in chat.
  Proof: targeted `rtk rg -n` checks over these skills for negative checks, retention proof, task/spec evidence updates, and stale-proof rejection.
  Evidence: Completed. Targeted `rtk rg -n "targeted negative proof for retired identifiers|retained-surface proof|source-of-truth proof plus drift/sync checks|removal/refactor/retention evidence|targeted old-surface negative checks|remove/refactor/retain/not-applicable evidence|stale-proof rejection" .agents/skills/go-verification-before-completion/SKILL.md .agents/skills/validation-closeout-session/SKILL.md` found negative checks, retention proof, task/spec evidence updates, generated/mirror drift proof, and stale-proof rejection.

- [x] T007 [Checkpoint 3] Regenerate and verify skill mirrors from canonical `.agents/skills`.
  Files: `.claude/skills/`, `.gemini/skills/`, `.github/skills/`, `.cursor/skills/`, `.opencode/skills/`.
  Depends on: T004, T005, T006.
  Acceptance: mirrors reflect canonical source-managed skill edits; no mirror is hand-edited as an independent authority; agent mirrors remain untouched unless agent sources are changed.
  Proof: `rtk make skills-sync`, then `rtk make skills-check`. If agent sources unexpectedly change, also run `rtk make agents-sync` and `rtk make agents-check`; otherwise record that agent mirror proof is not applicable.
  Evidence: Completed. `rtk make skills-sync` passed with "skills sync complete (non-destructive)", then `rtk make skills-check` passed with "skills check complete (non-destructive)"; generated skill mirrors in `.claude/skills/`, `.gemini/skills/`, `.github/skills/`, `.cursor/skills/`, and `.opencode/skills/` now reflect canonical `.agents/skills`. `scripts/dev/sync-agents.sh` was read and `rtk git diff --name-only -- .codex/agents .claude/agents` showed no changed agent source or agent mirror files, so `agents-sync` / `agents-check` were not applicable.

- [x] T008 [Checkpoint 4] Run and record the task-specific legacy-surface audit for this amendment.
  Files: all files changed by T002 through T007, plus `specs/legacy-code-cleanup-discipline/tasks.md` and `specs/legacy-code-cleanup-discipline/spec.md`.
  Depends on: T002, T003, T004, T005, T006, T007.
  Acceptance: each old or ambiguous instruction surface encountered during implementation is classified as removed, refactored into the active invariant, intentionally retained with reason/proof/exit condition, or not applicable; targeted searches prove no known retired permissive wording remains where text search is reliable.
  Proof: `rtk git status --short --branch` and targeted `rtk rg -n` positive/negative checks chosen from the actual changed identifiers and instruction phrases.
  Evidence: Completed. `rtk git status --short --branch` showed the expected changed authority docs, canonical `.agents/skills`, generated skill mirrors, guardrail script, and task-local bundle, plus pre-existing unrelated `SOUL.md` / `specs/orchestrator-soul-md/`. Positive audit `rtk rg -n "Replaced or unused legacy code|Cleanup required by the approved task is in scope|missing in-scope legacy cleanup is a planning-readiness failure|targeted negative proof for retired identifiers|Legacy cleanup replacement proof|source-of-truth proof plus drift/sync checks|unexplained surviving replaced or unused legacy surface" AGENTS.md docs scripts/ci/required-guardrails-check.sh .agents/skills .claude/skills .cursor/skills .gemini/skills .github/skills .opencode/skills specs/legacy-code-cleanup-discipline` found the invariant in authority docs, canonical skills, generated mirrors, validation docs, guardrail script, and this task bundle. Negative audit `rtk bash -lc 'if rg -n "Might be useful later|cleanup is optional|optionally clean up|leave cleanup for later|remember cleanup later" ...; then ...; else ...; fi'` reported "retired permissive cleanup wording absent from authority/docs/skills/mirrors". Retained surfaces: `modernize-check` remains intentionally informational in `docs/build-test-and-development-commands.md` and is explicitly not a substitute for task-specific legacy cleanup proof; `optional cleanup` / `record_only` language remains only in `go-design-spec` and mirrors as taste/local-style review classification, not replacement cleanup permission, with the new invariant and planning/coding/review proof rules as the exit condition if that wording becomes ambiguous. Agent-source proof was not applicable because `rtk git diff --name-only -- .codex/agents .claude/agents` returned no changed agent source or mirror files.

- [x] T009 [Checkpoint 5] Run repository-owned validation for the changed planning/docs/skill/guardrail surface.
  Files: repository root and changed files from this ledger.
  Depends on: T008.
  Acceptance: changed authority docs, skills, mirrors, and guardrail surfaces pass their owning checks; docs drift is checked when base/head refs are available; runtime Go checks are run only if implementation unexpectedly touches runtime Go files, otherwise record runtime proof as not applicable.
  Proof: `rtk make guardrails-check`; `rtk make skills-check`; `BASE_REF=origin/main HEAD_REF=HEAD rtk make docs-drift-check` when refs are available; `rtk git diff --check`; conditional `rtk make check` if runtime Go files change.
  Evidence: Completed. `rtk make guardrails-check` passed; `rtk make skills-check` passed; `BASE_REF=origin/main HEAD_REF=HEAD rtk make docs-drift-check` passed with "no files changed, docs drift check passed"; `rtk git diff --check` passed; runtime Go proof was not applicable because `rtk bash -lc 'changed_go=$(git diff --name-only -- "*.go" "*.mod" "*.sum" ...); ...'` reported "no runtime Go/module files changed".

- [x] T010 [Final] Close out task-local evidence and workflow state.
  Files: `specs/legacy-code-cleanup-discipline/tasks.md`, `specs/legacy-code-cleanup-discipline/spec.md`.
  Depends on: T009.
  Acceptance: every completed task has current evidence; `spec.md` `Validation` and `Outcome` summarize fresh proof without raw sensitive data; no new workflow-control artifact is created; final status names any residual blocker or says implementation is verified.
  Proof: final `rtk git status --short --branch`, final reread of `tasks.md` and `spec.md`, and `rtk git diff --check -- specs/legacy-code-cleanup-discipline/tasks.md specs/legacy-code-cleanup-discipline/spec.md`.
  Evidence: Completed. `spec.md` `Validation` and `Outcome` were updated with the implementation proof and verified result; no new workflow-control artifact was created. Final proof after this closeout edit: `rtk git status --short --branch`, final reread of `specs/legacy-code-cleanup-discipline/tasks.md` and `specs/legacy-code-cleanup-discipline/spec.md`, and `rtk git diff --check -- specs/legacy-code-cleanup-discipline/tasks.md specs/legacy-code-cleanup-discipline/spec.md`.
