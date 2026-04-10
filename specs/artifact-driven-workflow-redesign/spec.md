## Context

The current repository workflow already separates orchestration, specification, and execution better than a plain `spec -> tasks -> code` loop: `workflow-plan.md` owns routing/control for non-trivial work, `spec.md` owns final decisions, and `plan.md` owns coder-facing execution order. That model is strong for bounded changes, but it still under-specifies the technical design context that a coding LLM needs for non-trivial implementation.

The missing layer is not more product requirements. The missing layer is task-specific technical context: how the change fits the repository architecture, which components and files participate, what the runtime interaction sequence is, where ownership and source-of-truth boundaries sit, what can run in parallel, what must stay sequential, and where data flow or side effects create correctness risk.

The next open seam is session control. The repository now has stronger artifacts and resume order, but it still under-specifies when a session must stop after finishing one workflow phase versus continuing directly into the next one. Without an explicit session-to-phase rule, artifact boundaries remain real on paper but soft in practice.

Research across GitHub Spec Kit, BMAD Method, Superpowers, and Spec-Driven Workflow showed the same stable pattern: mature AI-assisted workflows do not rely on a single requirements spec alone for non-trivial implementation. They carry additional technical context forward through separate artifacts, but the useful pattern is the role separation, not any one foreign template.

The user explicitly chose the richer direction: do not collapse the missing context into one `design.md`; adopt a design-bundle model with multiple technical artifacts when the task size and risk justify it.

## Scope / Non-goals

In scope:
- define the target artifact architecture for non-trivial work in this repository
- define the staged workflow from framing through implementation readiness
- define the session-to-phase boundary model for non-trivial work
- define repository-wide baseline artifacts vs per-task artifacts
- define which design-bundle artifacts are mandatory vs conditional
- define trigger rules for conditional artifacts
- define multi-session resume rules and artifact reading order
- define which current repository instructions and skills need later alignment
- provide a phased implementation plan and reusable prompts for future sessions

Non-goals:
- implementing all workflow, skill, and documentation changes in this pass
- importing GitHub Spec Kit, BMAD, Superpowers, or Spec-Driven Workflow templates verbatim
- forcing the full design bundle onto tiny direct-path tasks
- turning `spec.md` into a PRD, architecture encyclopedia, or task board
- making every possible technical concern a standalone artifact by default

## Constraints

- `spec.md` remains the canonical decisions artifact.
- `workflow-plan.md` and `plan.md` remain first-class artifacts; they are not being replaced.
- The workflow must optimize for LLM implementation quality and multi-session resumability, not human-only document aesthetics.
- Repository-wide stable knowledge and per-task design knowledge must stay clearly separated.
- Artifact roles must stay single-purpose enough to avoid competing sources of truth.
- Richer documentation is allowed only when it removes real ambiguity, planning drift, or resume friction.
- The repository's orchestrator/subagent ownership model remains unchanged: final decisions stay with the orchestrator; subagents remain research-only/read-only.

## Decisions

1. Adopt a design-bundle architecture for non-trivial technical work.
   - New default artifact chain:
     `workflow-plan.md -> spec.md -> design/ -> plan.md -> implementation`
   - For tiny or direct-path work, this chain may still collapse locally with an explicit skip rationale.

2. Add a repository-wide stable architecture baseline artifact.
   - New file: `docs/repo-architecture.md`
   - Purpose:
     - describe stable component boundaries and ownership
     - describe major runtime flows and extension seams
     - map source-of-truth responsibilities
     - link to existing structure, config, commands, and CI policy docs
   - This file should prevent each new task from re-deriving basic repository architecture from scratch.

3. Make `technical design` an explicit stage between `specification` and `planning`.
   - Updated high-level flow for non-trivial work:
     `intake -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> implementation -> review/reconciliation -> validation`
   - `technical design` produces the task-local design bundle.

4. Use a multi-file design bundle rather than a single overloaded design document.
   - New default task-local layout:

```text
specs/<feature-id>/
  workflow-plan.md
  spec.md
  design/
    overview.md
    component-map.md
    sequence.md
    ownership-map.md
    data-model.md          # conditional
    dependency-graph.md    # conditional
    contracts/             # conditional
  plan.md
  test-plan.md             # conditional
  rollout.md               # conditional
```

5. Define the required core design-bundle artifacts for non-trivial work.
   - `design/overview.md`
     - purpose: design entrypoint, chosen approach, artifact index, unresolved seams, and readiness summary
   - `design/component-map.md`
     - purpose: affected packages/modules/components, responsibilities, and what changes vs what remains stable
   - `design/sequence.md`
     - purpose: call order, sync/async boundaries, failure points, side effects, and parallel vs sequential behavior
   - `design/ownership-map.md`
     - purpose: source-of-truth ownership, allowed dependency direction, and responsibility boundaries

