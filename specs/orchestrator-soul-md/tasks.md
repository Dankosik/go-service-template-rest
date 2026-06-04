# SOUL.md Orchestrator Personality Layer Tasks

## Goal Contract

Goal objective: Add the approved repository-level `SOUL.md` orchestrator personality layer and wire its lower-precedence boundary into repository guardrails.
Stopping condition: all required tasks are checked, `rtk make guardrails-check`, `rtk make agents-check`, and `rtk git diff --check` pass or record a concrete blocker, docs-drift handling is recorded, and ledger-owned closeout evidence is current.
Read first: this `tasks.md`, `spec.md`, `workflow-plan.md`, `workflow-plans/planning.md`, `research/soul-md-agent-personality-practices.md`, `AGENTS.md`, `docs/spec-first-workflow.md`, `Makefile`, `scripts/ci/required-guardrails-check.sh`, and `docs/build-test-and-development-commands.md`.
Do not change: approved root-only SOUL scope; AGENTS.md authority over workflow, gates, commands, paths, subagent protocol, artifacts, and validation; no host-specific SOUL mirrors; no runtime Go code, API, generated contract, data, deployment, or CI job topology changes unless a listed proof unexpectedly exposes a required reopen.
Progress log: after each checkpoint, update the task checkbox and `Evidence` line with exact proof or blocker. If blocked, stop and record `Blocked:` with the missing input, failing command, or reopen target.
Blocked-stop rule: if implementation needs host-specific SOUL distribution, a changed precedence model, a new workflow decision, or a validation rule not approved in `spec.md`, stop and reopen `specification` or `workflow planning` as named in `spec.md`; do not decide it inside implementation.

## Implementation Handoff

Consumes: approved `spec.md`, compact design decisions in `spec.md`, completed research/specification workflow plans, this task ledger, and planning-phase workflow state.
Task-ledger review: PASS.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: none.
Proof obligations:
- `SOUL.md` must stay identity/personality guidance plus the explicit precedence boundary, not a second workflow manual.
- `AGENTS.md` must include both a prose lower-precedence bridge and `@SOUL.md` without weakening `@RTK.md` or any existing invariant.
- `scripts/ci/required-guardrails-check.sh` must require root `SOUL.md` and check the AGENTS/SOUL boundary in both files.
- Because `scripts/ci/` changes are docs-drift-relevant, update `docs/build-test-and-development-commands.md` or a more precise docs surface if implementation evidence shows it is a better fit.
- Run `rtk make docs-drift-check BASE_REF=<base> HEAD_REF=<head>` when valid refs exist; otherwise record why valid refs were unavailable and keep the docs update in the change set.
Reopen target: `specification` if implementation would change the approved SOUL purpose, root-only scope, lower-precedence boundary, or no-separate-design decision; `workflow planning` if scope expands to host-specific SOUL distribution or mirroring.

Subagent gates consumed:
- Research gate: local-only, complete, recorded in `workflow-plan.md` and `workflow-plans/research.md`.
- Formal clarification gate: scoped-down local challenge, PASS, recorded in `workflow-plan.md`, `workflow-plans/specification.md`, and `spec.md`.
- Technical design review: not required; separate technical design was not triggered by approved `spec.md`.
- Task-ledger review fan-out: local-only rationale recorded in `workflow-plans/planning.md`; no spawned lanes because the available `spawn_agent` tool requires explicit user permission for subagents, and the ledger has one tightly coupled instruction/docs/guardrail proof surface.

## Tasks

- [x] T001 [Instruction Layer] Create root `SOUL.md` with the approved compact personality shape.
  Depends on: none.
  Files: `SOUL.md`.
  Proof: targeted read verifies required sections `Role`, `Core Operating Beliefs`, `Engineering Balance`, `Default Behavior Under Ambiguity`, `Communication Style`, `Avoid`, and `Boundaries`; targeted read verifies the file contains the AGENTS/task-local precedence boundary and does not add commands, paths, artifact templates, workflow gates, validation matrices, or repository architecture rules.
  Evidence: PASS - added root `SOUL.md`; targeted `rtk rg` section/boundary scan verified all required sections and the AGENTS/task-local precedence boundary; targeted command-pattern scan for RTK, make, Go test, base/head refs, and fenced command blocks returned no command/template/path instructions.

