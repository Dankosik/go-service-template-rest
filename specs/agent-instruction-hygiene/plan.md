# Plan

## Strategy

Make targeted repository-instruction changes in one local phase, then validate with mirror and guardrail checks.

## Steps

1. Add shared subagent docs.
2. Add review skills for delivery, distributed, and observability.
3. Update Codex config and affected agent files.
4. Add agent mirror sync/check tooling and Makefile/Docker/CI integration.
5. Sync Claude agent mirrors and skill mirrors.
6. Update README and command docs.
7. Run validation and close out task artifacts.

## Validation Plan

- `make agents-check`
- `make skills-check`
- `make guardrails-check`
- targeted `rg` checks for inherited model policy, fan-out policy, and new review skill routing

## Reopen Conditions

Reopen planning if:
- Codex agent TOML fields are unsupported by local validation,
- the sync script cannot produce stable Claude mirrors,
- adding review skills requires bundled references or eval scaffolding to pass existing repository checks.

## Implementation Readiness

PASS, with the lightweight-local waiver recorded in `workflow-plan.md`.