6. Define conditional design-bundle artifacts by trigger.
   - `design/data-model.md`
     - create when the task changes persisted state, schema, cache contract, projections, replay behavior, or migration shape
   - `design/contracts/`
     - create when the task changes API contracts, event contracts, generated contracts, or material internal interfaces between subsystems
   - `design/dependency-graph.md`
     - create when the task changes module/package dependency shape, generated-code dependency flow, or introduces a coupling risk that must be made explicit
   - `rollout.md`
     - create when the task needs migration sequencing, backfill/verify/contract choreography, mixed-version compatibility, or explicit deploy/failback notes
   - `test-plan.md`
     - create when validation obligations are too large or multi-layered to fit cleanly inside `plan.md`

7. Keep artifact responsibilities sharply separated.
   - `workflow-plan.md`
     - owns routing, current stage, next action, blockers, and resume control
   - `spec.md`
     - owns approved problem framing, scope, constraints, decisions, and accepted open questions
   - `design/`
     - owns task-specific technical design context
   - `plan.md`
     - owns phased implementation order, checkpoints, change surface hints, and proof obligations
   - `research/*.md`
     - owns preserved evidence and comparisons, not final decisions

8. Upgrade `workflow-plan.md` from a pre-research routing note into a live resume/control artifact.
   - It should answer:
     - what stage the task is in now
     - which artifacts are approved vs draft vs missing
     - what the next action is
     - what is blocked
     - what can run in parallel
   - This makes cross-session resume deterministic instead of chat-history-dependent.

9. Standardize resume order for later sessions.
   - Orchestrator resume read order:
     1. `workflow-plan.md`
     2. `spec.md`
     3. `design/overview.md`
     4. required design artifacts for the task
     5. `plan.md`
     6. optional `test-plan.md`, `rollout.md`, and selected `research/*.md`
   - Stage inference should come from artifacts, not from memory:
     - no approved `spec.md` -> framing/specification not complete
     - approved `spec.md` but no approved design bundle -> technical design not complete
     - approved design bundle but no approved `plan.md` -> planning not complete
     - approved `plan.md` -> implementation-ready

10. Align skill ownership to the new artifact model.
    - `spec-document-designer`
      - remains the owner of `spec.md` authoring and normalization
    - `go-design-spec`
      - should be repurposed from “final spec assembly” toward integrated technical-design-bundle assembly and reconciliation
    - `planning-and-task-breakdown`
      - should plan from `spec.md + design/` rather than from `spec.md` alone for non-trivial work
    - No new skill name is required immediately; adapt existing skill boundaries first and revisit only if that becomes awkward in practice.

11. Do not create standalone artifact sprawl by default outside the named bundle.
    - The repository will support multiple technical artifacts, but each file must answer one class of question.
    - `design/overview.md` links to the bundle instead of repeating it.
    - `spec.md` should not absorb technical design.
    - `plan.md` should not absorb architecture reconstruction.
    - Conditional files should be created only when their trigger is met.

12. Treat this richer model as the default for non-trivial work, not for all work.
    - Direct-path and tiny changes may still skip the design bundle when:
      - the change is local
      - the behavior delta is obvious
      - no ownership/data/sequence ambiguity exists
      - the orchestrator records an explicit skip rationale

13. Add explicit session-bounded phases for non-trivial work.
    - Default model:
      - `specification session` = workflow planning + research + synthesis + specification
      - `technical design`
      - `planning`
      - `implementation:<plan phase>`
      - optional dedicated `review` or `validation` sessions only when the workflow plan or implementation plan explicitly says so
    - Policy:
      - one session owns one session-bounded phase
      - when that phase reaches its completion marker, the owning artifact plus `workflow-plan.md` must be updated and the session should stop
      - the next phase begins in a new session
    - Applicability:
      - mandatory for `full orchestrated` work and for non-trivial work that uses separate `spec.md`, `design/`, or `plan.md` artifacts
      - default-but-waivable for `lightweight local` work when the waiver is recorded up front
      - not required for tiny or `direct path` work
    - `workflow-plan.md` must record:
      - current session scope
      - phase status
      - completion marker
      - whether the session boundary has been reached
      - whether the task is ready for the next session
      - which phase the next session starts with
      - the stop rule that prevents casual same-session phase hopping

## Open Questions / Assumptions

- [assumption] Adapting `go-design-spec` is a lower-risk first move than inventing a brand-new design-bundle skill name and updating all discoverability surfaces at once.
- [assumption] `dependency-graph.md` should stay conditional rather than mandatory, because many non-trivial tasks change behavior without materially changing dependency shape.
- [open question] Whether the repository should later add a lightweight design-bundle example/template under `docs/` or keep all artifact guidance inside workflow docs and skills.
- [open question] Whether `docs/repo-architecture.md` should include one canonical request-flow sequence, one startup/shutdown sequence, and one async/background-flow sequence by default, or stay purely boundary/ownership oriented.
- [open question] A future direction around one-session-per-phase orchestrator skills is preserved in [`research/session-phase-skills-direction.md`](research/session-phase-skills-direction.md); it is not part of the approved workflow contract yet.

