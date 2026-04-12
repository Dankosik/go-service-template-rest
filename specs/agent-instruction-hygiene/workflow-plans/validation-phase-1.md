# Validation Phase 1

## Goal

Prove the instruction/tooling changes are internally consistent and mirror-safe.

## Commands

- `make agents-check`
- `make skills-check`
- `make guardrails-check`
- targeted shell checks for the new config fields, review-skill references, and README/docs updates

## Closeout Updates

- `tasks.md` checkboxes updated.
- `spec.md` `Validation` and `Outcome` updated.
- `workflow-plan.md` current phase/status updated.

## Residual Risk Policy

Full `make check` was not run because the change is docs/config/scripts/skill instructions only. Narrower validation covered mirrors, guardrails, shell syntax, TOML parsing, whitespace, and targeted stale-string/routing checks.
