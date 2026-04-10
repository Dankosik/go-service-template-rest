# Session-Phase Skills Prompt Pack

Status: future implementation prompt pack, not yet part of the approved repository contract

## Purpose

This file preserves ready-to-run prompts for a possible next workflow iteration:

- `one session = one phase` for non-trivial work
- orchestrator-facing skills for each session-phase
- one master `workflow-plan.md` plus a phase-specific workflow plan for each phase

It is a prompt pack for future implementation sessions, not an approved workflow by itself.

## Assumed Direction

These prompts assume the repository will explore a model like this:

```text
specs/<feature-id>/
  workflow-plan.md                # master control artifact
  workflow-plans/
    workflow-planning.md
    research.md
    specification.md
    technical-design.md
    planning.md
    implementation-phase-01.md
    review-phase-01.md
    validation-phase-01.md
  spec.md
  design/
  plan.md
  test-plan.md
  rollout.md
```

Where:
- `workflow-plan.md` stays the task entrypoint, current status, and resume index
- `workflow-plans/<phase>.md` captures the orchestration contract for one specific phase
- a dedicated orchestrator-facing skill exists for each session-phase

## Shared Design Rules To Reuse In Every Session

Every future implementation prompt in this pack assumes these rules:

- `AGENTS.md` remains the authority for workflow policy, ownership, gates, and artifact authority
- `docs/spec-first-workflow.md` remains the detailed workflow mechanics and artifact-order reference
- phase/session skills do not become the new source of truth for workflow policy
- artifact file shapes should live in one repository-level place, not be restated differently in every skill
- the session skill for phase `X` must not silently start phase `Y`
- every phase session must update the master `workflow-plan.md`
- phase-specific `workflow-plans/<phase>.md` are local routing/control artifacts for one phase only

## Prompt 1: Update The Contract For Session-Phase Workflow Plans

```text
Implement the repository workflow-contract update for session-bounded phases and per-phase workflow plans.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- docs/repo-architecture.md
- README.md
- specs/artifact-driven-workflow-redesign/spec.md
- specs/artifact-driven-workflow-redesign/plan.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Update the repository contract so non-trivial work can support:
  - one session = one phase
  - one master `workflow-plan.md`
  - one phase-specific workflow plan per phase under `workflow-plans/`

Required changes:
- keep `workflow-plan.md` as the master control artifact
- add a second layer:
  - `workflow-plans/<phase>.md`
- define the role split:
  - `workflow-plan.md` = current phase, artifact status, next session, blockers, links to phase workflow plans
  - `workflow-plans/<phase>.md` = phase-local orchestration for one phase only
- update the non-trivial artifact model accordingly
- define which phases normally get their own `workflow-plans/<phase>.md`
- define which post-code phases may be conditional (`implementation-phase-N`, `review-phase-N`, `validation-phase-N`)
- update resume order to read:
  - master `workflow-plan.md`
  - current `workflow-plans/<phase>.md`
  - then the phase artifacts
- encode that a non-trivial session should complete one phase and stop before starting the next
- preserve direct-path exceptions for tiny work

Constraints:
- do not let `workflow-plans/<phase>.md` replace the master `workflow-plan.md`
- do not let phase workflow plans become competing design or execution artifacts
- keep the wording repository-native and compatible with the artifact-driven redesign

Validation:
- `rg -n "workflow-plans/|one session|next session|current phase|phase workflow" AGENTS.md docs/spec-first-workflow.md README.md`
- manual comparison between AGENTS.md and docs/spec-first-workflow.md

Deliverable:
- updated contract docs only
```

## Prompt 2: Create `workflow-planning-session`

```text
Create the orchestrator-facing skill for the workflow-planning session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/workflow-planning-session/SKILL.md`
  - `.agents/skills/workflow-planning-session/evals/evals.json`

Skill purpose:
- one session dedicated only to workflow planning
- produce or update:
  - master `workflow-plan.md`
  - `workflow-plans/workflow-planning.md`

The skill must define:
- required inputs
- what must be read first
- allowed writes
- prohibited actions
- expected outputs
- stop condition
- required updates to the master `workflow-plan.md`

Hard boundaries:
- do not run research
- do not write `spec.md`
- do not write `design/`
- do not write `plan.md`
- do not drift into implementation

Constraints:
- this is an orchestrator-facing wrapper skill, not a domain specialist
- keep it aligned with repository workflow ownership
- do not redefine the workflow differently from AGENTS.md

Validation:
- validate the eval JSON
- re-read the skill against AGENTS.md and docs/spec-first-workflow.md