## Plan Summary / Link

Execution of the repository changes will follow [`plan.md`](plan.md) and [`session-prompts.md`](session-prompts.md).

Control summary:
1. Update the core workflow contract docs to recognize the new `technical design` stage and design bundle.
2. Add the repository-wide stable baseline architecture document.
3. Align planning and design skills with the new artifact model.
4. Add explicit session-bounded phase control and resume-stop semantics to the workflow contract and supporting skills.
5. Update discoverability surfaces so future sessions load the right mental model.
6. Keep implementation staged so contract changes land before skill/discoverability cleanup.

## Validation

This target architecture is grounded in:
- repository workflow and artifact rules in [AGENTS.md](/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md)
- repository workflow mechanics in [docs/spec-first-workflow.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md)
- repository structure and boundaries in [docs/project-structure-and-module-organization.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/project-structure-and-module-organization.md)
- repository command, config, and CI constraint docs:
  - [docs/build-test-and-development-commands.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/build-test-and-development-commands.md)
  - [docs/configuration-source-policy.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/configuration-source-policy.md)
  - [docs/ci-cd-production-ready.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/ci-cd-production-ready.md)
- preserved external pattern research in [specs/orchestrator-spec-design-skill/research/external-spec-patterns.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/orchestrator-spec-design-skill/research/external-spec-patterns.md)

Borrowed patterns:
- explicit artifact chain and independently testable slices from GitHub Spec Kit
- separate architecture/implementation-readiness layer from BMAD
- design-before-plan and anti-placeholder discipline from Superpowers
- early proof/validation thinking from Spec-Driven Workflow

Adapted patterns:
- multi-artifact technical design is adapted into a repository-native design bundle
- architecture context is split into repo-wide baseline vs task-local bundle
- planning remains a separate artifact instead of being merged into design

Own synthesis:
- `docs/repo-architecture.md` as the stable baseline
- `design/` as the task-local technical context bundle
- `workflow-plan.md` as the live resume/control artifact

Executed follow-up verification for the session-boundary pass:
- `rtk rg -n "Session boundary reached|Ready for next session|current session scope|session-bounded|stop rule|one session|new session|phase-scoped" AGENTS.md docs/spec-first-workflow.md README.md specs/artifact-driven-workflow-redesign/spec.md specs/artifact-driven-workflow-redesign/plan.md specs/artifact-driven-workflow-redesign/workflow-plan.md specs/artifact-driven-workflow-redesign/session-prompts.md .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md`
- `rtk rg -n "same session|next session|handoff boundary|phase-collapse waiver|waiver" .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md AGENTS.md docs/spec-first-workflow.md`
- `rtk git diff --check -- AGENTS.md docs/spec-first-workflow.md README.md specs/artifact-driven-workflow-redesign/spec.md specs/artifact-driven-workflow-redesign/plan.md specs/artifact-driven-workflow-redesign/workflow-plan.md specs/artifact-driven-workflow-redesign/session-prompts.md .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md .claude/skills/spec-document-designer/SKILL.md .claude/skills/go-design-spec/SKILL.md .claude/skills/planning-and-task-breakdown/SKILL.md .cursor/skills/spec-document-designer/SKILL.md .cursor/skills/go-design-spec/SKILL.md .cursor/skills/planning-and-task-breakdown/SKILL.md .gemini/skills/spec-document-designer/SKILL.md .gemini/skills/go-design-spec/SKILL.md .gemini/skills/planning-and-task-breakdown/SKILL.md .github/skills/spec-document-designer/SKILL.md .github/skills/go-design-spec/SKILL.md .github/skills/planning-and-task-breakdown/SKILL.md .opencode/skills/spec-document-designer/SKILL.md .opencode/skills/go-design-spec/SKILL.md .opencode/skills/planning-and-task-breakdown/SKILL.md`

## Outcome

Approved target workflow architecture:
- repository-wide stable baseline: `docs/repo-architecture.md`
- non-trivial task chain: `workflow-plan.md -> spec.md -> design/ -> plan.md -> implementation`
- non-trivial session chain by default:
  - `specification session`
  - `technical design`
  - `planning`
  - `implementation:<plan phase>`
- required design-bundle core:
  - `design/overview.md`
  - `design/component-map.md`
  - `design/sequence.md`
  - `design/ownership-map.md`
- conditional expansions:
  - `design/data-model.md`
  - `design/dependency-graph.md`
  - `design/contracts/`
  - `test-plan.md`
  - `rollout.md`
- `workflow-plan.md` now also needs explicit session control:
  - current session scope
  - phase status
  - completion marker
  - `Session boundary reached`
  - `Ready for next session`
  - `Next session starts with`
  - `Stop rule`

This redesign now covers both artifact boundaries and session boundaries so future sessions do not have to re-decide when to stop, what marks a phase complete, or how to resume the next phase safely.
