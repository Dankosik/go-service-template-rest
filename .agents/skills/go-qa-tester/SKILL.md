---
name: go-qa-tester
description: "Implement deterministic Go tests from approved requirements and test obligations with strong fail-path coverage, invariant traceability, and review-clean evidence. Use when adding or upgrading Go tests after behavior is already approved, especially for API, data, cache, security, or concurrency changes where the hard part is choosing the smallest proving layer and encoding reliable negative-path coverage."
---

# Go QA Tester

## Purpose
Implement executable, deterministic Go tests that prove approved behavior, expose regressions early, and make changed Go code easier to trust in review and handoff.

## Scope
- implement and maintain unit, integration, contract, fuzz, benchmark, and example tests when they are the smallest honest proof for approved behavior
- translate requirements, invariants, fail paths, and bug-reproduction intent into explicit assertions
- choose the smallest sufficient proving layer and keep scenario coverage risk-based
- keep tests deterministic, isolated, readable, and diagnostically useful
- run relevant repository validation commands and report factual outcomes

## Boundaries
Do not:
- redesign product behavior, API contracts, storage semantics, rollout policy, or domain transitions in tests
- hide unclear requirements behind permissive assertions, brittle helper magic, or implementation mirroring
- prefer broad end-to-end coverage when a smaller layer can prove the same obligation more reliably
- normalize flaky timing, shared-state coupling, or nondeterministic failures as acceptable noise
- create or repair workflow, research, specification, design, or planning artifacts from this skill
- claim readiness when critical scenarios remain unimplemented, flaky, or unverified

If expected behavior is missing or contradictory, stop and name the smallest unblock question. Keep file paths proposed until repository evidence confirms them.

## Implementation Workflow
1. **Load approved behavior.** Read the task artifact, spec, plan, test plan, bug report, or review finding that defines the behavior. Also read touched code and nearby tests.
2. **Build the obligation list.** Name each behavior to prove, including happy, fail, edge, cancellation, concurrency, tenant, cache, data, or boundary cases that are actually triggered by the approved change.
3. **Choose the proving layer.** Prefer the smallest layer that can catch the real regression without inventing semantics. Use unit tests for local logic, contract tests for transport semantics, integration tests for storage/cache/process seams, fuzz tests for input-heavy code, and benchmarks only for performance obligations.
4. **Implement deterministic tests.** Control time, randomness, environment, external dependencies, cleanup, and goroutine coordination explicitly. Use repository-local test style before introducing new helper patterns.
5. **Tighten assertions.** Assert observable behavior and durable error categories, not incidental implementation details. Use `errors.Is` or `errors.As` when wrapping is part of the contract. Avoid string comparisons unless exact text is public behavior.
6. **Verify honestly.** Run the narrow command that proves the changed test surface, then the smallest broader repository command justified by risk. For concurrency-sensitive work, include race-aware execution when available.
7. **Report concretely.** Name changed test files, scenario coverage, executed commands, observed results, and any unresolved behavior or validation risk.

## Load Repository Guidance
Load only the local guidance that matches the changed surface:
- always useful for test work: `docs/build-test-and-development-commands.md`, touched code, nearby tests, and any approved task artifact that defines expected behavior
- transport or client-visible API behavior: `api/openapi/service.yaml`, `internal/api/README.md`, generated handler integration, and existing `internal/infra/http/*_test.go`
- SQL, cache, or migration behavior: touched repository/query/cache code, `env/migrations/`, `test/*integration*_test.go`, and existing data tests
- goroutines, channels, worker pools, shutdown, context, or time: touched runtime code, lifecycle tests under `cmd/service/internal/bootstrap`, and any approved cancellation/shutdown expectation
- auth, tenant isolation, or trust-boundary behavior: touched auth or boundary code and the approved security expectation

## Lazy Reference Selector
Read only the files needed for the current test task:

