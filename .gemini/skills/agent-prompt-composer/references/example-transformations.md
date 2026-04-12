# Example Transformations

These examples show the quality bar for this skill.
Each example links to a rough user-input fixture and then shows the final English prompt shape.

## Example 1: HTTP OPTIONS / CORS Policy Bug

Raw fixture:
- `evals/files/http-options-cors.md`

Final prompt:

```md
Objective
Fix and regression-test HTTP `OPTIONS` handling so known routes return the correct `Allow` header and `204 No Content`, while CORS preflight remains explicitly fail-closed when CORS is not enabled.

Confirmed Signals And Exact Identifiers
- `OPTIONS`
- `Allow`
- `preflight`
- `CORS`
- `problem json`
- `router_test.go`
- confirmed repo files: `internal/infra/http/router.go`, `internal/infra/http/router_test.go`

Relevant Repository Context
- HTTP transport policy lives in `internal/infra/http/`.
- Router behavior is implemented in `internal/infra/http/router.go`.
- The package already has focused HTTP policy tests in `internal/infra/http/router_test.go`.
- OpenAPI is source-of-truth in `api/openapi/service.yaml`; only touch it if the public contract actually changes.

Inspect First
- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`
- likely `internal/infra/http/openapi_contract_test.go` only if the change turns out to affect contract/runtime alignment

Requested Change / Problem Statement
- The current router behavior around `OPTIONS` appears mixed: existing paths should return `204` plus the correct `Allow` header, while CORS preflight should remain explicit and fail-closed when CORS is not enabled.
- Preserve the `problem+json` response shape and message stability where policy intentionally returns an error response.
- Avoid unnecessary OpenAPI churn if this is only transport policy and test coverage.

Constraints / Preferences / Non-goals
- Do not change OpenAPI or generated API bindings unless the public contract truly changes.
- Keep the current fail-closed CORS posture.
- Prefer a focused router-policy fix with regression tests over unrelated cleanup.

Acceptance Criteria / Expected Outcome
- `OPTIONS` on a known non-preflight path returns `204 No Content` with the correct `Allow` header.
- CORS preflight without enabled CORS remains fail-closed with the expected `problem+json` response.
- Tests clearly cover both branches and keep message/header behavior intentional.

Validation / Verification
- Run focused HTTP package tests first, especially the router policy tests in `./internal/infra/http`.
- If the change ends up touching contract-visible behavior or generated API code, run `make openapi-check`; otherwise keep verification scoped to the HTTP package and relevant repo checks.

Assumptions / Open Questions
- [assumption] The issue is localized to router policy and tests rather than generated OpenAPI bindings.
```

## Example 2: Repo-Local Skill / Prompt Tooling Request

Raw fixture:
- `evals/files/skill-tooling.md`

Final prompt:

```md
Objective
Rename and revise the existing repository-local prompt-composer skill so it turns rough, incomplete, repetitive, or multilingual user task notes into strong English prompts for coding agents working inside this repo.

Confirmed Signals And Exact Identifiers
- `skill`
- `english prompt`
- `codex`
- `claude`
- `skills-sync`
- `examples`
- `evals`
- `messy user input`
- explicit constraint: local to this repo, not global

Relevant Repository Context
- Canonical local skills live under `.agents/skills/`.
- Runtime mirrors are maintained by `scripts/dev/sync-skills.sh` and the corresponding `make skills-sync` / `make skills-check` flows.
- This repo uses a spec-first workflow with master control in `specs/<feature-id>/workflow-plan.md`, phase-local routing in `specs/<feature-id>/workflow-plans/<phase>.md`, decisions in `specs/<feature-id>/spec.md`, and explicit validation evidence.

Inspect First
- `.agents/skills/`
- `scripts/dev/sync-skills.sh`
- `README.md`
- `AGENTS.md`

Requested Change / Problem Statement
- Rename and update the existing repo-local prompt-composition skill so it reconstructs intent from messy, incomplete, repetitive, or multilingual input instead of treating any single language as the special trigger.
- The resulting prompt should help another coding agent start in the right repo surfaces, preserve exact identifiers, and include the right validation direction.
- Reuse existing repository skill conventions rather than inventing a parallel format.

Constraints / Preferences / Non-goals
- Keep the solution version-controlled inside this repository.
- Do not build a global/home-directory skill.
- Keep dependencies minimal.
- Include examples and lightweight validation assets only if they materially improve maintainability.

Acceptance Criteria / Expected Outcome
- The renamed skill exists under the canonical repo-local skill path, with frontmatter, README/catalog references, examples, and eval prompts updated consistently.
- Supporting references/examples/evals are present and consistent with repo conventions.
- Runtime mirrors can be refreshed with the existing sync flow.
- The skill output is clearly better than literal translation and is tailored to this repository.

Validation / Verification
- Run `make skills-sync` and `make skills-check`.
- Validate any structured eval file such as `evals/evals.json`.
- Use realistic messy fixtures across more than one language style to review whether the produced prompt is repo-aware and actionable.

Assumptions / Open Questions
- [assumption] Updating the README skill library and the skills catalog is appropriate for discoverability.
```

## Example 3: Flaky Shutdown / Drain Investigation

Raw fixture:
- `evals/files/flaky-shutdown.md`

Final prompt:

```md
Objective
Investigate and fix a flaky shutdown/drain path where `context canceled` or worker shutdown handling may be incorrect. Do not paper over the issue by simply increasing timeouts.

Confirmed Signals And Exact Identifiers
- `shutdown`
- `drain`
- `context canceled`
- `worker`
- `test hangs`
- `bootstrap`
- `health/readiness`
- `race`
- `integration`

Relevant Repository Context
- Service lifecycle and shutdown wiring live under `cmd/service/internal/bootstrap/`.
- Readiness and drain state live in `internal/app/health/service.go`.
- Integration tests live in `test/` and are run separately from default unit tests.
- The repo prefers explicit evidence and race-aware verification for concurrency-sensitive changes.

Inspect First
- likely `cmd/service/internal/bootstrap/`
- `internal/app/health/service.go`
- nearby tests around bootstrap, health/readiness, and any shutdown or integration paths
- `test/` if the flake is only visible in integration coverage

Requested Change / Problem Statement
- The reported failure sounds like a flaky shutdown/drain path where cancellation or worker stop behavior is mishandled and occasionally leaves a test hanging.
- Start with root-cause investigation and the smallest proving surface before choosing a fix.
- Avoid timeout inflation unless evidence shows timing itself is the real contract.

Constraints / Preferences / Non-goals
- Preserve intended graceful shutdown and drain semantics.
- Keep the change scoped; do not widen into unrelated lifecycle cleanup.
- Treat race/integration behavior as part of the verification story.

Acceptance Criteria / Expected Outcome
- The root cause is identified clearly enough to support a focused fix.
- The fix preserves or improves shutdown/readiness semantics.
- Flaky behavior is covered by targeted regression proof, not just a broader timeout.

Validation / Verification
- Run the narrowest relevant package tests first.
- Run `make test-race` for concurrency-sensitive shutdown changes.
- Run `make test-integration` if the affected behavior crosses the integration boundary.

Assumptions / Open Questions
- [assumption] The flake is most likely in bootstrap or readiness/drain coordination, but the exact failing test is not yet identified.
```
