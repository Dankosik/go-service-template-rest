# Example Transformations

These examples show compact, routing-sufficient Intake briefs. They do not use every possible heading.

## Example 1: HTTP OPTIONS / CORS Policy Bug

Raw fixture: `files/http-options-cors.md`

```md
Objective
Fix and regression-test HTTP `OPTIONS` handling so a known non-preflight route returns `204 No Content` with the correct `Allow` header, while CORS preflight remains fail-closed when CORS is disabled.

Success criteria
- Focused router tests cover both branches and preserve intentional `problem+json` behavior.
- OpenAPI and generated bindings remain unchanged.

Constraints and evidence
- Bounded repository inspection established that existing router composition and contract tests own both `OPTIONS` branches, OpenAPI owns neither fallback, and no client-visible contract decision remains open.
- The change is reversible, has one package owner, and focused router tests provide bounded validation.
- Inspect `internal/infra/http/router.go` and `internal/infra/http/router_contract_test.go` first.
- Preserve the exact signals `OPTIONS`, `Allow`, `preflight`, `CORS`, and `problem json`.
- Run focused `./internal/infra/http` tests.

Reopen the smallest owner if new repository evidence invalidates the recorded router ownership, no-contract-delta fact, or bounded proof.

Path / first owner
`direct` -> Implementation with the accepted outcome and routing facts above.
```

## Example 2: Repo-Local Skill / Prompt Tooling Request

Raw fixture: `files/skill-tooling.md`

```md
Revise the existing repository-local prompt-composer skill so it reconstructs arbitrary messy user intent into one routing-sufficient English brief governed by canonical Intake.

Success means canonical Intake owns the output contract, the `.agents/skills/` source and useful examples agree, the skill preserves exact identifiers, and it adds repository context only when it changes routing.

Inspect `.agents/skills/agent-prompt-composer/`. Keep dependencies minimal. Validate with `git diff --check`. Do not create a global or home-directory skill.

Path / first owner
`direct` -> Implementation in `.agents/skills/agent-prompt-composer/`, including alignment of any disclosed example that still describes a separate handoff contract.
```

## Example 3: Flaky Shutdown / Drain Investigation

Raw fixture: `files/flaky-shutdown.md`

```md
Objective
Find and fix the root cause of the flaky shutdown/drain path where `context canceled` may be swallowed or a worker may fail to stop, causing a test hang.

Success criteria
- The fix preserves graceful shutdown and readiness semantics.
- Targeted regression proof covers the failing branch; race or integration proof is included when the affected boundary requires it.

Inspect/evidence
- Start with `cmd/service/internal/bootstrap/`, `internal/health/service.go`, and nearby shutdown/readiness tests.
- Preserve the exact signals `context canceled`, `shutdown`, `drain`, `worker`, `race`, and `integration`.
- Do not use a timeout increase as the primary fix without evidence that timeout behavior is the contract.

If bounded inspection cannot identify the failing owner or reproducer, stop with the evidence gathered and the smallest next diagnostic target.

Path / first owner
`structured` -> Research with the open question: identify the failing owner and reproducer and determine which observed lifecycle boundary can change the eventual fix. `direct` is insufficient while the concurrency or lifecycle cause and proof boundary remain unresolved.
```
