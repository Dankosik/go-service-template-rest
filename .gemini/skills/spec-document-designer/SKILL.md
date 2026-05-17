---
name: spec-document-designer
description: "Design and normalize repository-native `spec.md` documents for this workflow. Use when the orchestrator has a framed change or synthesized research and needs to turn it into a stable direct/lean/full-shaped `spec.md` with the right section depth, decision placement, inline `Risk Challenge` or formal clarification-gate reconciliation, audit trail, and handoff before non-trivial planning. Skip raw ideation, triggered design-bundle assembly, full task breakdown, and implementation coding."
---

# Spec Document Designer

## Purpose
Turn a framed request or synthesized research into a repository-native `spec.md` that is honest, stable, and ready to hand off into `technical design` without turning it into a PRD, a design bundle, or a task list.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Scope
- draft a fresh `spec.md` when the task is mature enough for specification
- normalize existing drafts that are too thin, too bloated, or shaped like a foreign template
- choose the right section depth for the task while staying inside the repository's artifact model
- translate useful coverage prompts from external spec workflows into repo-native sections
- keep blockers, assumptions, inline `Risk Challenge` or clarification outcomes, validation hooks, and handoff links visible before lean tasking or triggered `technical design`
- keep the handoff focused on the current decision frontier: record downstream consequences that matter, but do not promote every visible consequence into a new design decision inside `spec.md`

## Boundaries
Do not:
- refine a raw product idea; use `idea-refine`
- perform engineering framing on an under-shaped request; use `spec-first-brainstorming`
- absorb unresolved cross-domain design contradictions that belong in `go-design-spec` or specialist `*-spec` skills
- assemble a triggered split `design/` bundle; that belongs to `go-design-spec`
- produce task breakdown, execution sequencing, or coder instructions; that belongs to `planning-and-task-breakdown`
- silently skip triggered `technical design` by smuggling split-design detail into `spec.md`; lean-local compact design answers are allowed when their trigger rationale is explicit
- mark lean-local `spec.md` approved without inline `Risk Challenge`, or mark full-orchestrated/protected-domain `spec.md` approved while the formal `spec-clarification-challenge` gate is unresolved or blocked
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
- For lean-local work, the handoff path is `spec.md` with compact design answers and inline `Risk Challenge` -> `tasks.md`.
- For full-orchestrated or protected-domain work, the handoff path is `spec.md -> triggered design context -> tasks.md`, and formal `spec-clarification-challenge` is required before approval unless explicitly waived by an eligible exception.
- For dedicated specification sessions, this pass ends at approved `spec.md`; triggered `technical design` or planning begins in a new session unless an upfront direct/lean waiver was already recorded.
- Use the repository's default section set unless merging sections makes the file clearer.
- Treat external frameworks as coverage prompts, not as headings to copy.
- Put stable decisions in `spec.md`, lean compact design answers in `spec.md` or one `design/overview.md`, split task-local technical context in `design/` when triggered, execution order and task detail in `tasks.md`, and preserved evidence in `research/*.md`.
- Prefer short explicit bullets over template sludge.
- Omit empty sections instead of padding the document for completeness theater.
- Deep coverage is good, but `spec.md` should close only the decisions needed before technical design. Later-domain consequences that do not require a new decision now should stay as constraints, proof obligations, or explicit `no new decision required` notes.

## Context Intake (Dynamic Loading)

Rule: load the smallest sufficient set of artifacts. Do not bulk-load folders by default.

Always load:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Load by trigger:
- existing spec rewrite or continuation: the active `spec.md`
- triggered workflow control: the matching `workflow-plan.md`
- research-backed synthesis: the relevant `research/*.md`
- formal spec approval: `.agents/skills/spec-clarification-challenge/SKILL.md` when full-orchestrated, high-risk, protected-domain, or otherwise triggered
- task-breakdown drift check: the matching `tasks.md` when it exists
- existing technical-design bundle nearby: `design/overview.md` and only the smallest set of affected design artifacts needed to confirm ownership boundaries, not to author triggered split design in this pass

Conflict resolution:
- repository contract beats the reference file
- the more specific artifact beats generic guidance
- if a foreign template conflicts with repo artifact ownership, keep the repo model and translate the useful idea instead

