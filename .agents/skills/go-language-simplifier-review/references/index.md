# Reference Selector

Use broad false-simplification triage only when no narrower selector owns the pressure.

| Pressure | Load |
| --- | --- |
| Broad cleanup, DRY, deduplication, or readability claim spans several local axes. | [false-simplification-patterns.md](false-simplification-patterns.md) |
| Helpers, wrappers, interfaces, callbacks, option bags, or helper buckets changed. | [helper-extraction-economics.md](helper-extraction-economics.md) |
| Stable same-package policy is repeated, drifting, or moved away from its local owner. | [source-of-truth-extraction.md](source-of-truth-extraction.md) |
| Branches, sentinels, named returns, defer, cleanup, rollback, audit, or phase order changed. | [control-flow-and-temporal-coupling.md](control-flow-and-temporal-coupling.md) |
| Predicates, negatives, flags, modes, same-typed args, or option decoding obscure a decision. | [predicate-condition-and-mode-clarity.md](predicate-condition-and-mode-clarity.md) |
| Error handling was deduplicated, normalized, mapped, logged, joined, or reordered. | [error-path-simplification.md](error-path-simplification.md) |
| Tables, helpers, assertions, fixtures, or terse failures obscure test proof intent. | [test-readability-and-proof-shape.md](test-readability-and-proof-shape.md) |
| Names or vocabulary obscure role, phase, ownership, or policy with merge risk. | [naming-and-intent-exposure.md](naming-and-intent-exposure.md) |
| Cleanup touches alias isolation, nil/empty, receivers, zero values, lifetime, cleanup, or stdlib contracts. | [go-semantic-stop-signs.md](go-semantic-stop-signs.md) |
