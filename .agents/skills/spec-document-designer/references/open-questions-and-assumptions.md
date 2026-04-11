# Open Questions And Assumptions

Load this file when a spec has unknowns, TBDs, blocked clarification items, external policy decisions, or soft assumptions that could later change planning.

Use explicit labels so the spec is honest without becoming a parking lot.

## Good: Actionable Assumptions

```markdown
## Open Questions / Assumptions
- [assumption] The existing tenant ID remains the isolation boundary for this change; no new tenant hierarchy is introduced.
- [accepted_risk] The first implementation will not backfill historical records because the feature only affects new writes.
- [defer_to_design] The exact package boundary for the reload coordinator belongs in `design/ownership-map.md`; the spec only decides that config remains the source of truth.
```

Why this works: each item says how to treat the uncertainty and keeps design-only detail out of the spec.

## Good: User-Only Decision

```markdown
## Open Questions / Assumptions
- [requires_user_decision] Product must choose whether expired invites should be hidden or shown as disabled. Planning is blocked because this changes API response semantics and tests.
```

Why this works: the spec does not invent product policy to get past approval.

## Bad: Decorative Unknowns

```markdown
## Open Questions / Assumptions
- TBD
- Maybe performance?
- Need to check stuff.
```

Why this fails: it does not identify the decision, impact, owner, or unblock path.

## Bad: Hiding A Blocker As A Decision

```markdown
## Decisions
- Expired invites probably stay visible, but this can change later.
```

Why this fails: the uncertainty changes API behavior, so it belongs in `Open Questions / Assumptions` until resolved or explicitly accepted.

## Foreign-Template Translation Examples

| Foreign section | Repo-native translation |
|---|---|
| "Risks and dependencies" | Stable accepted constraints go in `Constraints`; unresolved items go in `Open Questions / Assumptions`. |
| "TBD list" | Replace bare TBDs with `[assumption]`, `[requires_user_decision]`, `[defer_to_design]`, `[targeted_research]`, or `[accepted_risk]` plus the impact. |
| "Business questions" | If the repo cannot answer them safely, record `requires_user_decision` and keep the spec draft or blocked. |
| "Future work" | Put explicit non-goals in `Scope / Non-goals`; do not leave vague future work in assumptions. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; repo workflow rules decide final placement.

- NASA, "Appendix C: How to Write a Good Requirement": https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- IREB CPRE Online Glossary: https://cpre.ireb.org/en/downloads-and-resources/glossary
- Frattini et al., "Requirements Quality Research: a harmonized Theory, Evaluation, and Roadmap": https://arxiv.org/abs/2309.10355
