# Reference Selector

Load at most one reference by default; load more only for an independent pressure.

| Pressure | Load | Required effect |
| --- | --- | --- |
| An OpenAPI, SQL, or protobuf source, its generated output, or a drift-check failure changes. | [generated-source-of-truth-and-drift.md](generated-source-of-truth-and-drift.md) | Edit the canonical source, run that source's `*-generate`, and prove with its `*-check` — instead of hand-editing generated Go or treating the drift check as the generator. |
| The change adds a file, moves code, or the obvious edit site would need a new import. | [earliest-owner-and-import-direction.md](earliest-owner-and-import-direction.md) | Land the change in the package `depguard` already lets import what it needs — instead of at the file where the symptom appeared. |
| Adjacent code looks modernizable, or a stdlib call could replace a local helper. | [gates-and-policy-ownership.md](gates-and-policy-ownership.md) | Leave mechanical idiom to `make lint-all` and `modernize-check`, and check a suggested swap against the policy the old code carried — instead of widening the diff or dropping a semantic. |
| A pure generic slice or map transformation would add a loop or helper. | [collection-transformations.md](collection-transformations.md) | Use one clear stdlib call first, then the installed `lo` operation; keep policy explicit. |

Errors, context, nil/zero, receivers, aliasing, and resource lifetime belong to
`go-idiomatic`; goroutines, channels, and shutdown to `go-concurrency`; test
shape and oracles to `go-test-implementation`. The phase composes those as review
lenses over the same diff, so this selector does not restate them.