| Load when... | Reference |
| --- | --- |
| you need to turn requirements, invariants, or bug reports into named test cases | `references/obligation-to-test-translation.md` |
| you must decide unit vs integration vs contract vs fuzz vs benchmark/example coverage | `references/minimal-proving-layer-selection.md` |
| you are writing ordinary Go tests, helpers, subtests, fixtures, fuzz seeds, or examples | `references/go-test-construction-patterns.md` |
| goroutines, channels, shutdown, backpressure, timers, deadlines, races, or `testing/synctest` are involved | `references/deterministic-concurrency-and-time-tests.md` |
| API, SQL, migrations, cache, idempotency, tenant isolation, or repository integration behavior is involved | `references/api-data-cache-test-patterns.md` |
| errors, wrapping, context propagation, cancellation, deadline behavior, or fail-fast lifecycle is involved | `references/error-context-and-cancellation-tests.md` |
| you need command selection, race/fuzz/integration evidence, or final reporting wording | `references/verification-command-evidence.md` |

The references contain examples and Exa source links gathered from Go primary docs plus repository-local test patterns. If a new example depends on a newer Go feature or external library behavior not covered there, refresh that source with Exa before relying on it.

## Core Test Discipline
- Obligations first: test what must be true, not what is easiest to assert.
- Preserve approved semantics rather than mirroring implementation structure.
- Prefer explicit Go assertions and actionable failure messages over opaque assertion frameworks.
- Use `t.Helper()` in helpers and keep helper layers thin enough that scenario intent stays visible.
- Use `t.Run` and table-driven tests when shared structure improves clarity.
- Use `t.Parallel()` only when isolation and shared-resource safety are clear.
- Use `t.Setenv`, `t.TempDir`, and `t.Cleanup` for local resource control where appropriate.
- Prefer deterministic coordination primitives over timing sleeps and polling luck.
- Treat critical omitted scenarios as defects, not documentation debt.

## Inference Discipline
- Lock assertions to approved behavior and observable contract, not guessed implementation mechanics.
- If a request must be rejected, deduplicated, retried, or treated as equivalent but does not pin the exact transport mapping, test the semantic behavior and escalate the missing status/header/payload detail before making it exact.
- Do not invent cache key dimensions, replay status codes, resume-checkpoint strategy, diagnostics fields, config symbols, package layout, tenant rules, or file ownership unless approved docs or current repository code already pin them.
- When duplicate suppression matters, assert the observable side effect stays single, but do not assume a reservation, fingerprint, or dedup-storage model unless it already exists.
- When the implementation surface does not exist yet, mark placements and validation commands as proposed or conditional.

## Review-Clean Bar
Before handoff, check the tests against likely review axes:
- QA: critical obligations and fail paths are actually covered
- idiomatic Go: structure, helpers, error checks, cleanup, and subtests are clear
- simplification: scenario intent is obvious on first read
- domain: invariants and transitions are proven, not assumed
- reliability/concurrency: timeouts, cancellation, shutdown, and race risks are exercised where relevant
- security: denial and trust-boundary behavior is explicit where relevant
- DB/cache: transaction, pagination, migration, cache correctness, and degradation behavior are exercised where relevant
- performance: tests do not normalize obviously wasteful hot-path behavior

Tighten predictable review findings before handoff.

## Handoff Notes
When reporting test work, include:
- implemented test scope and key test files
- scenario coverage by behavior, not vague "more tests" wording
- validation commands and observed results
- design escalations or residual risks, if any

## Escalate When
Escalate when:
- expected behavior is unclear or contradictory (`go-domain-invariant-spec`, `go-design-spec`, or `api-contract-designer-spec`)
- reliability, cancellation, retry, or distributed semantics are missing (`go-reliability-spec` or `go-distributed-architect-spec`)
- data ownership, cache correctness, migration expectations, or tenant-scoping rules are unresolved (`go-data-architect-spec` or `go-db-cache-spec`)
- authorization or trust-boundary behavior is unclear (`go-security-spec`)
- observability or performance expectations are needed before tests can be precise (`go-observability-engineer-spec` or `go-performance-spec`)
- the testing strategy itself is unclear or missing (`go-qa-tester-spec`)
