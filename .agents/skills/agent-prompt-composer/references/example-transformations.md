# Example Transformations

These examples show compact, conditional handoffs. They do not use every possible heading.

## Example 1: HTTP OPTIONS / CORS Policy Bug

Raw fixture: `evals/files/http-options-cors.md`

```md
Objective
Fix and regression-test HTTP `OPTIONS` handling so a known non-preflight route returns `204 No Content` with the correct `Allow` header, while CORS preflight remains fail-closed when CORS is disabled.

Success criteria
- Focused router tests cover both branches and preserve intentional `problem+json` behavior.
- OpenAPI and generated bindings remain unchanged unless bounded inspection proves the public contract changes.

Constraints and evidence
- Inspect `internal/infra/http/router.go` and `internal/infra/http/router_test.go` first.
- Preserve the exact signals `OPTIONS`, `Allow`, `preflight`, `CORS`, and `problem json`.
- Run focused `./internal/infra/http` tests; run `make openapi-check` only if contract-owned files change.

Stop if the behavior is owned outside router policy or requires a public-contract decision; report the owner and required reopen instead of broadening the patch.
```

## Example 2: Repo-Local Skill / Prompt Tooling Request

Raw fixture: `evals/files/skill-tooling.md`

```md
Revise the existing repository-local prompt-composer skill so it reconstructs arbitrary messy user intent and emits a compact English coding-agent handoff rather than performing literal translation or generic prompt polish.

Success means the canonical `.agents/skills/` source and useful examples/evals agree; the skill preserves exact identifiers, adds repository context only when it changes execution, and remains local to this repository.

Inspect `.agents/skills/agent-prompt-composer/`. Keep dependencies minimal. Validate with `git diff --check`. Do not create a global or home-directory skill.
```

## Example 3: Flaky Shutdown / Drain Investigation

Raw fixture: `evals/files/flaky-shutdown.md`

```md
Objective
Find and fix the root cause of the flaky shutdown/drain path where `context canceled` may be swallowed or a worker may fail to stop, causing a test hang.

Success criteria
- The fix preserves graceful shutdown and readiness semantics.
- Targeted regression proof covers the failing branch; race or integration proof is included when the affected boundary requires it.

Inspect/evidence
- Start with `cmd/service/internal/bootstrap/`, `internal/app/health/service.go`, and nearby shutdown/readiness tests.
- Preserve the exact signals `context canceled`, `shutdown`, `drain`, `worker`, `race`, and `integration`.
- Do not use a timeout increase as the primary fix without evidence that timeout behavior is the contract.

If bounded inspection cannot identify the failing owner or reproducer, stop with the evidence gathered and the smallest next diagnostic target.
```
