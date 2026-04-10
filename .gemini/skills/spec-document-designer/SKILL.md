---
name: spec-document-designer
description: "Design and normalize repository-native `spec.md` documents for this workflow. Use when the orchestrator has a framed change or synthesized research and needs to turn it into a stable `spec.md` with the right section depth, decision placement, audit trail, and plan handoff before `planning-and-task-breakdown`. Skip raw ideation, specialist domain design, full task breakdown, and implementation coding."
---

# Spec Document Designer

## Purpose
Turn a framed request or synthesized design work into a planning-ready repository-native `spec.md` without turning it into a PRD, a task list, or a research dump.

## Scope
- draft a fresh `spec.md` when the task is mature enough for specification
- normalize existing drafts that are too thin, too bloated, or shaped like a foreign template
- choose the right section depth for the task while staying inside the repository's artifact model
- translate useful coverage prompts from external spec workflows into repo-native sections
- keep planning blockers, assumptions, and validation hooks visible before handoff to `planning-and-task-breakdown`

## Boundaries
Do not:
- refine a raw product idea; use `idea-refine`
- perform engineering framing on an under-shaped request; use `spec-first-brainstorming`
- absorb unresolved cross-domain design contradictions that belong in `go-design-spec` or specialist `*-spec` skills
- produce task breakdown, execution sequencing, or coder instructions; that belongs to `planning-and-task-breakdown`
- copy BMAD, Spec Kit, Superpowers, or SDD templates directly into this repository's `spec.md`

## Escalate When
Escalate if:
- the request is still idea-shaped, solution-led, or missing its behavior delta
- current external guidance materially affects the design and has not been researched yet
- the draft still contains unresolved domain contradictions that would make `spec.md` dishonest
- the work is tiny enough that a separate spec pass would be ceremony instead of risk reduction
- planning would still need to reopen core design after the spec pass

## Core Defaults
- `spec.md` is the canonical decisions artifact.
- Use the repository's default section set unless merging sections makes the file clearer.
- Treat external frameworks as coverage prompts, not as headings to copy.
- Put execution detail in `plan.md`, preserved evidence in `research/*.md`, and stable decisions in `spec.md`.
- Prefer short explicit bullets over template sludge.
- Omit empty sections instead of padding the document for completeness theater.

## Context Intake (Dynamic Loading)

Rule: load the smallest sufficient set of artifacts. Do not bulk-load folders by default.

Always load:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `references/spec-patterns.md`

Load by trigger:
- existing spec rewrite or continuation: the active `spec.md`
- non-trivial work: the matching `workflow-plan.md`
- research-backed synthesis: the relevant `research/*.md`
- planning drift or handoff check: the matching `plan.md`
- conflicting specialist notes: the smallest set of affected design artifacts

Conflict resolution:
- repository contract beats the reference file
- the more specific artifact beats generic guidance
- if a foreign template conflicts with repo artifact ownership, keep the repo model and translate the useful idea instead

Unknowns:
- if critical facts are missing, mark them as `[assumption]` or escalate

## Hard Skills

### Mission
Make `spec.md` stable enough for planning while preserving the repository's single-source-of-truth discipline.

### Default Posture
- decision-first
- artifact-disciplined
- minimum sufficient structure
- no placeholder tolerance

### Spec Shape Competency
- Start from the repository's default sections:
  - `Context`
  - `Scope / Non-goals`
  - `Constraints`
  - `Decisions`
  - `Open Questions / Assumptions`
  - `Plan Summary / Link`
  - `Validation`
  - `Outcome`
- Merge sections only when it makes the document clearer.
- Expand depth based on task risk and ambiguity, not on habit.

### Coverage Competency
- Capture the behavior delta, affected actors, scope cuts, edge semantics, validation expectations, and material constraints.
- Use these external patterns only as coverage lenses:
  - independently testable slices
  - user journeys or demoable units
  - measurable success criteria
  - selective NFRs
  - proof artifacts
  - repository standards
  - rejected alternatives when they materially affect planning
- If a pattern adds no execution value for this task, leave it out.

### Artifact Ownership Competency
- Stable decisions belong in `Decisions`.
- Evidence history, comparisons, and raw external research belong in `research/*.md`.
- Task sequencing and execution detail belong in `plan.md`.
- Unresolved but visible gaps belong in `Open Questions / Assumptions`.
- Do not force the planner to recover execution order from `spec.md` when a separate `plan.md` is warranted.

### Planning Readiness Competency
- A planning-ready spec must let `planning-and-task-breakdown` derive phases without silently reopening core design.
- Keep blockers, accepted risks, and reopen conditions explicit.
- Preserve non-goals and scope cuts so planning does not re-expand the change.

### Spec Review Competency
- Scan for placeholders, `TODO`, `TBD`, contradictions, duplicated content, scope spread, implementation leakage, and research dumped into `Decisions`.
- Remove decorative sections that do not help the reader or the planner.
- Rescue soft but material constraints that often get lost, such as operator expectations, UX promises, or policy language.

### Evidence Threshold
- Important claims should rest on repository evidence or linked research.
- If current external guidance could change the design and is missing, reopen research instead of writing a confident fiction.

### Review Blockers For This Skill
- idea-shaped input
- unresolved specialist conflict
- task dump inside `spec.md`
- raw research inside `Decisions`
- medium/high-risk work with no visible non-goals or validation story

## Workflow

### 1. Confirm The Handoff Boundary
- Check whether the task is mature enough for spec design.
- If the real problem is still framing, send it back upstream instead of pretending a spec exists.

### 2. Load The Minimum Authoritative Context
- Read the active repository contract first.
- Load only the artifacts that materially affect the current spec pass.
- Avoid broad repository tours unless the spec truly depends on them.

### 3. Choose The Spec Shape
- Start with the default section set.
- Decide how much depth each section needs for this task.
- If a foreign framework suggests extra sections, translate the useful concern into an existing repo section or a linked artifact.

### 4. Build A Coverage Pass
- Use `references/spec-patterns.md` to ask which concerns matter here:
  - user-visible slices
  - edge cases
  - key entities or state
  - scope ladder
  - relevant NFRs
  - validation/proof expectations
  - repository constraints
- Keep only the concerns that actually sharpen planning.

### 5. Write Or Repair `spec.md`
- Place each fact into the correct artifact.
- Keep `Decisions` authoritative and compact.
- Link out instead of duplicating detail when preserved evidence already exists elsewhere.

### 6. Run A Planning-Readiness Review
- Ask whether planning can proceed without reopening the design.
- If yes, finalize the spec and keep the plan handoff explicit.
- If no, escalate to the missing upstream skill or specialist lane.

## Output Expectations
Return or write spec work using the repository-native section order:
- `Context`
- `Scope / Non-goals`
- `Constraints`
- `Decisions`
- `Open Questions / Assumptions`
- `Plan Summary / Link`
- `Validation`
- `Outcome`

Rules:
- Merge sections when clearer.
- Do not create empty sections.
- Do not dump full task lists or execution steps into `spec.md`.
- When blocked, say what upstream skill or research pass must reopen and why.

## Definition Of Done
The pass is complete when:
- the spec is honest about what is decided and what is not
- stable decisions are separated from raw evidence
- scope cuts and non-goals are explicit
- validation expectations are visible early enough for planning
- the next planning step is clear without turning the spec into a plan

## Anti-Patterns
- copying external template headings directly into the repo default shape
- turning `spec.md` into a PRD, audit report, or task board
- filling every possible NFR category whether it matters or not
- hiding contradictions under generic wording
- treating raw research notes as final decisions
- using this skill when framing, specialist design, or planning clearly owns the work
