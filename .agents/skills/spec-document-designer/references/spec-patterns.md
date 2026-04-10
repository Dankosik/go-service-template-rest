# Spec Patterns Reference

Use this file as a compact pattern bank when designing or repairing repository-native `spec.md` documents.

Rule: these patterns are prompts for coverage, not a second source of truth and not headings to copy mechanically.

## Default Repo Shape

The default `spec.md` sections are:

1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Plan Summary / Link`
7. `Validation`
8. `Outcome`

Use all of them only when they materially help the reader.

## External Pattern Bank

### Spec Kit

Borrow when the spec needs:
- independently testable slices
- explicit edge cases
- clear capability requirements
- measurable success criteria
- explicit assumptions

Do not copy:
- PRD-style section sprawl
- task structure into `spec.md`
- multi-artifact packs as default behavior

### BMAD

Borrow when the spec needs:
- discovery-first assembly
- user journeys to expose hidden capability gaps
- scope ladder (`MVP / later / not now`)
- selective NFRs only when relevant
- a polish pass for duplication, contradictions, or dropped soft constraints

Do not copy:
- rigid step-file choreography
- forced menus or interaction rituals
- full PRD + architecture pack as the baseline artifact model

### Superpowers

Borrow when the spec needs:
- design-before-plan discipline
- a `2-3` approach comparison before locking a decision
- anti-placeholder self-review
- human-readable design narrative

Do not copy:
- highly interactive one-question workflow when the repository does not need it
- implementation-plan detail into the spec

### Spec-Driven Workflow

Borrow when the spec needs:
- a clarification sufficiency check
- latest-standards research when current guidance materially changes the design
- demoable units
- proof artifacts
- repository standards as an explicit constraint

Do not copy:
- full numbered audit/proofs artifact structure by default
- verbose product-spec sections when the task only needs a lean decision record

## Concern -> Artifact Mapping

| Concern | Ask yourself | Likely home |
|---|---|---|
| Behavior delta | What changes for the user/operator and what stays the same? | `Context`, `Decisions` |
| Independently testable slices | What are the smallest meaningful behavior slices or user-visible checkpoints? | `Decisions`, `Validation` |
| Edge cases / failure semantics | Which corner cases could change planning or acceptance semantics? | `Constraints`, `Decisions`, `Open Questions / Assumptions` |
| Capability contract | What must the system be able to do, without implementation leakage? | `Decisions` |
| Key entities / state | Which domain objects or state transitions matter for reasoning about the change? | `Context`, `Decisions` |
| Scope ladder | What is in scope now, later, and explicitly not now? | `Scope / Non-goals` |
| Relevant NFRs | Which quality constraints actually shape design: performance, security, compliance, etc.? | `Constraints` |
| Alternatives / rejected paths | Did the team reject a materially different path that planning should not quietly revive? | `Decisions` |
| Proof / validation expectations | What evidence would show the feature is real, correct, or ready? | `Validation` |
| Repository standards | Which repo-specific patterns, compatibility rules, or workflow constraints shape the spec? | `Context`, `Constraints` |
| Unresolved ambiguity | What is still unknown, who owns it, and what unblocks it? | `Open Questions / Assumptions` |
| Execution order | What must happen first, in phases, or at checkpoints? | `plan.md`, not `spec.md` |

## Translation Rules

1. If it is a stable decision, keep it in `Decisions`.
2. If it is evidence, comparison, benchmark output, or external research, move it to `research/*.md`.
3. If it is execution order, task decomposition, or coder instructions, move it to `plan.md`.
4. If it is unresolved but visible, keep it in `Open Questions / Assumptions`.
5. If the detail is too large for `spec.md`, summarize the decision and link out.
6. If a foreign section adds no planning value here, do not import it.

## Lightweight Self-Review

Before finishing the spec, check:

1. Is this request mature enough for spec design, or should it go back to framing?
2. Can planning proceed without silently reopening core design?
3. Did any raw research leak into `Decisions`?
4. Did any task list or implementation sequence leak into `spec.md`?
5. Are scope cuts and non-goals explicit enough to prevent drift?
6. Are material edge cases or proof expectations visible?
7. Did I import only the patterns that help this task, rather than all possible sections?
8. If a specialist contradiction remains, did I escalate instead of smoothing it over?
