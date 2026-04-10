## Phase 1: Core Contract Rewrite

Objective:
- update the repository's primary workflow contract so the new artifact architecture is official

Files:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Tasks:
- add `technical design` as an explicit workflow state/stage
- update the high-level execution loops to include the design-bundle stage
- change the artifact model from `spec -> plan` to `spec -> design -> plan` for non-trivial work
- define required vs conditional design-bundle artifacts
- redefine `workflow-plan.md` as a live resume/control artifact
- add a session-boundary gate so non-trivial work stops after one session-bounded phase
- define which phase transitions require a new session
- define how `workflow-plan.md` records `current session scope`, `phase status`, `completion marker`, `Session boundary reached`, `Ready for next session`, and `Next session starts with`
- update planning-entry and planning gates so non-trivial work plans from approved `spec.md + design/`
- document explicit skip rationale for tiny/direct-path tasks

Acceptance Criteria:
- both core docs describe the same target workflow
- `AGENTS.md` and `docs/spec-first-workflow.md` no longer imply that `plan.md` derives from `spec.md` alone for non-trivial work
- the design-bundle stage and artifact responsibilities are explicit
- resume order is written down
- same-session phase hopping is explicitly blocked for non-trivial work unless an upfront waiver exists

Planned Verification:
- `rg -n "technical design|design/|repo-architecture|resume order|design bundle|workflow-plan" AGENTS.md docs/spec-first-workflow.md`
- manual diff read for contract drift between the two files

Review / Checkpoint:
- stop after both files are updated and compare wording side-by-side before touching skills

Exit Criteria:
- the repository's top-level workflow contract is stable enough that downstream skill updates can follow it instead of redefining it

## Phase 2: Repository-Wide Architecture Baseline

Objective:
- add the stable repo-wide context document that future tasks can load before task-local design

Files:
- `docs/repo-architecture.md` (new)
- optional link updates in:
  - `docs/project-structure-and-module-organization.md`
  - `README.md`

Tasks:
- write `docs/repo-architecture.md`
- keep it focused on stable architecture and ownership, not task-local design
- describe:
  - primary component boundaries
  - source-of-truth ownership
  - request/runtime/startup/background flow baselines
  - extension seams
  - links to existing commands/config/CI docs
- add minimal cross-links from existing docs if useful

Acceptance Criteria:
- the new doc helps an LLM understand the repo baseline without re-reading the whole tree
- it does not duplicate task-local design artifacts
- it links cleanly to structure/config/commands/CI docs instead of restating them all

Planned Verification:
- manual read for overlap drift against:
  - `docs/project-structure-and-module-organization.md`
  - `docs/build-test-and-development-commands.md`
  - `docs/configuration-source-policy.md`
  - `docs/ci-cd-production-ready.md`
- `git diff --check`

Review / Checkpoint:
- confirm the new doc is stable baseline architecture, not a generic onboarding essay

Exit Criteria:
- later design-bundle artifacts can reference `docs/repo-architecture.md` as the repository baseline

## Phase 3: Skill Alignment

Objective:
- align skill boundaries with the new artifact model

