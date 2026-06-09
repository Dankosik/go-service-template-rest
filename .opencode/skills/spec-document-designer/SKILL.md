---
name: spec-document-designer
description: "Design and normalize repository-native `spec.md` documents for this workflow. Use when the orchestrator has a framed change or synthesized research and needs to turn it into a stable direct/lean/full-shaped review-ready `spec.md` with the right section depth, decision placement, inline `Risk Challenge` or formal clarification-gate reconciliation, audit trail, and handoff before mandatory specification review. Skip raw ideation, triggered design-bundle assembly, full task breakdown, and implementation coding."
---

# Spec Document Designer

## Purpose
Turn a framed request or synthesized research into a repository-native `spec.md` that is honest, stable, and ready to hand off into mandatory specification review without turning it into a PRD, a design bundle, or a task list.

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
- keep blockers, assumptions, inline `Risk Challenge` or clarification outcomes, validation hooks, and handoff links visible before specification review
- keep the handoff focused on the current decision frontier: record downstream consequences that matter, but do not promote every visible consequence into a new design decision inside `spec.md`
- for replacement or cleanup-discipline work, name known legacy surfaces and the expected remove/refactor/retain semantics before tasking begins
- for non-trivial custom implementation, new runtime dependencies, or meaningful helper/abstraction choices, record dependency/OSS due diligence before tasking begins
- for non-trivial architecture, system-design, workflow, integration, resilience, consistency, data-flow, or abstraction choices, record Pattern Fit Diligence before tasking begins or route to research/technical design when the pattern comparison is not ready

## Boundaries
Do not:
- refine a raw product idea; use `idea-refine`
- perform engineering framing on an under-shaped request; use `spec-first-brainstorming`
- absorb unresolved cross-domain design contradictions that belong in `go-design-spec` or specialist `*-spec` skills
- assemble a triggered split `design/` bundle; that belongs to `go-design-spec`
- produce task breakdown, execution sequencing, or coder instructions; that belongs to `planning-and-task-breakdown`
- silently skip triggered `technical design` by smuggling split-design detail into `spec.md`; lean-local compact design answers are allowed when their trigger rationale is explicit
- mark lean-local `spec.md` review-ready without inline `Risk Challenge`, or mark full-orchestrated/protected-domain `spec.md` review-ready while the formal `spec-clarification-challenge` gate is unresolved or blocked
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
- For lean-local work, the handoff path is `spec.md` with compact design answers, recorded subagent gate decision, and inline `Risk Challenge` -> specification review -> `tasks.md`.
- For full-orchestrated or protected-domain work, the handoff path is `spec.md -> specification review -> triggered design context -> technical design review -> tasks.md`, and formal `spec-clarification-challenge` is required before the spec is review-ready.
- For dedicated specification sessions, this pass ends at review-ready `spec.md`; specification review begins in a new session unless the task is direct-path and no `spec.md` is required.
- Use the repository's default section set unless merging sections makes the file clearer.
- Treat external frameworks as coverage prompts, not as headings to copy.
- Put stable decisions in `spec.md`, lean compact design answers in `spec.md` or one `design/overview.md`, split task-local technical context in `design/` when triggered, execution order and task detail in `tasks.md`, and preserved evidence in `research/*.md`.
- For replacement specs, record known old identifiers, routes, configs, commands, generated outputs, fixtures, docs, skills, agents, or mirrors and decide whether each is removed, refactored into the active path, or retained with owner, reason, proof, and exit condition.
- For dependency-sensitive specs, record selected and rejected stdlib, established repository-pattern, mature OSS, and custom-code options with current evidence. Missing dependency/OSS due diligence blocks review-readiness when the task would otherwise add a dependency, build custom infrastructure, or introduce a meaningful helper/abstraction.
- For pattern-sensitive specs, record selected and rejected design or system-design patterns with current source descriptions, real-use examples, task applicability, Go/repository fit, and custom-design justification when no known pattern fits. Missing Pattern Fit Diligence blocks review-readiness when the task would otherwise invent architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape.
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
- formal spec review-readiness: `.agents/skills/spec-clarification-challenge/SKILL.md` when full-orchestrated, high-risk, protected-domain, or otherwise triggered
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
| A `spec-clarification-challenge` pass returned findings, or non-trivial spec review-readiness is questionable | Reconcile findings into final spec sections and gate status instead of pasting transcripts or marking review-ready through blockers | `references/clarification-gate-reconciliation.md` |
| Validation language is vague, acceptance criteria need proof shaping, or `Outcome` is being written | Separate forward-looking proof obligations from evidence-backed outcome claims instead of writing "run tests" or "done" | `references/validation-and-outcome-sections.md` |
| A non-trivial spec is at `spec.md -> design/` handoff, or design detail leaks into the spec | Keep `spec.md` at behavior-decision level with explicit handoff and reopen conditions instead of stuffing component maps, sequences, ownership maps, or task lists into the spec | `references/spec-handoff-to-technical-design.md` |

