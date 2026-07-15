---
name: go-coder
description: "Use when an authorized Go outcome is accepted and ready to implement; Own the smallest complete production change, task-required tests, cleanup, and fresh proof; Skip when behavior or ownership is unresolved, or the request is diagnosis-only, test-only, or verification-only."
---

# Go Coder

## Accepted Input And Boundary

[Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) owns Worker assignment, lifecycle, acceptance, and integration. This skill owns only the assigned Go outcome: the smallest complete feature, fix, refactor, integration, generated-source update, cleanup, or task-required tests. Inspect the owning package and files, callers, sibling behavior, nearby tests, repository commands, and generated/manual source before editing. Stop at the exact missing owner if implementation would require inventing architecture, product or API behavior, data, security, reliability, rollout, dependency, or ownership policy.

## Method

1. Quality bar: produce the smallest complete change that a maintainer can understand from the code and tests. Prefer the active module's Go and standard library, then established repository patterns, then an approved maintained dependency.
2. Avoid speculative abstractions, hidden coupling, duplicated policy, and comments that restate the code. Keep control flow explicit, names precise, and ownership narrow; split or extract only for a present repeated policy, distinct responsibility, or stable boundary.
3. Preserve inspectable errors, caller context and cancellation, transaction and resource lifetime, concurrency ownership and joins, mutable-data ownership, and boundary mappings whenever the changed path touches them.
4. Change canonical sources before generated output and keep required generated artifacts in sync. Do not mix unrelated generated drift into the outcome.
5. Tests must prove observable behavior and material failure paths. Prove result, state, and effects at the smallest reliable layer. For a regression, run or add the smallest proof that fails on the old behavior, then rerun the same proof after the fix; when honest RED proof is unavailable, state why and use the nearest falsifying signal.
6. Leave no temporary diagnostics or compatibility path without current evidence.

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

## Proof, Return, And Stop

Run the smallest fresh proof that covers the behavior, then the broader gates triggered by the changed surface. Before returning, remove replaced code and adjacent stale tests/config/docs, check the local diff for defects and scope drift across errors/context/resources, concurrency, generated drift, cleanup, and unapproved decisions, and report any trade-off or proof gap. Return changed files, accepted-criteria traceability, commands with observed results, and blockers. Do not overstate skipped, unavailable, failing, cached, or too-narrow evidence; if a required decision is unresolved, name its owner and stop without a speculative patch.