Files:
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/go-design-spec/SKILL.md`
- `.agents/skills/planning-and-task-breakdown/SKILL.md`
- optional supporting references if they become inconsistent

Tasks:
- keep `spec-document-designer` focused on `spec.md`
- repurpose `go-design-spec` toward integrated design-bundle assembly/reconciliation
- update `planning-and-task-breakdown` so non-trivial work plans from `spec.md + design/`
- remove wording that still assumes planning from spec alone
- add explicit escalation when required design artifacts are missing
- make each skill stop at its session handoff boundary instead of flowing into the next phase

Acceptance Criteria:
- each skill has one clear artifact boundary
- skill descriptions match the rewritten core contract docs
- no skill quietly redefines the workflow differently from `AGENTS.md`
- spec, design, and planning skills explicitly end their session at the handoff boundary

Planned Verification:
- `rg -n "plan.md|spec.md|design/|technical design|artifact" .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md`
- manual read for boundary overlap or gaps

Review / Checkpoint:
- compare the three skills together, not one by one, to catch ownership drift

Exit Criteria:
- a future orchestrator can load the right skill at the right stage without improvising artifact ownership

## Phase 4: Discoverability And Surface Cleanup

Objective:
- update user-facing discoverability surfaces so the new workflow is visible where people actually look

Files:
- `README.md`
- optional doc index/link surfaces discovered during implementation

Tasks:
- update the workflow overview to mention the design-bundle stage
- mention the session-bounded phase rule for non-trivial work
- update skill descriptions where they mention planning from spec alone
- add discoverability for `docs/repo-architecture.md` if appropriate
- ensure the richer model is described as default for non-trivial work, not mandatory for tiny changes

Acceptance Criteria:
- the README reflects the new workflow accurately
- discoverability surfaces do not teach the old simplified model
- the repo baseline and design bundle are easy to find
- the README makes the stop-after-phase rule visible without turning into a second workflow spec

Planned Verification:
- `rg -n "spec.md|plan.md|design/|repo-architecture|workflow" README.md`
- manual read of workflow and skill-library sections

Review / Checkpoint:
- confirm discoverability language stays concise and does not become a second workflow spec

Exit Criteria:
- new contributors and future sessions can discover the updated model without digging through implementation history

## Phase 5: Optional Example And Template Follow-Up

Objective:
- add one small but concrete example/template layer if the rewritten docs still feel too abstract in practice

Files:
- optional new example docs under `docs/` or `specs/`
- optional reference updates for skills

Tasks:
- decide whether a design-bundle example is needed after phases 1-4 land
- if needed, add one compact example showing:
  - when to create `data-model.md`
  - when to create `contracts/`
  - when to skip `dependency-graph.md`

Acceptance Criteria:
- examples remove ambiguity without creating a second source of truth
- no example becomes a hidden mandatory template

Planned Verification:
- manual read for consistency with core contract docs

Review / Checkpoint:
- only do this phase if real ambiguity remains after the contract rewrite

Exit Criteria:
- optional only; skip if phases 1-4 are already sufficient

## Recommended Execution Order

Wave 1:
1. Phase 1: Core Contract Rewrite
2. Phase 2: Repository-Wide Architecture Baseline

Wave 2:
1. Phase 3: Skill Alignment
2. Phase 4: Discoverability And Surface Cleanup

Wave 3:
1. Phase 5 only if needed

## Session Execution Policy

- Treat each numbered phase in this plan as a separate session by default.
- Finish the phase, update the owning artifacts plus `workflow-plan.md`, and stop.
- Start the next numbered phase in a new session.
- If a phase reopens earlier work, stop after recording the reopen target instead of continuing across the boundary in the same session.
- Only tiny/direct-path work may collapse those boundaries, and only with an upfront recorded waiver.

## Parallelization Notes

Safe to run in parallel once this spec is fixed:
- Session A: `AGENTS.md`
- Session B: `docs/spec-first-workflow.md`
- Session C: `docs/repo-architecture.md`

But reconcile A + B together before skill updates.

Safe to run in parallel after A/B/C are stable:
- Session D: skill alignment
- Session E: README/discoverability cleanup

Avoid parallelizing:
- `AGENTS.md` and skill updates in the same first wave
- skill updates before the contract docs settle

## Handoffs / Reopen Conditions

Reopen the target architecture if:
- the contract rewrite reveals that `go-design-spec` cannot be adapted cleanly without confusing its purpose
- `design/dependency-graph.md` proves to be either always-needed or nearly-never-needed in practice
- `docs/repo-architecture.md` starts duplicating task-local design rather than staying repository-wide
- planning still needs to reconstruct design context after the new bundle is in place
