# Reference Selector

Load at most one reference by default; load more only for an independent pressure.

| Pressure | Load | Required effect |
| --- | --- | --- |
| An OpenAPI, SQL, or protobuf source, its generated output, or a drift-check failure changes. | [generated-source-of-truth-and-drift.md](generated-source-of-truth-and-drift.md) | Identify the target's canonical source and generator, then establish generated consistency through the selected proof plan. |
| The change adds a file, moves code, or the obvious edit site would need a new import. | [earliest-owner-and-import-direction.md](earliest-owner-and-import-direction.md) | Place the change using the target's accepted package boundaries and current dependency rules. |
| Adjacent code looks modernizable, or a stdlib call could replace a local helper. | [gates-and-policy-ownership.md](gates-and-policy-ownership.md) | Resolve the target module's idioms and local lint policy while preserving the old code's observable semantics. |
| A pure generic slice or map transformation would add a loop or helper. | [collection-transformations.md](collection-transformations.md) | Use one clear stdlib call first, then the installed `lo` operation; keep policy explicit. |

Errors, context, nil/zero, receivers, aliasing, and resource lifetime belong to
`go-idiomatic`; goroutines, channels, and shutdown to `go-concurrency`; test
shape and oracles to `go-test-implementation`. The phase composes those as review
lenses over the same diff, so this selector does not restate them.
