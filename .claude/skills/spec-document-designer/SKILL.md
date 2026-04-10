---
name: spec-document-designer
description: "Design and normalize repository-native `spec.md` documents for this workflow. Use when the orchestrator has a framed change or synthesized research and needs to turn it into a stable `spec.md` with the right section depth, decision placement, clarification-gate reconciliation, audit trail, and technical-design handoff before non-trivial planning. Skip raw ideation, technical-design-bundle assembly, full task breakdown, and implementation coding."
---

# Spec Document Designer

## Purpose
Turn a framed request or synthesized research into a repository-native `spec.md` that is honest, stable, and ready to hand off into `technical design` without turning it into a PRD, a design bundle, or a task list.

## Scope
- draft a fresh `spec.md` when the task is mature enough for specification
- normalize existing drafts that are too thin, too bloated, or shaped like a foreign template
- choose the right section depth for the task while staying inside the repository's artifact model
- translate useful coverage prompts from external spec workflows into repo-native sections
- keep blockers, assumptions, clarification outcomes, validation hooks, and plan-summary links visible before handoff to `technical design`

## Boundaries
Do not:
- refine a raw product idea; use `idea-refine`
- perform engineering framing on an under-shaped request; use `spec-first-brainstorming`
- absorb unresolved cross-domain design contradictions that belong in `go-design-spec` or specialist `*-spec` skills
- assemble the task-local `design/` bundle; that belongs to `go-design-spec`
- produce task breakdown, execution sequencing, or coder instructions; that belongs to `planning-and-task-breakdown`
- silently skip `technical design` for non-trivial work by smuggling design detail into `spec.md`
- mark non-trivial `spec.md` approved while the autonomous `spec-clarification-challenge` gate is unresolved or blocked; if the gate is blocked, leave the spec unapproved with a reopen target
- copy BMAD, Spec Kit, Superpowers, or SDD templates directly into this repository's `spec.md`

## Escalate When
Escalate if:
- the request is still idea-shaped, solution-led, or missing its behavior delta
- current external guidance materially affects the design and has not been researched yet
- the draft still contains unresolved domain contradictions that would make `spec.md` dishonest
- the clarification challenge returns `blocks_spec_approval`, `blocks_specific_domain`, or `requires_user_decision` items that the orchestrator has not reconciled
- the work is tiny enough that a separate spec pass would be ceremony instead of risk reduction
- non-trivial work still lacks a stable decisions record that `go-design-spec` can carry into `design/` without reopening core framing

## Core Defaults
- `spec.md` is the canonical decisions artifact.
- For non-trivial work, the handoff path is `spec.md -> design/ -> plan.md`.
- For non-trivial work, `spec.md` approval requires the autonomous `spec-clarification-challenge` gate before handoff to `technical design`.
- For non-trivial work, this pass ends the current session at approved `spec.md`; `technical design` begins in a new session unless an upfront `direct path` or `lightweight local` waiver was already recorded.
- Use the repository's default section set unless merging sections makes the file clearer.
- Treat external frameworks as coverage prompts, not as headings to copy.
- Put stable decisions in `spec.md`, task-local technical context in `design/`, execution detail in `plan.md`, and preserved evidence in `research/*.md`.
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
- non-trivial spec approval: `.agents/skills/spec-clarification-challenge/SKILL.md`
- handoff drift check: the matching `plan.md`
- existing technical-design bundle nearby: `design/overview.md` and only the smallest set of affected design artifacts needed to confirm ownership boundaries, not to author design in this pass

Conflict resolution:
- repository contract beats the reference file
- the more specific artifact beats generic guidance
- if a foreign template conflicts with repo artifact ownership, keep the repo model and translate the useful idea instead

Unknowns:
- if critical facts are missing, mark them as `[assumption]` or escalate

## Hard Skills

### Mission
Make `spec.md` stable enough for `technical design` while preserving the repository's single-source-of-truth discipline.

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
- Task-local technical design context belongs in `design/`.
- Task sequencing and execution detail belong in `plan.md`.
- Unresolved but visible gaps belong in `Open Questions / Assumptions`.
- Do not force `go-design-spec` or the planner to recover ownership, sequence, or execution order from `spec.md` when separate artifacts are warranted.

### Technical-Design Handoff Competency
- A non-trivial spec must let `go-design-spec` derive the task-local `design/` bundle without silently reopening core problem framing.
- Keep blockers, accepted risks, and reopen conditions explicit.
- Preserve non-goals and scope cuts so technical design does not re-expand the change.
- Keep only the planning summary or plan link in `spec.md` when a separate `plan.md` will exist.

### Clarification-Gate Competency
- Before approving non-trivial `spec.md`, ensure the orchestrator has run a read-only `spec-clarification-challenge` lane, preferably through `challenger-agent`, using exactly that one skill.
- The challenge returns questions for orchestrator reconciliation; it does not write files or make final decisions.
- Resolve each planning-critical item from existing evidence, targeted research, an expert subagent lane, explicit risk acceptance, design deferral, or `requires_user_decision`.
- Store only final resolved outcomes in `spec.md`: stable outcomes in `Decisions`, remaining assumptions in `Open Questions / Assumptions`, and proof consequences in `Validation`.
- Do not copy raw clarification transcripts into `spec.md`.

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

### 6. Run Or Enforce The Clarification Gate
- For non-trivial work, confirm the `spec-clarification-challenge` gate ran after candidate decisions existed and before approval.
- If the gate has not run, prepare the compact bundle and route one read-only subagent lane, preferably `challenger-agent`, using only `spec-clarification-challenge`.
- If the gate returns material questions, keep `spec.md` draft or blocked until the orchestrator reconciles them.
- If targeted expert research is required, route the appropriate upstream research or expert lane instead of inventing an answer in the spec.
- If a question is truly external product or business policy, record `requires_user_decision` and leave the spec blocked or partially draft.
- If material decisions changed or a major seam reopened and then resolved, rerun the clarification challenge once on the updated candidate synthesis.

### 7. Run A Technical-Design-Handoff Review
- Ask whether non-trivial work can proceed into `technical design` without reopening the problem frame.
- If yes, finalize the spec, keep the downstream design handoff explicit, and stop at the handoff boundary instead of beginning `technical design` in the same session.
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
- Do not use `spec.md` as a substitute for `design/`.
- When blocked, say what upstream skill or research pass must reopen and why.

## Definition Of Done
The pass is complete when:
- the spec is honest about what is decided and what is not
- the clarification gate is reconciled, explicitly waived by an eligible direct/local exception, or left blocked with rationale
- stable decisions are separated from raw evidence
- scope cuts and non-goals are explicit
- validation expectations are visible early enough for technical design and later planning
- the session stops at approved `spec.md` for non-trivial work unless an explicit waiver already allows phase collapse
- the next technical-design or reopen step is clear without turning the spec into a design bundle or a plan

## Anti-Patterns
- copying external template headings directly into the repo default shape
- turning `spec.md` into a PRD, audit report, or task board
- smuggling component maps, sequence design, or ownership maps into `spec.md` to avoid `design/`
- filling every possible NFR category whether it matters or not
- hiding contradictions under generic wording
- treating raw research notes as final decisions
- using this skill when framing, specialist design, or planning clearly owns the work
