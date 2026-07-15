---
name: go-test-implementation
description: "Use when approved behavior or test design is ready for executable Go proof; Own the smallest deterministic test code and fixtures covering required success and failure paths; Skip when production behavior must change, proof strategy is unresolved, existing tests need review, or claims need closeout verification."
---

# Go Test Implementation

## Accepted Input And Boundary

Implement tests only from accepted behavior and, when needed, an accepted proof strategy: a direct outcome, task, spec, test plan, bug reproducer, or concrete defect. Read the touched code, callers, nearby tests, fixtures, and repository-owned commands. Do not change production behavior, weaken a gate, or invent product, API, data, security, reliability, rollout, ownership, or proof policy in tests. Stop at the exact missing owner when expected behavior or the proving layer is unresolved or contradictory.

## Method

1. List only the triggered obligations, including material fail, edge, cancellation, concurrency, tenant, data, cache, and boundary behavior.
2. Choose the smallest layer that can observe the real regression: unit for local logic, contract for transport semantics, integration for real storage/cache/process seams, fuzz for input-heavy invariants, and benchmark only for an approved performance obligation.
3. Derive independent oracles from accepted behavior, not implementation branches. Assert result, relevant state, required effects, forbidden side effects, and durable error categories; use `errors.Is` or `errors.As` when wrapping is contractual and exact text only when public.
4. Reuse repository test style before adding helpers. Keep scenario intent visible, fixtures minimal and truthful, cleanup observable, and helper logic independent of the implementation under test.
5. Control time, randomness, environment, external dependencies, cleanup, and goroutine coordination. Replace timing guesses with deterministic signals; do not normalize flaky timing, shared-state coupling, leaks, or skipped proof.
6. Run the focused fresh command, then the smallest broader repository gate justified by the affected surface. Add race-aware execution only when the scenario exercises concurrency-sensitive code.
7. Reject critical omissions, permissive assertions, pass-by-no-panic tests, unobserved goroutine or side-effect outcomes, dishonest goldens, and skips or exclusions that make a gate pass without proving the obligation.

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

## Proof, Return, And Stop

Return changed test and fixture files, behavior-level obligation coverage, commands with observed results, forbidden-side-effect proof, and any unresolved behavior or proof gap. Return success only when every critical obligation has passing proof, the tests are deterministic and reviewable, and every positive claim is bounded by commands actually run. Otherwise report the exact unresolved owner without claiming readiness.
