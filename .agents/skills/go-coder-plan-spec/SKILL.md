---
name: go-coder-plan-spec
description: "Design an execution-grade coder plan (`65-coder-detailed-plan.md`) from an approved spec package in a spec-first workflow. Use after G2 when coding needs atomic, traceable, checkpointed tasks with explicit evidence, while preserving coder freedom in technical implementation details. Skip when writing production code, redesigning architecture, or performing domain code review."
---

# Go Coder Plan Spec

## Purpose
Create a deterministic, execution-grade plan for coding after spec sign-off.

Success means:
- `60-implementation-plan.md` is transformed into an atomic execution artifact for coding;
- every critical spec obligation is mapped to tasks and verification evidence;
- `go-coder` gets clear what/why/order constraints without losing freedom in how to implement code.

## Scope And Boundaries
In scope:
- produce `65-coder-detailed-plan.md` for the active feature;
- convert approved constraints from `15/20/30/40/50/55/60/70/90` into atomic tasks;
- define checkpoints, progress tracking, and clarification triggers;
- keep plan outcome-oriented and implementation-technique-agnostic.

Out of scope:
- writing production code or tests;
- changing approved architecture/contract/security/reliability decisions;
- replacing specialist-domain ownership from existing `*-spec` roles;
- enforcing exact file-path lock-in for coder execution.

## Hard Skills
### Mission
- Bridge strategic spec artifacts and real coding execution with minimal interpretation drift.
- Preserve approved intent while making execution observable, auditable, and checkpointed.
- Keep coder autonomy for technical realization details.

### Default Posture
- `60` remains strategic source of implementation intent.
- `65` is execution contract for coding.
- Plan by outcomes and dependencies, not by prescribing low-level code mechanics.
- Prefer smallest safe tasks with explicit evidence over broad narrative packages.

### Phase And Gate Competency
- This skill runs after `G2` and before coding starts.
- Target output is readiness for `G2.5` (`Detailed Plan Ready`).
- If blocker-level ambiguity appears, do not invent local decisions; open clarification and pause affected tasks.

### Atomic Task Design Competency
- Each task must change one observable behavior or one tightly coupled behavior set.
- Task size should allow meaningful verification without repo-wide execution whenever possible.
- Avoid mega-tasks that mix unrelated domains.

### Traceability Closure Competency
For each task, require explicit links to:
- decisions (`ARCH/DOM/REL/DATA/DBC/OBS/SEC/DOPS/TST/DES` as applicable);
- invariants and acceptance obligations;
- test obligations from `70`.

The plan must enable closure proof:
- `decision/invariant/obligation -> task(s) -> command(s) -> expected evidence`.

### Change Surface Competency (No Path Lock-In)
- Define expected change surface by module/layer/package area.
- Do not force exact file lists as mandatory.
- Allow coder to create/move/split files or perform local refactoring when it improves implementation quality and keeps approved intent.

### Task Sequence Competency (Outcome-Oriented)
- Define high-level sequencing by dependencies and outcomes only.
- Do not prescribe low-level coding technique, function extraction order, or exact internals.

### Verification And Evidence Competency
Each task must include:
- smallest sufficient verification commands;
- expected evidence (observable pass criteria).

Evidence must be concrete enough to decide `done` vs `not done` without reinterpretation.

For behavior-changing tasks:
- include explicit observable behavior checks (before/after or equivalent) in verification expectations.

### Checkpoint Competency
- Insert checkpoints after small task groups (default: every `2-4` tasks).
- Checkpoint definition must specify what the later execution/review stage must confirm:
  - required checks for the group are completed;
  - spec alignment;
  - decision coverage progress;
  - evidence quality;
  - actual touched files/modules are reconciled against declared `Change Surface` with a short justification for deviations;
  - go/no-go for next group.
- This skill defines checkpoint contracts only; it does not execute or validate code changes.

### Clarification Competency
Define mandatory clarification trigger contract for blocked tasks:
- `request_id`
- `blocked_task_id`
- `ambiguity_type` (`contract`, `invariant`, `security`, `reliability`, `test`, `other`)
- `conflicting_sources`
- `decision_impact`
- `proposed_options`
- `owner`
- `resume_condition`

No continuation on blocked task before resolution criteria are met.

