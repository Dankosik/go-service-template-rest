# Example Transformations

These examples show the shortest routing-sufficient prompt for each fixture.

## Example 1: HTTP OPTIONS / CORS Policy Bug

Raw fixture: `files/http-options-cors.md`

```md
Fix and regression-test HTTP `OPTIONS` handling: a known non-preflight route returns `204 No Content` with the correct `Allow` header, while disabled CORS preflight remains fail-closed. Preserve intentional `problem+json` behavior; OpenAPI and generated bindings stay unchanged.

Inspect `internal/infra/http/router.go` and `internal/infra/http/router_contract_test.go`. Current router ownership makes this reversible, one-owner work with focused `./internal/infra/http` tests as bounded proof. Reopen if inspection disproves that ownership, contract neutrality, or proof boundary.

Path: `direct` -> Implementation.
```

## Example 2: Repo-Local Skill / Prompt Tooling Request

Raw fixture: `files/skill-tooling.md`

```md
Revise `.agents/skills/agent-prompt-composer/` so arbitrary messy input becomes the shortest routing-sufficient English Intake prompt, not a translation. Canonical Intake owns the contract; preserve exact identifiers and add repository context only when it changes routing. Keep the skill and useful examples aligned, create no global or home-directory skill, and validate with `git diff --check`.

Path: `direct` -> Implementation.
```

## Example 3: Flaky Shutdown / Drain Investigation

Raw fixture: `files/flaky-shutdown.md`

```md
Find and fix the root cause of the flaky shutdown/drain path where `context canceled` may be swallowed or a worker may fail to stop, hanging the test. Preserve graceful shutdown and readiness; do not raise the timeout unless evidence makes timeout behavior the contract. Start with `cmd/service/internal/bootstrap/`, `internal/health/service.go`, and nearby lifecycle tests. Success requires targeted regression proof plus race or integration proof when the affected boundary demands it.

If bounded inspection cannot identify the failing owner or reproducer, stop with the evidence and smallest next diagnostic target.

Path: `structured` -> Research, because the lifecycle cause and proof boundary remain open.
```

## Example 4: Ready Native Orchestrator Entry

```text
$orchestrator
Use specs/category-mapping-knn-first/tasks.md. Stop before live rollout.
```

The native skill and ledger own role behavior, workflow, accepted decisions,
proof, and current state; the prompt carries only the locator and missing stop
delta.
