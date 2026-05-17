# External Agent Workflow Practices

Accessed: 2026-05-17

## Scope

This note extracts concrete workflow patterns from current LLM-agent workflow systems that are relevant to simplifying `go-service-template-rest`. It does not treat external systems as authority for this repository.

## Source Map

| System | Source | Concrete Workflow Facts |
| --- | --- | --- |
| BMAD Method | [Workflow Map](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/docs/reference/workflow-map.md), [Getting Started](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/docs/tutorials/getting-started.md) | Uses phases for analysis, planning, solutioning, and implementation. Produces PRD, architecture, epics/stories, sprint status, story files, and review outputs. Provides `bmad-quick-dev` to skip larger phases for small, well-understood work. Uses document handoff between phases and an implementation-readiness check. |
| BMAD project context | [Project Context](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/docs/explanation/project-context.md) | Uses a project-level implementation guide/constitution-style document to keep agent decisions aligned with project rules and preferences across workflows. |
| Superpowers | [README](https://github.com/obra/superpowers), [executing-plans skill](https://github.com/obra/superpowers/blob/main/skills/executing-plans/SKILL.md) | Uses composable skills that trigger around brainstorm, worktree isolation, plan writing, plan execution, TDD, code review, and branch finishing. Plans are expected to be detailed enough for low-context execution. Execution loads and reviews the plan first, executes bite-sized tasks, runs specified verification, and stops on blockers. |
| GSD | [README](https://github.com/gsd-build/get-shit-done), [Commands](https://github.com/gsd-build/get-shit-done/blob/main/docs/COMMANDS.md), [User Guide](https://github.com/gsd-build/get-shit-done/blob/main/docs/USER-GUIDE.md) | Uses discuss -> plan -> execute -> verify -> ship. Keeps persistent state in `.planning` artifacts such as project, requirements, roadmap, state, and phase context. Runs fresh contexts for heavy work, supports wave-based parallel execution, provides quick-task flags (`--full`, `--validate`, `--discuss`, `--research`), and has state validation/sync commands. |
| GitHub Spec Kit | [README](https://github.com/github/spec-kit) | Uses constitution -> specify -> plan -> tasks -> implement, with optional clarify/analyze/checklist commands. Keeps project principles in `.specify/memory/constitution.md`, creates feature specs, implementation plans, task lists, and validates prerequisites before implementation. |
| Kiro Quick Plan | [Quick Plan](https://kiro.dev/docs/specs/quick-plan) | Auto-generates requirements, design, and tasks in one pass for well-understood features, without approval gates between phases. It still saves normal spec artifacts and recommends standard specs for unfamiliar or high-stakes work. |
| OpenSpec | [Concepts](https://github.com/Fission-AI/OpenSpec/blob/main/docs/concepts.md) | Emphasizes progressive rigor and delta specs for brownfield development, so change folders describe added, modified, or removed behavior instead of forcing agents to mentally diff full specs. |
| Task Master | [PRD creation and parsing](https://docs.task-master.dev/getting-started/quick-start/prd-quick), [RPG Method](https://docs.task-master.dev/capabilities/rpg-method) | Starts from a focused PRD, parses it into structured tasks with dependencies, priorities, and test strategies. The RPG template is optional for complex work and emphasizes functional vs structural decomposition, explicit dependency graphs, topological order, progressive refinement, and complexity-based expansion. |
| Warp planning | [Agent planning and execution](https://docs.warp.dev/agent-platform/capabilities/planning/) | Uses persistent editable plans with version history, selective execution, real-time progress monitoring, export/share options, and reuse across conversations. Planning is a first-class UI object rather than a directory of separate phase files. |

## Patterns That Reduce Overhead Without Dropping Quality

### 1. Phase Compression, Not Phase Deletion

BMAD, GSD, and Kiro all keep the same basic concerns: clarify, plan, execute, verify. The simplification comes from quick tracks, flags, and one-pass modes, not pretending those concerns disappear. BMAD has `bmad-quick-dev` for small work. GSD has `/gsd-quick` with flags that enable only the quality agents needed for the task. Kiro Quick Plan collapses requirements, design, and tasks into one pass for well-understood work while still producing artifacts.

Implication for this repo: keep the concerns, but make most gates trigger-based. The workflow should ask, "Which quality risk exists here?" before creating another file or phase. The default bounded non-trivial path should be `lean local`; `full orchestrated` should require a named risk trigger.

### 2. One Durable State Surface Beats Many Routing Files

GSD keeps a central `STATE.md` and supporting planning artifacts. Warp stores an editable, versioned plan that can be reopened across conversations. Spec Kit keeps a durable constitution plus feature-level spec/plan/tasks. BMAD's `project-context.md` keeps implementation rules available across workflows. None of these require a separate routing file for every phase by default.

Implication for this repo: `workflow-plan.md` can remain the compatible live state/control surface when multi-session state is real. `workflow-plans/<phase>.md` should be conditional when a phase has multi-lane orchestration, multi-session routing, or a formal challenge gate that needs its own record. For `lean local`, `spec.md` status plus `tasks.md` progress should usually be enough.

### 3. Executable Task Slices Are A Strong Quality Carrier

Superpowers emphasizes plans with exact file paths, task-sized steps, and verification steps. Spec Kit, Kiro, and Task Master translate requirements into task artifacts. GSD execution produces summaries and verification artifacts per phase.

Implication for this repo: `tasks.md` should become the main working artifact for `lean local` implementation. The simplification target should be earlier ceremony, not the executable handoff. Only tiny `direct path` work should routinely skip it.

### 4. Context Freshness Is Handled By State Plus Bounded Context, Not Larger Documents

GSD explicitly optimizes against context bloat through fresh subagent contexts and persistent artifacts. Superpowers dispatches low-context plan tasks and reviews them. Warp makes the plan reusable as context across conversations.

Implication for this repo: preserve concise state and handoff context, but avoid duplicating status across master and phase files. Smaller state surfaces are more likely to be read and followed by modern agents.

### 5. Quality Gates Are Often Configurable Or Mode-Dependent

GSD exposes toggles for research, plan checking, verifier, model profile, and parallelization. Spec Kit makes clarify/analyze/checklist optional enhancements around the core flow. Kiro uses Quick Plan for well-understood work and standard Feature Specs where review gates add value. Task Master uses the more structured RPG method when complexity warrants it, not for every task.

Implication for this repo: subagent challenge gates, split design files, test plans, rollout plans, and preserved research should be triggered by risk signals. Inline `Risk Challenge` can cover lean local work when the cost of a separate agent or artifact exceeds the risk.

### 6. Tooling Reduces Process Load

GSD has commands to validate and sync state. Warp gives a plan editor, version history, selective execution, and progress view. Spec Kit has CLI prerequisites and generated command templates.

Implication for this repo: future implementation should encode simplification in templates, checks, and skills, not only prose. The repo already has `rtk make agents-check`, `rtk make skills-check`, and `rtk git diff --check` as implementation-time proof gates.

### 7. Model Capability Replaces Ceremony Only After The Risk Is Bounded

The compared systems rely on agent capability in specific, bounded places:

- BMAD Quick Flow lets small work skip larger planning phases, but still produces a spec and code.
- Superpowers asks the agent to review a plan critically before execution and then follow bite-sized tasks with verification.
- GSD quick flags can skip discussion or research, but the user can re-enable plan checking, verifier, or research per task.
- Spec Kit leaves `clarify`, `analyze`, and `checklist` optional around the core specify/plan/tasks/implement path.
- Kiro Quick Plan relies on the model to generate requirements, design, and tasks without per-phase approval gates only for well-understood features.
- OpenSpec relies on delta specs to reduce brownfield specification load instead of rewriting full behavior descriptions for each change.
- Task Master uses the heavier RPG template only when dependency and module complexity justify it.
- Warp lets the user edit, version, and selectively execute a plan instead of forcing separate planning documents.

Interpretation for this repo: model capability should replace ceremony only when the workflow has already classified the task as `direct path` or `lean local` and no escalation trigger is present. The model can merge research/spec/design/planning locally for bounded work, but it must still leave a durable decision and proof trail through `spec.md` and `tasks.md` when implementation is non-trivial.

### 8. Delta Specs Fit Brownfield Services

OpenSpec's delta-spec framing is especially relevant because this repository is a service template used to evolve existing Go services. A compact `ADDED / MODIFIED / REMOVED` section makes agents describe the change, not re-document the whole service. That reduces context load while preserving behavior and contract clarity.

Implication for this repo: the lean `spec.md` shape should include `Behavior / Contract Delta` by default. Full orchestrated work may still need richer contract, design, or rollout artifacts, but brownfield lean work should start from deltas.

## External Anti-Patterns To Avoid

- Do not replace workflow discipline with "trust the agent." All compared systems still preserve some durable plan or task state.
- Do not require every quality mechanism for every task. External systems increasingly expose quick paths, flags, or optional checks.
- Do not make generated planning artifacts stale. GSD's state validation/sync pattern is a useful warning: fewer files reduce the drift surface.
- Do not hide package, security, or dependency-risk decisions inside execution. GSD's package checkpoint behavior shows that some risk classes still need explicit human or verifier checkpoints.
- Do not turn a project-context or constitution card into another large always-loaded manual. Keep it short enough to actually be read by agents.

## Fit For This Repository

The best fit is not BMAD's large PRD/architecture/story stack and not GSD's large `.planning` operating system. This repository already has the right quality concepts. It needs a trigger-based artifact model:

- direct path: no persistent bundle unless needed
- lean local: `spec.md` plus `tasks.md` for non-trivial implementation, with optional `research.md` or one `design.md` only when triggered
- full orchestrated: keep preserved workflow state, research, challenges, design, tasks, and validation gates
