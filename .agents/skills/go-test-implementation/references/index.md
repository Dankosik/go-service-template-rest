# Reference Selector

Load at most one reference by default; load more only for independent test pressures.

| Pressure | Load |
| --- | --- |
| Approved requirements, invariants, review findings, or bug notes must become named scenarios. | [obligation-to-test-translation.md](obligation-to-test-translation.md) |
| The proving layer is unclear. | [minimal-proving-layer-selection.md](minimal-proving-layer-selection.md) |
| Ordinary Go test structure, helpers, fixtures, fuzz seeds, benchmarks, or examples are the main risk. | [go-test-construction-patterns.md](go-test-construction-patterns.md) |
| Goroutines, shutdown, backpressure, timers, deadlines, race detection, or `testing/synctest` are involved. | [deterministic-concurrency-and-time-tests.md](deterministic-concurrency-and-time-tests.md) |
| Wrapped errors, context propagation, cancellation, deadlines, or fail-fast behavior are involved. | [error-context-and-cancellation-tests.md](error-context-and-cancellation-tests.md) |
| HTTP, OpenAPI, strict parsing, idempotency, headers, CORS/fallback, or async resources are involved. | [api-contract-test-patterns.md](api-contract-test-patterns.md) |
| SQL, migrations, transactions, tenant isolation, cache behavior, or Docker-gated integration is involved. | [data-cache-integration-test-patterns.md](data-cache-integration-test-patterns.md) |
| Final command selection, race/fuzz/integration evidence, or validation wording is the main risk. | [verification-command-evidence.md](verification-command-evidence.md) |

If a reference would not change the test decision, do not load it. Verify decisive newer Go or dependency behavior against the active toolchain or current official primary docs.
