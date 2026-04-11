# Spec Section Depth Examples

Load this file when deciding how much detail a `spec.md` section needs, trimming a bloated draft, or rescuing a spec that is too thin for `technical design`.

Repository sources of truth still win: `AGENTS.md`, `docs/spec-first-workflow.md`, and `references/spec-patterns.md`. The snippets below are examples of repo-native placement, not reusable templates.

## Good: Small Direct-Path Depth

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

Why this works: the task is narrow, the non-goal prevents scope spread, and validation is concrete without creating a separate plan.

## Good: Non-Trivial Depth

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

Why this works: the spec has enough depth for design to derive lifecycle and ownership choices, but it still avoids component maps, sequence diagrams, and task ordering.

## Bad: Too Thin

```markdown
## Decisions
- Add runtime token reload.

## Validation
- Test it.
```

Why this fails: design cannot tell the source of truth, failure behavior, non-goals, or proof path.

## Bad: Too Bloated

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

Why this fails: it copies a foreign document shape, mixes artifact ownership, and still leaves the actual planning-critical risk unresolved.

## Foreign-Template Translation Examples

| Foreign prompt | Repo-native translation |
|---|---|
| "List all personas and user journeys." | Keep only the actor or operator context that changes acceptance semantics in `Context`; put stable behavior decisions in `Decisions`. |
| "Fill every NFR category." | Put only quality constraints that shape design in `Constraints`; omit irrelevant categories. |
| "Define the implementation roadmap." | Summarize the planning intent in `Plan Summary / Link`; move sequencing to `plan.md`. |
| "Attach complete architecture." | Keep the spec decision-level; route component maps, sequences, and ownership maps to `design/`. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; do not copy their templates into this repository's spec shape.

- NASA, "Appendix C: How to Write a Good Requirement": https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- IEEE SA, "IEEE 830-1984: IEEE Guide for Software Requirements Specifications": https://standards.ieee.org/ieee/830/1220/
- IREB CPRE Online Glossary, "Software requirements specification" and related RE terms: https://cpre.ireb.org/en/downloads-and-resources/glossary
- Frattini et al., "Requirements Quality Research: a harmonized Theory, Evaluation, and Roadmap": https://arxiv.org/abs/2309.10355