Neighboring-reference rule:
- For mixed-content cleanup, prefer `decision-placement-and-artifact-ownership.md`.
- For final handoff readiness or design leakage at review-readiness time, prefer `spec-handoff-to-technical-design.md`.
- For foreign-template sprawl plus depth problems, prefer `spec-section-depth-examples.md` if the main decision is sizing; prefer `spec-patterns.md` if the main decision is translating framework concerns.
- Do not load a broad reference when a narrower row already matches the symptom.

## Hard Skills

### Mission
Make `spec.md` stable enough for mandatory specification review while preserving the repository's single-source-of-truth discipline.

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
- For lean-local specs, prefer the compact shape from `docs/spec-first-workflow.md`: `Intent`, `Scope / Non-goals`, `Behavior / Contract Delta`, `Decisions`, `Dependency / OSS Due Diligence`, `Compact Design`, `Subagent Gate Decision`, `Risk Challenge`, `Task Handoff`, `Validation`, and `Outcome`.
- Merge sections only when it makes the document clearer.
- Expand depth based on task risk and ambiguity, not on habit.

### Coverage Competency
- Capture the behavior delta, affected actors, scope cuts, edge semantics, validation expectations, and material constraints.
- Capture dependency/OSS due diligence when relevant: current Go stdlib fit, established repo pattern fit, OSS candidates, maintenance/release activity, adoption such as stars or domain-equivalent signals, license, security posture, transitive dependency cost, API stability, repository-boundary fit, selected option, rejected options, and custom-code justification.
- Capture Pattern Fit Diligence when relevant: known candidate patterns, concise source descriptions, real-use examples, why the pattern forces match or do not match this task, Go/repository fit, selected pattern, rejected patterns, and custom-design justification.
- When the behavior delta replaces an old path, capture the legacy-surface delta explicitly: what old surfaces are gone, what is refactored into the active path, what remains intentionally retained, and what proof will make that claim auditable.
- Ask and answer before review-readiness: `Does this change replace an existing path?` If yes, list known old code, tests, fixtures, configs, docs, generated outputs, skills, agents, or mirrors. If no, record `No known replacement surface`.
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
- A dependency-sensitive spec must let technical design or planning consume the chosen dependency/custom-code approach without rerunning open-ended library selection.
- A pattern-sensitive spec must let technical design or planning consume the selected design/system pattern without rerunning open-ended pattern selection; when the comparison is too dense, link to preserved `research/pattern-fit.md` and route detailed application to `design/`.
- Keep blockers, accepted risks, and reopen conditions explicit.
- Preserve non-goals and scope cuts so technical design does not re-expand the change.
- Keep only the task-breakdown or handoff link in `spec.md`.
- When another domain is only a dependent consequence, record the consequence and its owner instead of expanding `spec.md` into a parallel design bundle.

### Clarification-Gate Competency
- Before marking lean-local `spec.md` review-ready, ensure the subagent gate decision is recorded as consumed lane summaries or local-only rationale, and inline `Risk Challenge` is recorded as `PASS` or `CONCERNS`; `FULL_REQUIRED` escalates to the formal path.
- Before marking full-orchestrated or protected-domain `spec.md` review-ready, ensure the orchestrator has run formal `spec-clarification-challenge`. Broad formal clarification should use multiple read-only `challenger-agent` lanes with distinct lenses; each lane still uses exactly one skill. Scoped-down single-lane gates need a rationale listing every default lens, the approval-critical question considered for that lens, retained lane or lanes, and why omitted lenses cannot change review-readiness.
- Formal clarification is not waivable while the work remains full-orchestrated, protected-domain, high-risk, hard-to-reverse, cross-domain, or user-requested deep challenge. If the trigger no longer applies, record shape reclassification with trigger-matrix evidence before marking the formal gate not expected.
- The challenge returns questions for orchestrator reconciliation; it does not write files or make final decisions.
- Resolve each planning-critical item from existing evidence, targeted research, an expert subagent lane, explicit risk acceptance, design deferral, or `requires_user_decision`.
- Store only final resolved outcomes in `spec.md`: stable outcomes in `Decisions`, remaining assumptions in `Open Questions / Assumptions`, and proof consequences in `Validation`.
- Do not copy raw clarification transcripts into `spec.md`.