Unknowns:
- if critical facts are missing, mark them as `[assumption]` or escalate

## Reference Files Selector

References are compact rubrics and example banks, not exhaustive checklists. Load at most one reference by default after core context. Load more than one only when the task clearly spans independent decision pressures, such as both clarification-gate reconciliation and validation closeout.

Before loading a reference, use this behavior-change thesis test: "When loaded for symptom X, this file makes the model choose Y instead of likely mistake Z." If no row below matches that sentence, do not load a reference.

| Symptom | Behavior change | Load |
|---|---|---|
| Foreign-template pressure, framework vocabulary, or coverage gaps without a narrower symptom | Translate only behavior-changing coverage concerns into repo-native sections instead of copying PRD, BMAD, Spec Kit, Superpowers, or SDD headings | `references/spec-patterns.md` |
| A spec is too thin, too bloated, or hard to size | Choose risk-proportional section depth instead of writing either "add X, test it" or a full PRD/design bundle | `references/spec-section-depth-examples.md` |
| `spec.md` mixes final decisions with research, design detail, tasks, or transcripts | Move each fact to its owning artifact instead of using `spec.md` as an all-purpose dump | `references/decision-placement-and-artifact-ownership.md` |
| The draft has `TODO`, `TBD`, blockers, soft assumptions, or user-only decisions | Label and route uncertainty by unblock path instead of inventing certainty or hiding blockers in `Decisions` | `references/open-questions-and-assumptions.md` |
| A `spec-clarification-challenge` pass returned findings, or non-trivial spec approval is questionable | Reconcile findings into final spec sections and gate status instead of pasting transcripts or approving through blockers | `references/clarification-gate-reconciliation.md` |
| Validation language is vague, acceptance criteria need proof shaping, or `Outcome` is being written | Separate forward-looking proof obligations from evidence-backed outcome claims instead of writing "run tests" or "done" | `references/validation-and-outcome-sections.md` |
| A non-trivial spec is at `spec.md -> design/` handoff, or design detail leaks into the spec | Keep `spec.md` at behavior-decision level with explicit handoff and reopen conditions instead of stuffing component maps, sequences, ownership maps, or task lists into the spec | `references/spec-handoff-to-technical-design.md` |

Neighboring-reference rule:
- For mixed-content cleanup, prefer `decision-placement-and-artifact-ownership.md`.
- For final handoff readiness or design leakage at approval time, prefer `spec-handoff-to-technical-design.md`.
- For foreign-template sprawl plus depth problems, prefer `spec-section-depth-examples.md` if the main decision is sizing; prefer `spec-patterns.md` if the main decision is translating framework concerns.
- Do not load a broad reference when a narrower row already matches the symptom.

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
  - `Task Breakdown / Handoff Link`
  - `Validation`
  - `Outcome`
- For lean-local specs, prefer the compact shape from `docs/spec-first-workflow.md`: `Intent`, `Scope / Non-goals`, `Behavior / Contract Delta`, `Decisions`, `Compact Design`, `Risk Challenge`, `Task Handoff`, `Validation`, and `Outcome`.
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
- Research-backed decisions should link to the specific preserved research note or source surface when the provenance would help a later session trust or revisit the decision; keep the link short and do not copy the evidence narrative into `Decisions`.
- Task-local technical design context belongs in `design/`.
- Task sequencing and execution detail belong in `tasks.md`.
- Unresolved but visible gaps belong in `Open Questions / Assumptions`.
- Label `Open Questions / Assumptions` by unblock path when the uncertainty affects later sessions: `[assumption]` for bounded proceed-and-revisit assumptions, `[accepted_risk]` for deliberate risk with limits or proof obligations, `[requires_user_decision]` for external product or policy choices, `[targeted_research]` for repository evidence still needed, `[defer_to_design]` only for design-owned details after behavior is decided, and `[reopen_spec_if_false]` for downstream discoveries that would invalidate a spec-level assumption.
- Do not force `go-design-spec` or the planner to recover ownership, sequence, or execution order from `spec.md` when separate artifacts are warranted; for lean local, make those compact answers explicit in `spec.md`.