### Coder Autonomy Competency
The plan must explicitly preserve coder autonomy:
- coder chooses technical decomposition and low-level implementation order;
- coder decides refactoring approach within approved intent and scope boundaries;
- plan governs outcomes/constraints/evidence, not coding style internals.

### Review-Ready Planning Competency
Every task should carry a compact review checklist for in-flight quality validation.

Checklist focus:
- intended behavior reached;
- no contract/invariant drift;
- verification evidence collected;
- unresolved ambiguity status explicit.

### Execution Mode Competency
The plan should support both execution handoff modes:
- in-session execution (task-by-task in one active session);
- batch execution (separate run with mandatory checkpoints).

The selected mode and checkpoint discipline must be explicit in the plan header.

## Working Rules
1. Confirm feature phase readiness: `G2` passed, `Spec Freeze` active, blocker-level open questions closed.
2. Load current feature artifacts in this order:
   - `60` (strategic plan)
   - `15/30/40/50/55` (constraints)
   - `70` (test obligations)
   - `90` (accepted decisions)
   - `80` (remaining uncertainties/reopen notes)
3. Build a closure map of critical obligations before writing tasks.
4. Split work into atomic tasks with explicit dependency order.
5. For each task, define change surface, sequence, verification commands, expected evidence, checklist, and progress field.
6. Add checkpoints after each small task group.
7. Define execution mode (`in-session` or `batch`) and keep checkpoint discipline explicit.
8. Add clarification triggers and schema for ambiguity handling.
9. Validate internal consistency: no task contradicts frozen decisions.
10. Output `65-coder-detailed-plan.md` as the execution artifact.

## Output Expectations
Primary artifact:
- `specs/<feature-id>/65-coder-detailed-plan.md`

Mandatory sections:
1. `Execution Context`
   - scope boundaries, non-goals, critical invariants, forbidden changes.
2. `Execution Mode`
   - `in-session` or `batch`, with checkpoint policy.
3. `Task Graph`
   - ordered atomic tasks with dependency edges.
4. `Task Cards`
   - `Task ID`
   - `Objective`
   - `Spec Traceability`
   - `Change Surface`
   - `Task Sequence`
   - `Verification Commands`
   - `Expected Evidence`
   - `Review Checklist`
   - `Ambiguity Triggers`
   - `Change Reconciliation` (actual touched files/modules and short deviation rationale when different from expected surface)
   - `Progress Status` (`todo/in_progress/done/blocked`)
5. `Checkpoint Plan`
   - checkpoint cadence and go/no-go criteria for the later execution/review stage.
6. `Clarification Contract`
   - required fields and resolution policy.
7. `Coverage Matrix`
   - obligation-to-task closure summary.
8. `Execution Notes`
   - optional operational notes only when required by approved spec decisions.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of sources. Do not bulk-load unrelated docs.

Always load:
- `docs/spec-first-workflow.md` (G2, freeze, Phase 3 expectations)
- `specs/<feature-id>/60-implementation-plan.md`
- `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- `specs/<feature-id>/30-api-contract.md`
- `specs/<feature-id>/40-data-consistency-cache.md`
- `specs/<feature-id>/50-security-observability-devops.md`
- `specs/<feature-id>/55-reliability-and-resilience.md`
- `specs/<feature-id>/70-test-plan.md`
- `specs/<feature-id>/80-open-questions.md`
- `specs/<feature-id>/90-signoff.md`

Load additional docs only when needed to resolve a concrete planning ambiguity.

## Definition Of Done
- `65-coder-detailed-plan.md` exists and is complete.
- All critical approved obligations are mapped to at least one task with evidence.
- Tasks are atomic, ordered by dependencies, and checkpointed.
- Plan preserves coder technical autonomy and avoids low-level coding prescriptions.
- Clarification triggers and schema are explicitly included.
- No contradictions with approved frozen spec decisions.
- Checkpoints are defined as declarative validation contracts; no claim of code execution/review completion is made by this skill.

## Anti-Patterns
- Rewriting architecture during execution planning.
- Copying `60` narrative as-is without atomic task decomposition.
- Forcing exact file paths as hard constraints.
- Describing low-level coding internals instead of outcome/dependency sequence.
- Missing evidence expectations for task completion.
- Missing checkpoint policy.
- Ignoring `70` obligations when building coder tasks.
- Silent ambiguity handling without clarification triggers.
