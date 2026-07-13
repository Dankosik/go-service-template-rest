---
name: go-qa-tester
description: "Implement the smallest deterministic Go test set that proves approved behavior and its material fail paths."
---

# Go QA Tester

Use for authorized Go test implementation after behavior and any required test strategy are approved. Follow [implementation/validation/closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md).

Treat the accepted direct outcome, task, spec, test plan, bug reproducer, or review finding as authority. Read the touched code, nearby tests, and repository-owned commands. Do not redesign product, contract, storage, security, reliability, or rollout semantics in tests. If expected behavior is missing or contradictory, stop and reopen its smallest owner.

## Method

1. List only the triggered obligations, including material fail, edge, cancellation, concurrency, tenant, data, cache, or boundary behavior.
2. Choose the smallest layer that can observe the real regression: unit for local logic, contract for transport semantics, integration for real storage/cache/process seams, fuzz for input-heavy invariants, and benchmark only for an approved performance obligation.
3. Reuse repository test style before adding helpers. Keep scenario intent visible and avoid implementation mirroring.
4. Control time, randomness, environment, external dependencies, cleanup, and goroutine coordination. Do not normalize flaky timing or shared-state coupling.
5. Assert observable behavior, forbidden side effects, and durable error categories. Use `errors.Is` or `errors.As` when wrapping is contractual; compare exact text only when it is public behavior.
6. Run the focused fresh command, then the smallest broader repository gate justified by the affected surface. Add race-aware execution only when the scenario exercises concurrency-sensitive code.
7. Report changed test files, behavior-level coverage, commands and observed results, plus any unresolved behavior or proof gap.

Critical omitted obligations, permissive assertions, pass-by-no-panic tests, and unobserved goroutine or side-effect outcomes are defects. Apply the phase proof-integrity rule to every test, fixture, golden, skip/exclusion, and validation-command change.

## Symptom-Driven References

Load at most one reference by default; load more only for independent test pressures.

| Pressure | Load |
| --- | --- |
| Approved requirements, invariants, review findings, or bug notes must become named scenarios. | [obligation-to-test-translation.md](references/obligation-to-test-translation.md) |
| The proving layer is unclear. | [minimal-proving-layer-selection.md](references/minimal-proving-layer-selection.md) |
| Ordinary Go test structure, helpers, fixtures, fuzz seeds, benchmarks, or examples are the main risk. | [go-test-construction-patterns.md](references/go-test-construction-patterns.md) |
| Goroutines, shutdown, backpressure, timers, deadlines, race detection, or `testing/synctest` are involved. | [deterministic-concurrency-and-time-tests.md](references/deterministic-concurrency-and-time-tests.md) |
| Wrapped errors, context propagation, cancellation, deadlines, or fail-fast behavior are involved. | [error-context-and-cancellation-tests.md](references/error-context-and-cancellation-tests.md) |
| HTTP, OpenAPI, strict parsing, idempotency, headers, CORS/fallback, or async resources are involved. | [api-contract-test-patterns.md](references/api-contract-test-patterns.md) |
| SQL, migrations, transactions, tenant isolation, cache behavior, or Docker-gated integration is involved. | [data-cache-integration-test-patterns.md](references/data-cache-integration-test-patterns.md) |
| Final command selection, race/fuzz/integration evidence, or validation wording is the main risk. | [verification-command-evidence.md](references/verification-command-evidence.md) |

If a reference would not change the test decision, do not load it. Verify decisive newer Go or dependency behavior against the active toolchain or current official primary docs.

Return success only when every critical obligation has passing proof, the tests are deterministic and reviewable, and every positive claim is bounded by commands actually run. Otherwise report the exact unresolved owner without claiming readiness.