### Spec Review Competency
- This is an authoring self-check only; it is not the mandatory specification-review gate and does not replace read-only review lanes.
- Before handoff, scan for placeholders, `TODO`, `TBD`, contradictions, duplicated content, scope spread, implementation leakage, and research dumped into `Decisions`.
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
- For lean local, confirm the inline `Risk Challenge` ran after candidate decisions existed and before review-readiness.
- For full-orchestrated or protected-domain work, confirm the `spec-clarification-challenge` gate ran after candidate decisions existed and before review-readiness.
- If a required formal gate has not run, prepare the compact bundle and route the appropriate read-only challenger lane set, preferably `challenger-agent` with `spec-clarification-challenge`. For broad or multi-domain full-orchestrated, protected-domain, high-risk, cross-domain, hard-to-reverse, or user-requested deep challenge work, route distinct lenses instead of one generic challenger; use a single lane only with scoped-down rationale that proves omitted lenses cannot affect review-readiness. Keep `Lens` as coverage metadata and keep the existing challenge classifications stable.
- If the gate returns material questions, keep `spec.md` draft or blocked until the orchestrator reconciles them.
- If targeted expert research is required, route the appropriate upstream research or expert lane instead of inventing an answer in the spec.
- A direct/lean waiver may affect same-session phase collapse only; it cannot waive targeted expert research or subagent work required by a clarification finding.
- If a question is truly external product or business policy, record `requires_user_decision` and leave the spec blocked or partially draft.
- If material decisions changed or a major seam reopened and then resolved, rerun the clarification challenge once on the updated candidate synthesis.

### 7. Prepare The Specification-Review Handoff
- Ask whether the work can proceed into mandatory specification review without reopening the problem frame.
- If yes, finalize the review-ready spec, keep the specification-review handoff explicit, and stop at the handoff boundary instead of beginning the next concern in the same session.
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
- Do not copy another bundle under `specs/` as a template; use it only as task-local precedent and preserve the current task's own trigger rationale.
- When blocked, say what upstream skill or research pass must reopen and why.

## Definition Of Done
The pass is complete when:
- the spec is honest about what is decided and what is not
- the subagent gate decision plus inline lean `Risk Challenge` or formal clarification gate is reconciled, marked not expected after recorded shape reclassification, or left blocked with rationale
- stable decisions are separated from raw evidence
- scope cuts and non-goals are explicit
- validation expectations are visible early enough for specification review, lean tasking or triggered technical design, mandatory technical design review, and later planning
- replacement specs leave no known legacy-surface removal, refactor, retention, or negative-proof decision for implementation to infer
- replacement specs either list known legacy surfaces or explicitly say `No known replacement surface`
- pattern-sensitive specs either record selected/rejected Pattern Fit Diligence or route the missing comparison to research or technical design before planning
- the session stops at review-ready `spec.md` for dedicated specification work; specification review is the next non-trivial gate and phase-collapse waiver does not waive targeted expert research or subagent work required by a clarification finding
- the next technical-design or reopen step is clear without turning the spec into a design bundle or a plan

## Anti-Patterns
- copying external template headings directly into the repo default shape
- turning `spec.md` into a PRD, audit report, or task board
- smuggling full component maps, sequence design, or ownership maps into `spec.md` to avoid triggered `design/`
- filling every possible NFR category whether it matters or not
- hiding contradictions under generic wording
- treating raw research notes as final decisions
- naming a popular pattern without task-specific applicability, Go-fit, and rejected alternatives
- using this skill when framing, specialist design, or planning clearly owns the work
