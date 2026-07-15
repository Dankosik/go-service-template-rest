# Reference Selector

Load at most one reference by default; load more only for independent pressures.

| Pressure | Load |
| --- | --- |
| Current Go or stdlib may replace a helper, dependency, or old idiom. | [stdlib-first-modern-go.md](stdlib-first-modern-go.md) |
| Helpers, interfaces, package moves, exports-for-tests, or repeated policy affect ownership. | [helper-extraction-and-package-ownership.md](helper-extraction-and-package-ownership.md) |
| Decoding, limits, unknown fields, normalization, or validation order changes. | [boundary-decoding-and-validation.md](boundary-decoding-and-validation.md) |
| Wrapped errors, cancellation, transport mapping, or log-and-return behavior changes. | [errors-context-and-boundary-mapping.md](errors-context-and-boundary-mapping.md) |
| Bodies, rows, files, transactions, cursor errors, or post-commit effects change. | [resource-lifetime-io-and-transactions.md](resource-lifetime-io-and-transactions.md) |
| Slices, maps, bytes, snapshots, caches, nil/empty shape, or mutex-bearing values cross owners. | [mutable-state-aliasing.md](mutable-state-aliasing.md) |
| Goroutines, channels, fan-out, worker pools, shutdown, timers, or tickers change. | [concurrency-and-background-work.md](concurrency-and-background-work.md) |
| Tests, fuzzing, clocks, randomness, failure messages, or verification shape changes. | [testing-verification-patterns.md](testing-verification-patterns.md) |
| OpenAPI, sqlc, protobuf, mocks, generated files, config, or mirrors change. | [generated-source-of-truth-and-drift.md](generated-source-of-truth-and-drift.md) |