### Technical-Design Handoff Competency
- A full-orchestrated or design-triggered spec must let `go-design-spec` derive the task-local `design/` bundle without silently reopening core problem framing.
- A lean-local spec must let `tasks.md` be written from explicit `Compact Design` answers without forcing hidden design recovery during planning.
- Keep blockers, accepted risks, and reopen conditions explicit.
- Preserve non-goals and scope cuts so technical design does not re-expand the change.
- Keep only the task-breakdown or handoff link in `spec.md`.
- When another domain is only a dependent consequence, record the consequence and its owner instead of expanding `spec.md` into a parallel design bundle.

### Clarification-Gate Competency
- Before approving lean-local `spec.md`, ensure inline `Risk Challenge` is recorded as `PASS` or `CONCERNS`; `FULL_REQUIRED` escalates to the formal path.
- Before approving full-orchestrated or protected-domain `spec.md`, ensure the orchestrator has run a read-only `spec-clarification-challenge` lane, preferably through `challenger-agent`, using exactly that one skill.
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
- If the task has foreign-template pressure or under-covered acceptance semantics, use `references/spec-patterns.md` to ask which concerns matter here:
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
- Link out instead of duplicating detail when preserved evidence already exists elsewhere, especially for non-obvious research-backed decisions that a later session may need to audit.

### 6. Run Or Enforce The Clarification Gate
- For lean local, confirm the inline `Risk Challenge` ran after candidate decisions existed and before approval.
- For full-orchestrated or protected-domain work, confirm the `spec-clarification-challenge` gate ran after candidate decisions existed and before approval.
- If a required formal gate has not run, prepare the compact bundle and route one read-only subagent lane, preferably `challenger-agent`, using only `spec-clarification-challenge`.
- If the gate returns material questions, keep `spec.md` draft or blocked until the orchestrator reconciles them.
- If targeted expert research is required, route the appropriate upstream research or expert lane instead of inventing an answer in the spec.
- If a question is truly external product or business policy, record `requires_user_decision` and leave the spec blocked or partially draft.
- If material decisions changed or a major seam reopened and then resolved, rerun the clarification challenge once on the updated candidate synthesis.

### 7. Run A Technical-Design-Handoff Review
- Ask whether the work can proceed into lean tasking or triggered `technical design` without reopening the problem frame.
- If yes, finalize the spec, keep the downstream handoff explicit, and stop at the handoff boundary instead of beginning the next concern in the same session.
- If no, escalate to the missing upstream skill or specialist lane.

## Output Expectations
Return or write spec work using the repository-native section order:
- `Context`
- `Scope / Non-goals`
- `Constraints`
- `Decisions`
- `Open Questions / Assumptions`
- `Task Breakdown / Handoff Link`
- `Validation`
- `Outcome`

Rules:
- Merge sections when clearer.
- Do not create empty sections.
- Do not dump full task lists or execution steps into `spec.md`.
- Do not use `spec.md` as a substitute for triggered split `design/`; lean-local `Compact Design` is allowed only when sufficient and explicit.
- Do not copy a historical bundle under `specs/` as a template; use it only as task-local precedent and preserve the current task's own trigger rationale.
- When blocked, say what upstream skill or research pass must reopen and why.

## Definition Of Done
The pass is complete when:
- the spec is honest about what is decided and what is not
- the inline lean `Risk Challenge` or formal clarification gate is reconciled, explicitly waived by an eligible direct/lean exception, or left blocked with rationale
- stable decisions are separated from raw evidence
- scope cuts and non-goals are explicit
- validation expectations are visible early enough for lean tasking or triggered technical design and later planning
- the session stops at approved `spec.md` for dedicated specification work unless an explicit waiver already allows phase collapse
- the next technical-design or reopen step is clear without turning the spec into a design bundle or a plan

## Anti-Patterns
- copying external template headings directly into the repo default shape
- turning `spec.md` into a PRD, audit report, or task board
- smuggling full component maps, sequence design, or ownership maps into `spec.md` to avoid triggered `design/`
- filling every possible NFR category whether it matters or not
- hiding contradictions under generic wording
- treating raw research notes as final decisions
- using this skill when framing, specialist design, or planning clearly owns the work
