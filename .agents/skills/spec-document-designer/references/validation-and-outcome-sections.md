# Validation And Outcome Sections

Load this file when writing proof expectations before handoff, repairing vague acceptance criteria, or closing a spec after validation evidence exists.

`Validation` is forward-looking proof intent. `Outcome` is closeout after evidence exists.

## Good: Validation Hooks

```markdown
## Validation
- Unit tests prove token reload keeps last known-good values after a failed config read.
- Integration smoke proves reload works without process restart.
- Log assertions prove secret values are redacted on reload failure.
```

Why this works: each item is specific enough for planning and testing without dictating exact test file names or task order.

## Bad: Vague Validation

```markdown
## Validation
- Run tests.
- Make sure it works.
```

Why this fails: it gives planning no proof obligation and hides important failure paths.

## Good: Outcome Closeout

```markdown
## Outcome
- Implemented runtime token reload with last known-good fallback.
- Fresh validation: `go test ./internal/auth ./internal/config` passed on 2026-04-11.
- Follow-up: none; rollout risk remains limited to the existing config source.
```

Why this works: it is written after proof exists and records the evidence without pretending broader validation ran.

## Bad: Premature Outcome

```markdown
## Outcome
- Done.
```

Why this fails: it claims completion without evidence and does not say what was actually validated.

## Foreign-Template Translation Examples

| Foreign section | Repo-native translation |
|---|---|
| "Acceptance criteria" | Convert behavior-level criteria into `Decisions` and proof-level criteria into `Validation`. |
| "Definition of Done" | Keep task/procedure gates in `plan.md` or `tasks.md`; keep only proof expectations in `Validation`. |
| "QA plan" | If the validation surface is large, trigger `test-plan.md`; otherwise keep concise proof hooks in `Validation`. |
| "Release notes" | Do not put release copy in `spec.md`; after validation, summarize actual result in `Outcome` only when useful. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; repository validation rules remain authoritative.

- NASA, "Appendix C: How to Write a Good Requirement": https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- Gojko Adzic, "Examples make it easy to spot inconsistencies": https://gojko.net/2009/05/12/examples-make-it-easy-to-spot-inconsistencies/
- Frattini et al., "Requirements Quality Research: a harmonized Theory, Evaluation, and Roadmap": https://arxiv.org/abs/2309.10355
