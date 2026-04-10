# Session Prompts

These prompts assume the target architecture in:
- `specs/artifact-driven-workflow-redesign/spec.md`
- `specs/artifact-driven-workflow-redesign/plan.md`

Use them as copy-paste starting points for separate implementation sessions.

Session rule for all prompts:
- one prompt = one session-bounded phase
- finish the named phase, update the owning artifacts, and stop
- do not begin the next phase in the same session

## Session A: Rewrite `AGENTS.md`

```text
Implement the agreed workflow-contract rewrite for this repository.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/spec.md
- specs/artifact-driven-workflow-redesign/plan.md

Task:
- Update AGENTS.md so it reflects the approved target architecture from specs/artifact-driven-workflow-redesign/spec.md.

Required changes:
- Add `technical design` as an explicit workflow state between `specification` and `planning`.
- Change the non-trivial artifact chain from `workflow-plan.md -> spec.md -> plan.md` to `workflow-plan.md -> spec.md -> design/ -> plan.md`.
- Keep `spec.md` as the canonical decisions artifact, but make clear that non-trivial planning should not proceed from `spec.md` alone.
- Define the task-local design bundle and list:
  - required core artifacts:
    - `design/overview.md`
    - `design/component-map.md`
    - `design/sequence.md`
    - `design/ownership-map.md`
  - conditional artifacts:
    - `design/data-model.md`
    - `design/dependency-graph.md`
    - `design/contracts/`
    - `test-plan.md`
    - `rollout.md`
- Redefine `workflow-plan.md` as a live resume/control artifact, not just a pre-research routing note.
- Update workflow gates so non-trivial implementation planning requires approved `spec.md + design/` unless there is an explicit design-skip rationale for a tiny/direct-path task.
- Update the resume/read-order rules accordingly.
- Preserve the repository’s orchestrator/subagent ownership model.

Constraints:
- Do not import foreign templates.
- Do not rewrite the repository into a heavyweight mandatory process for tiny tasks.
- Keep the language repository-native and consistent with the existing style of AGENTS.md.

Validation:
- `rg -n "technical design|design/|repo-architecture|resume|workflow-plan" AGENTS.md`
- manually re-read AGENTS.md for internal consistency

Deliverable:
- updated AGENTS.md only

Stop condition:
- after `AGENTS.md` and the active task-local artifacts reflect the session-boundary-aware contract, stop
- do not continue into `docs/spec-first-workflow.md`, skills, or README in this session
```

## Session B: Rewrite `docs/spec-first-workflow.md`