- [x] T002 [Instruction Bridge] Update `AGENTS.md` with the lower-precedence SOUL bridge and include/reference.
  Depends on: T001.
  Files: `AGENTS.md`.
  Proof: targeted read verifies a short rule near the authority/purpose section says `SOUL.md` is lower-precedence orchestrator personality guidance; `AGENTS.md`, `docs/spec-first-workflow.md`, task-local artifacts, and explicit user/system/developer instructions remain operational authority; `@SOUL.md` and existing `@RTK.md` references are both present.
  Evidence: PASS - `rtk sed -n '1,18p' AGENTS.md` verified the lower-precedence SOUL rule near the authority/purpose section; targeted `rtk rg` verified the SOUL reference, the AGENTS/docs/task-local/user-system-developer override boundary, `@SOUL.md`, and existing `@RTK.md`.

- [x] T003 [Guardrails] Extend required repository guardrails for SOUL presence and AGENTS/SOUL precedence drift.
  Depends on: T001, T002.
  Files: `scripts/ci/required-guardrails-check.sh`.
  Proof: targeted read verifies `SOUL.md` is in the required file list; regex checks require `AGENTS.md` to reference `SOUL.md`; regex checks require both `AGENTS.md` and `SOUL.md` to preserve the lower-precedence boundary; no unrelated guardrail policies or Go import-boundary checks are weakened.
  Evidence: PASS - `rtk sed -n '1,36p' scripts/ci/required-guardrails-check.sh` verified `SOUL.md` in the required file list; `rtk sed -n '112,152p' scripts/ci/required-guardrails-check.sh` verified AGENTS/SOUL reference and lower-precedence boundary regexes in both files; targeted `rtk rg` verified the added SOUL checks and the existing Go import-boundary check remain present.

- [x] T004 [Docs] Update the docs surface required by the `scripts/ci/` guardrail change.
  Depends on: T003.
  Files: `docs/build-test-and-development-commands.md` unless implementation evidence identifies a more precise docs surface.
  Proof: targeted read verifies the guardrails-check documentation mentions root `SOUL.md` and the AGENTS/SOUL lower-precedence boundary; docs-drift policy has a docs change to pair with the `scripts/ci/` edit.
  Evidence: PASS - targeted `rtk rg -n -C 2` for `make guardrails-check`, root `SOUL.md`, and `AGENTS/SOUL lower-precedence boundary` verified the `guardrails-check` docs mention root `SOUL.md` and the AGENTS/SOUL lower-precedence boundary, pairing the `scripts/ci/` edit with a docs update.

- [x] T005 [Validation] Run and record the required validation loop.
  Depends on: T001, T002, T003, T004.
  Files: repository validation surface.
  Proof: `rtk make guardrails-check`; `rtk make agents-check`; `rtk git diff --check`; if valid refs exist, `rtk make docs-drift-check BASE_REF=<base> HEAD_REF=<head>`, otherwise record the exact reason docs-drift could not run locally and verify the docs update from T004 is present.
  Evidence: PASS - `rtk make guardrails-check` passed (`required repository guardrails check passed`); `rtk make agents-check` passed (`agents check complete`); `rtk git diff --check` passed; docs drift passed with `rtk make docs-drift-check BASE_REF=origin/main HEAD_REF=46390788a1122e123e461cb18917b8b9b9a8a512` after creating that temporary tracked-worktree commit with `rtk proxy git stash create docs-drift-current-worktree` because branch `HEAD` did not include the uncommitted `scripts/ci/` and docs edits.

- [x] T006 [Closeout] Record implementation evidence in ledger-owned closeout surfaces.
  Depends on: T005.
  Files: `specs/orchestrator-soul-md/tasks.md`, `specs/orchestrator-soul-md/spec.md`.
  Proof: this ledger's checkboxes and evidence lines match the completed work and fresh proof; `spec.md` validation/outcome is updated from actual evidence; no new workflow/process artifacts are created after implementation starts and `workflow-plan.md` remains pre-code routing history.
  Evidence: PASS - updated this ledger's checkboxes and evidence lines for T001-T006; updated `specs/orchestrator-soul-md/spec.md` status, validation, and outcome from actual implementation proof; no new workflow/process artifacts were created after implementation start and `workflow-plan.md` remained pre-code routing history; post-closeout `rtk git diff --check` passed.
