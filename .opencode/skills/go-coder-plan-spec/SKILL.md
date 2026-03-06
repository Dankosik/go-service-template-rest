---
name: go-coder-plan-spec
description: "Create execution-grade coding plans from approved requirements: atomic tasks, traceability, checkpoints, verification evidence, and clarification triggers while preserving coder autonomy."
---

# Go Coder Plan Spec

## Purpose
Turn approved requirements and constraints into an execution-grade coding plan that is atomic, traceable, check-pointed, and testable without over-prescribing implementation mechanics.

## Scope
- turn approved requirements, invariants, and constraints into atomic coding tasks
- define dependency-aware sequencing and change surface
- define checkpoints, progress tracking, and clarification triggers
- define verification commands and expected evidence for each task
- preserve coder autonomy in technical decomposition and low-level implementation details

## Boundaries
Do not:
- write production code or tests as the primary output
- rewrite approved architecture, contract, security, or reliability decisions
- replace specialist-domain ownership with local planning guesses
- force exact file paths or exact low-level coding technique unless a real constraint requires it

## Core Defaults
- Plan by outcomes and dependencies, not by coding style.
- Prefer the smallest safe tasks with explicit evidence over broad narrative work packages.
- Treat blocked ambiguity as a clarification trigger, not as permission to invent local decisions.
- Preserve coder freedom to refactor locally, split files, or adjust technical structure while keeping approved intent intact.

## Expertise

### Planning Preconditions
- Start from approved or otherwise stable intent.
- Surface blocker-level contradictions before decomposing work into tasks.
- If the plan depends on unresolved architecture, contract, security, or reliability questions, block the affected tasks instead of guessing.

### Atomic Task Design
- Each task should change one observable behavior or one tightly coupled behavior set.
- Task size should allow meaningful verification without repository-wide execution whenever possible.
- Avoid mega-tasks that mix unrelated domains or force broad retesting too early.

### Traceability Closure
For each task, maintain explicit links to:
- requirements or decisions
- invariants and acceptance behavior
- test obligations

The plan should support a clean closure chain:
- requirement or decision -> task -> verification command -> expected evidence

### Change Surface Without Path Lock-In
- Define expected change surface by module, layer, or package area.
- Do not force exact file lists as mandatory.
- Allow the coder to create, move, split, or consolidate files when it improves implementation quality and stays within approved intent.

### Outcome-Oriented Sequence
- Define sequencing by dependencies and outcomes.
- Do not prescribe function extraction order, exact internal naming, or micro-level implementation sequence unless those details are themselves constraints.

### Verification And Evidence
Each task should include:
- smallest sufficient verification commands
- expected evidence for `done` vs `not done`

For behavior-changing work, make before-and-after observable checks explicit.

### Checkpoints
- Insert checkpoints after small task groups, usually every `2-4` tasks.
- Each checkpoint should define what the later execution or review stage must confirm:
  - required checks completed for the group
  - alignment with approved intent
  - progress against critical obligations
  - evidence quality
  - reconciliation of actual touched modules against expected change surface
  - go/no-go for the next group

### Clarification Contract
For blocked tasks, capture:
- `request_id`
- `blocked_task_id`
- `ambiguity_type`
- `conflicting_sources`
- `decision_impact`
- `proposed_options`
- `owner`
- `resume_condition`

Blocked tasks should not continue until their resume condition is met.

### Coder Autonomy
The plan should explicitly preserve coder autonomy:
- the coder chooses technical decomposition and low-level implementation order
- the coder chooses local refactoring shape within approved intent
- the plan governs outcomes, constraints, checkpoints, and evidence, not coding style internals

### Review-Ready Task Cards
Each task should carry a compact review checklist:
- intended behavior reached
- no contract or invariant drift
- verification evidence collected
- unresolved ambiguity status explicit

### Execution Modes
The plan should support both:
- in-session execution
- batch execution

Whichever mode is used, checkpoint discipline must stay explicit.

## Plan Quality Bar
A good coding plan:
- covers all critical approved obligations
- uses atomic tasks with meaningful verification
- preserves coder autonomy
- makes blockers and clarifications explicit
- avoids contradictions with the approved design or requirements

## Deliverable Shape
Return the plan in a compact, execution-ready form:
- `Execution Context`
- `Execution Mode`
- `Task Graph`
- `Task Cards`
- `Checkpoint Plan`
- `Clarification Contract`
- `Coverage Matrix`
- `Execution Notes` when truly needed

A strong task card usually includes:
- `Task ID`
- `Objective`
- `Traceability`
- `Change Surface`
- `Task Sequence`
- `Verification Commands`
- `Expected Evidence`
- `Review Checklist`
- `Ambiguity Triggers`
- `Change Reconciliation`
- `Progress Status`

## Escalate When
Escalate if:
- approved intent is unstable or contradictory
- blocked questions would materially change architecture, contract, security, reliability, or invariants
- the work cannot be decomposed into atomic tasks with meaningful verification
- the plan would need exact file-path lock-in or low-level code prescriptions just to remain coherent