```text
Implement the detailed workflow-doc rewrite for this repository.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/spec.md
- specs/artifact-driven-workflow-redesign/plan.md

Task:
- Update docs/spec-first-workflow.md so it becomes the detailed runtime companion to the new artifact-driven workflow.

Required changes:
- Update the artifact model to introduce the `design/` bundle between `spec.md` and `plan.md`.
- Add `technical design` to the execution loop.
- Define the purpose of:
  - `workflow-plan.md`
  - `spec.md`
  - `design/`
  - `plan.md`
  - `research/*.md`
  - `test-plan.md`
  - `rollout.md`
- Document the required core design artifacts and the trigger rules for the conditional ones.
- Explain how `workflow-plan.md` now tracks current stage, blockers, next action, and resume order.
- Update the planning-entry/planning sections so `planning-and-task-breakdown` consumes approved `spec.md + design/` for non-trivial work.
- Add clear resume guidance: which artifacts to read first in a later session and how to infer the current stage from the artifacts.
- Keep direct-path and lightweight-local exceptions, but require explicit skip rationale for bypassing the design bundle.

Constraints:
- Keep this doc detailed, but do not let it become a second AGENTS.md.
- Keep examples concise and repository-native.
- Do not silently redefine ownership differently from AGENTS.md.

Validation:
- `rg -n "technical design|design/|repo-architecture|resume order|workflow-plan|rollout.md" docs/spec-first-workflow.md`
- compare the updated wording with AGENTS.md for drift

Deliverable:
- updated docs/spec-first-workflow.md only

Stop condition:
- after `docs/spec-first-workflow.md` and the active task-local artifacts reflect the detailed runtime model, stop
- do not continue into skills or README in this session
```

## Session C: Add `docs/repo-architecture.md`

```text
Author the new repository-wide architecture baseline document.

Read first:
- docs/project-structure-and-module-organization.md
- docs/build-test-and-development-commands.md
- docs/configuration-source-policy.md
- docs/ci-cd-production-ready.md
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/spec.md

Task:
- Create docs/repo-architecture.md as the stable repository-wide architecture baseline referenced by the new workflow.

The doc should cover:
- repository-wide component boundaries
- source-of-truth ownership
- stable dependency direction
- one clear description of primary runtime flows:
  - request/response path
  - startup/shutdown path
  - background/async extension path if applicable
- extension seams for future tasks
- links to existing structure/config/commands/CI docs instead of duplicating them

The doc should not do:
- task-local design
- speculative future architecture
- a full onboarding manual
- restate every command or every file in the repo

Constraints:
- keep it concise but technically useful for an LLM
- align with the existing repository layout and boundaries
- write it as a stable baseline, not as a feature-specific example

Validation:
- manual overlap check against:
  - docs/project-structure-and-module-organization.md
  - docs/build-test-and-development-commands.md
  - docs/configuration-source-policy.md
  - docs/ci-cd-production-ready.md
- `git diff --check`

Deliverable:
- new docs/repo-architecture.md
- only add cross-links elsewhere if they are clearly useful

Stop condition:
- after `docs/repo-architecture.md` is stable and any clearly necessary cross-links are done, stop
- do not continue into skill or README cleanup in this session
```

## Session D: Align Skills With The New Artifact Model

```text
Align the repository’s planning/design skills with the approved artifact-driven workflow.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/spec.md
- .agents/skills/spec-document-designer/SKILL.md
- .agents/skills/go-design-spec/SKILL.md
- .agents/skills/planning-and-task-breakdown/SKILL.md

Task:
- Update the relevant skills so they match the new workflow after the contract docs are rewritten.

Required changes:
- `spec-document-designer`
  - keep it focused on `spec.md`
  - make clear it hands off to technical design rather than directly to execution planning for non-trivial work
- `go-design-spec`
  - repurpose it toward integrated technical-design-bundle assembly/reconciliation
  - make it responsible for leaving `design/` stable enough for planning
  - remove “final spec assembly” framing if it conflicts with the new model
- `planning-and-task-breakdown`
  - update it to plan from `spec.md + design/`
  - escalate if required design artifacts are missing
  - avoid assuming that architecture/data/sequence context can be reconstructed from `spec.md` alone

Constraints:
- do not change the repository ownership model
- do not create a brand-new skill unless adaptation clearly fails
- keep the three skills’ boundaries sharp and non-overlapping

Validation:
- `rg -n "spec.md|design/|technical design|plan.md" .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md`
- manual read for ownership drift across the three skill files

Deliverable:
- updated skill files only

Stop condition:
- after the canonical and mirrored skill files are aligned, stop
- do not continue into README/discoverability cleanup in this session
```

## Session E: Update README And Discoverability Surfaces

```text
Update discoverability surfaces so the new workflow is visible where users actually look.

Read first:
- README.md
- AGENTS.md
- docs/spec-first-workflow.md
- docs/repo-architecture.md
- specs/artifact-driven-workflow-redesign/spec.md

Task:
- Update README.md to reflect the approved artifact-driven workflow.

Required changes:
- mention the new `technical design` stage for non-trivial work
- explain the role split between:
  - `spec.md`
  - `design/`
  - `plan.md`
- update any skill descriptions that still imply planning from spec alone
- make `docs/repo-architecture.md` discoverable if that fits the current README structure
- keep the richer model framed as default for non-trivial work, not mandatory ceremony for tiny fixes

Constraints:
- keep README concise
- do not turn README into a second workflow specification
- stay consistent with AGENTS.md and docs/spec-first-workflow.md

Validation:
- `rg -n "design/|technical design|repo-architecture|plan.md|spec.md" README.md`
- manual read of the workflow and skill-library sections

Deliverable:
- updated README.md only

Stop condition:
- after `README.md` reflects the final workflow model and discoverability is clean, stop
- do not continue into optional example/template work in this session
```

## Optional Session F: Add One Compact Example

```text
Only run this if the rewritten workflow still feels too abstract after the contract and skill updates land.

Read first:
- specs/artifact-driven-workflow-redesign/spec.md
- AGENTS.md
- docs/spec-first-workflow.md
- docs/repo-architecture.md

Task:
- Add one compact example or reference showing how a non-trivial task uses the design bundle.

The example should show:
- required design artifacts
- when `data-model.md` is created
- when `contracts/` is created
- when `dependency-graph.md` is skipped
- how `plan.md` consumes the approved design bundle

Constraints:
- keep it compact
- do not create a second source of truth
- do not introduce a mandatory template unless the repository explicitly wants one

Validation:
- manual consistency check against AGENTS.md and docs/spec-first-workflow.md

Deliverable:
- one example/reference artifact only if clearly needed
```