Deliverable:
- new canonical skill bundle only
```

## Prompt 3: Create `research-session`

```text
Create the orchestrator-facing skill for the research session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/research-session/SKILL.md`
  - `.agents/skills/research-session/evals/evals.json`

Skill purpose:
- one session dedicated to local or fan-out research only
- produce or update:
  - master `workflow-plan.md`
  - `workflow-plans/research.md`
  - optional `research/*.md`

The skill must define:
- how to read the current phase context
- how to plan research lanes for this session
- what outputs count as a finished research session
- what must be written back into `workflow-plan.md`
- how to stop before specification begins

Hard boundaries:
- do not finalize `spec.md`
- do not start `technical design`
- do not start `planning`
- no implementation

Constraints:
- support both local research and read-only subagent fan-out
- keep the skill session-bounded

Validation:
- validate eval JSON
- compare boundaries with the future `specification-session`

Deliverable:
- new canonical skill bundle only
```

## Prompt 4: Create `specification-session`

```text
Create the orchestrator-facing skill for the specification session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- .agents/skills/spec-document-designer/SKILL.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/specification-session/SKILL.md`
  - `.agents/skills/specification-session/evals/evals.json`

Skill purpose:
- one session dedicated to specification only
- likely reuse `spec-document-designer` internally as the deeper method, but expose a phase/session wrapper
- produce or update:
  - approved `spec.md`
  - master `workflow-plan.md`
  - `workflow-plans/specification.md`

The skill must define:
- required inputs from previous phases
- what counts as spec-ready input
- what the session may write
- what the session must not write
- what marks specification complete
- what gets handed off to technical design

Hard boundaries:
- do not assemble `design/`
- do not write `plan.md`
- do not start implementation

Constraints:
- do not turn this wrapper skill into a second copy of `spec-document-designer`
- keep the wrapper focused on session boundaries and outputs

Validation:
- validate eval JSON
- compare boundaries against `spec-document-designer`

Deliverable:
- new canonical skill bundle only
```

## Prompt 5: Create `technical-design-session`

```text
Create the orchestrator-facing skill for the technical-design session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- docs/repo-architecture.md
- .agents/skills/go-design-spec/SKILL.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/technical-design-session/SKILL.md`
  - `.agents/skills/technical-design-session/evals/evals.json`

Skill purpose:
- one session dedicated to task-local technical design only
- likely reuse `go-design-spec` internally as the deeper method, but expose a phase/session wrapper
- produce or update:
  - `design/`
  - master `workflow-plan.md`
  - `workflow-plans/technical-design.md`

The skill must define:
- required inputs
- when to load `docs/repo-architecture.md`
- required design artifacts
- conditional design artifacts
- phase-local stop condition
- what must be handed off to planning

Hard boundaries:
- do not reopen problem framing casually
- do not write `plan.md`
- do not start implementation

Constraints:
- `design/contracts/` must remain design-only context and not a runtime source of truth
- keep the wrapper skill session-oriented instead of duplicating `go-design-spec`

Validation:
- validate eval JSON
- compare boundaries against `go-design-spec`

Deliverable:
- new canonical skill bundle only
```

## Prompt 6: Create `planning-session`

```text
Create the orchestrator-facing skill for the planning session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- .agents/skills/planning-and-task-breakdown/SKILL.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/planning-session/SKILL.md`
  - `.agents/skills/planning-session/evals/evals.json`

Skill purpose:
- one session dedicated to implementation planning only
- likely reuse `planning-and-task-breakdown` internally as the deeper method, but expose a phase/session wrapper
- produce or update:
  - `plan.md`
  - optional `test-plan.md`
  - optional `rollout.md`
  - master `workflow-plan.md`
  - `workflow-plans/planning.md`

The skill must define:
- required inputs from `spec.md + design/`
- allowed outputs
- planning completion criteria
- handoff into implementation
- stop condition that prevents starting implementation in the same session

Hard boundaries:
- do not implement code
- do not reopen spec or design silently

Constraints:
- keep the wrapper focused on one planning session
- do not duplicate the whole deeper planning skill

Validation:
- validate eval JSON
- compare boundaries against `planning-and-task-breakdown`

Deliverable:
- new canonical skill bundle only
```

## Prompt 7: Create `implementation-phase-session`

```text
Create the orchestrator-facing skill for one implementation-phase session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- .agents/skills/go-coder/SKILL.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/implementation-phase-session/SKILL.md`
  - `.agents/skills/implementation-phase-session/evals/evals.json`

Skill purpose:
- one session dedicated to exactly one implementation phase or one bounded checkpoint from `plan.md`
- produce or update:
  - code and related artifacts for one phase only
  - master `workflow-plan.md`
  - `workflow-plans/implementation-phase-<n>.md`

The skill must define:
- how to select one explicit phase/checkpoint
- what inputs must be loaded
- what counts as “phase complete”
- how validation for that phase is recorded
- when the session must stop

Hard boundaries:
- do not begin the next implementation phase in the same session by default
- do not silently rewrite planning
- do not skip validation for the implemented phase

Constraints:
- this is an orchestrator-facing phase wrapper, not a replacement for `go-coder`
- the skill should assume `go-coder` is used within the allowed implementation boundary

Validation:
- validate eval JSON
- compare boundaries against `go-coder`

Deliverable:
- new canonical skill bundle only
```

## Prompt 8: Create `validation-closeout-session`

```text
Create the orchestrator-facing skill for the validation/closeout session.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- .agents/skills/go-verification-before-completion/SKILL.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Create a new canonical skill bundle:
  - `.agents/skills/validation-closeout-session/SKILL.md`
  - `.agents/skills/validation-closeout-session/evals/evals.json`

Skill purpose:
- one session dedicated to final validation and closeout only
- produce or update:
  - final validation evidence
  - `Outcome`
  - master `workflow-plan.md`
  - `workflow-plans/validation-phase-<n>.md` when phase-local validation is used

The skill must define:
- required proof inputs
- what “done” means
- how reopen conditions are recorded
- how to stop instead of falling back into silent implementation

Hard boundaries:
- do not implement new code as part of closeout
- do not claim completion without fresh evidence

Constraints:
- keep it aligned with `go-verification-before-completion`

Validation:
- validate eval JSON
- compare boundaries against `go-verification-before-completion`

Deliverable:
- new canonical skill bundle only
```

## Prompt 9: Add A Repository-Level Contract For `workflow-plans/<phase>.md`

```text
Add the repository-level contract for per-phase workflow plans.

Read first:
- AGENTS.md
- docs/spec-first-workflow.md
- docs/repo-architecture.md
- specs/artifact-driven-workflow-redesign/research/session-phase-skills-direction.md

Task:
- Update the repository instructions so that, for the future session-bounded workflow, each non-trivial phase may have its own phase-local workflow plan under `workflow-plans/`.

Required changes:
- define the role of:
  - master `workflow-plan.md`
  - `workflow-plans/<phase>.md`
- define the minimum shape of each phase workflow plan, for example:
  - `Phase`
  - `Goal`
  - `Inputs`
  - `Artifacts To Read`
  - `Allowed Skills`
  - `Planned Subagents`
  - `Allowed Writes`
  - `Out Of Scope`
  - `Expected Outputs`
  - `Completion Criteria`
  - `Workflow-Plan Update`
  - `Next Session Handoff`
- define which phases normally get a file
- define how the master `workflow-plan.md` links to phase workflow plans
- define resume order:
  - first master `workflow-plan.md`
  - then current `workflow-plans/<phase>.md`
  - then the phase artifacts

Constraints:
- do not let `workflow-plans/<phase>.md` replace artifact ownership or policy from AGENTS.md
- do not let them become second specs or second plans

Validation:
- `rg -n "workflow-plans/|Phase|Next Session Handoff|Allowed Skills|Planned Subagents" AGENTS.md docs/spec-first-workflow.md`
- manual comparison for drift between the two docs

Deliverable:
- updated repository instruction docs only
```

## Prompt 10: Update Discoverability And Mirrors After The Phase-Skill Work Lands

```text
After the phase/session skills and workflow-plan contract changes are implemented, update discoverability and runtime mirrors.

Read first:
- README.md
- AGENTS.md
- docs/spec-first-workflow.md
- all new `.agents/skills/*-session/` skills

Task:
- Update README and any discoverability surfaces so the repository explains:
  - session-bounded phases
  - phase/session skills
  - master `workflow-plan.md`
  - `workflow-plans/<phase>.md`
- sync the canonical skills into runtime mirrors

Required changes:
- update README workflow overview
- add phase/session skills to the skill catalog
- mention the role split between master workflow plan and phase workflow plans
- run:
  - `bash ./scripts/dev/sync-skills.sh`
  - `bash ./scripts/dev/sync-skills.sh --check`

Validation:
- `git diff --check`
- `bash ./scripts/dev/sync-skills.sh --check`
- `rg -n "workflow-plans/|session|phase" README.md`

Deliverable:
- updated discoverability surfaces
- synced mirrors
```

## Suggested Order

Recommended future rollout order:

1. Prompt 1: contract update for session-bounded phases and `workflow-plans/`
2. Prompts 2-8: create the phase/session skills
3. Prompt 9: add the repository-level phase-workflow-plan contract if Prompt 1 left gaps
4. Prompt 10: update discoverability and sync mirrors

## Notes

- These prompts assume the repository will intentionally explore a stricter session-boundary model.
- If future research rejects `one session = one phase`, do not apply this pack blindly.
- If the repository decides to adapt existing skills instead of adding wrappers, keep the wrapper/session intent and collapse the file count, but preserve the boundaries.
