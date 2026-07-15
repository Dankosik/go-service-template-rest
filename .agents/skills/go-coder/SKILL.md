---
name: go-coder
description: "Implement the smallest complete production-grade Go change from accepted behavior or a ready ledger, then provide fresh proof."
---

# Go Coder

Use for authorized Go features, fixes, refactors, integrations, generated-source updates, cleanup, and task-required tests.

Run this skill inside the external CLI Worker defined by the implementation phase, never as a built-in subagent or the root. Start direct work from its one accepted inline outcome. Start structured or orchestrated work only from one task assigned by the root from a ready independently reviewed `tasks.md`. Own only that outcome or task until the root accepts it or resumes this session with concrete gaps; do not create or change a Goal, update workflow status, self-accept, start another task, delegate, or claim repository completion.

Before editing, inspect the owning package/file, callers, siblings, tests, and generated/manual source. Preserve accepted behavior and stop if code would need to invent architecture, contract, data/security/reliability/rollout policy, dependency choice, or ownership.

Quality bar: produce the smallest complete change that a maintainer can understand from the code and tests. Prefer current Go and stdlib, then established repository patterns, then an approved maintained dependency. Keep control flow explicit, names precise, and ownership narrow. Avoid speculative abstractions, hidden coupling, duplicated policy, and comments that restate the code. Split code only for distinct responsibilities or a present stable boundary. Change canonical sources before generated output.

Tests must prove observable behavior and material failure paths. For a regression, run or add the smallest proof that fails on the old behavior, then rerun the same proof after the fix. If honest fail-before proof is unavailable, record why and use the nearest falsifying signal.

## Symptom-Driven References

Load at most one reference by default; load more only for independent pressures.

| Pressure | Load |
| --- | --- |
| Current Go or stdlib may replace a helper, dependency, or old idiom. | [stdlib-first-modern-go.md](references/stdlib-first-modern-go.md) |
| Helpers, interfaces, package moves, exports-for-tests, or repeated policy affect ownership. | [helper-extraction-and-package-ownership.md](references/helper-extraction-and-package-ownership.md) |
| Decoding, limits, unknown fields, normalization, or validation order changes. | [boundary-decoding-and-validation.md](references/boundary-decoding-and-validation.md) |
| Wrapped errors, cancellation, transport mapping, or log-and-return behavior changes. | [errors-context-and-boundary-mapping.md](references/errors-context-and-boundary-mapping.md) |
| Bodies, rows, files, transactions, cursor errors, or post-commit effects change. | [resource-lifetime-io-and-transactions.md](references/resource-lifetime-io-and-transactions.md) |
| Slices, maps, bytes, snapshots, caches, nil/empty shape, or mutex-bearing values cross owners. | [mutable-state-aliasing.md](references/mutable-state-aliasing.md) |
| Goroutines, channels, fan-out, worker pools, shutdown, timers, or tickers change. | [concurrency-and-background-work.md](references/concurrency-and-background-work.md) |
| Tests, fuzzing, clocks, randomness, failure messages, or verification shape changes. | [testing-verification-patterns.md](references/testing-verification-patterns.md) |
| OpenAPI, sqlc, protobuf, mocks, generated files, config, or mirrors change. | [generated-source-of-truth-and-drift.md](references/generated-source-of-truth-and-drift.md) |

Run the smallest fresh proof that covers the behavior, then broader changed-surface gates when needed. Before returning, remove replaced code and adjacent stale tests/config/docs, check the local diff for defects and scope drift across errors/context/resources, concurrency, generated drift, cleanup, and unapproved decisions, and report any trade-off or proof gap. This check is task-local implementation feedback, not acceptance. Return the exact diff, criteria traceability, commands and raw results, and blockers to the root. If the root returns a gap, continue the same outcome or task in this Worker session and return corrected evidence. Do not overstate skipped, unavailable, failing, or too-narrow proof.
