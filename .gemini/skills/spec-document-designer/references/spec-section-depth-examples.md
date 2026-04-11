# Spec Section Depth Examples

## Behavior Change Thesis
When loaded for a spec that is too thin, too bloated, or hard to size, this file makes the model choose risk-proportional section depth instead of writing either "add X, test it" or a full PRD/design bundle.

## When To Load
Load this when deciding how much detail a `spec.md` section needs, trimming a bloated draft, or rescuing a draft that lacks enough decisions for `technical design`.

## Decision Rubric
- For tiny direct-path work, include only the behavior delta, the scope cut that prevents drift, stable decisions, and concrete validation hooks.
- For non-trivial work, add constraints, assumptions, and handoff notes when they shape technical design or planning.
- Expand a section because risk or ambiguity demands it, not because the default section list exists.
- Omit empty headings. A missing section is better than a placeholder.
- Do not use section depth to smuggle component maps, call sequences, file edits, or task order into `spec.md`.

## Imitate: Small Direct-Path Depth

```markdown
## Context
The health endpoint currently reports only process liveness. Operators need one additional field that shows whether configuration loaded successfully.

## Scope / Non-goals
- In scope: add a `config_loaded` boolean to the existing health response.
- Non-goal: add database, cache, or downstream dependency checks.

## Decisions
- The endpoint keeps its current status-code behavior; the new field is informational only.
- The response remains backward compatible because existing fields are unchanged.

## Validation
- Focused handler test covers the new field for loaded and failed config states.
```

Copy this: narrow context, one scope cut, decision-level behavior, and proof hooks without a plan.

## Imitate: Non-Trivial Depth

```markdown
## Context
Admin token rotation currently requires a restart and creates an avoidable operator window where old tokens remain accepted. The change should support runtime reload while preserving current authentication semantics.

## Scope / Non-goals
- In scope: reload configured admin tokens from the existing configuration source.
- In scope: preserve the existing request authentication contract.
- Non-goal: introduce a new secret store or user-management model.

## Constraints
- Secret values must not be logged.
- Reload failure must keep the last known-good token set active.
- The behavior must fit the repository lifecycle model described in `docs/repo-architecture.md`.

## Decisions
- Token reload is a configuration concern, not a new auth domain.
- Failed reloads are reported as degraded operator state while request auth continues with last known-good data.

## Open Questions / Assumptions
- [assumption] The existing config source remains the source of truth for this change.

## Validation
- Unit tests cover successful reload, failed reload preserving old tokens, and log redaction.
- Integration smoke covers reload without process restart.
```

Copy this: enough source-of-truth, failure, scope, and proof detail for design, without package ownership or sequencing.

## Reject: Too Thin

```markdown
## Decisions
- Add runtime token reload.

## Validation
- Test it.
```

Failure: design cannot infer source of truth, failure behavior, non-goals, or proof obligations.

## Reject: Too Bloated

```markdown
## Personas
## Market Landscape
## Full Architecture
## API Spec
## Sprint Tasks
## Rollout Calendar
## Metrics Dashboard
## Risks
- TBD
```

Failure: this copies a foreign shape, mixes artifact ownership, and still leaves the planning-critical risk unresolved.

## Agent Traps
- Expanding all default sections even when the task is direct-path.
- Treating non-trivial depth as permission to write design.
- Leaving `TBD` in a bloated spec because the document already "looks complete."
- Removing non-goals while trimming, which lets planning re-expand scope.
