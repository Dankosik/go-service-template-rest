# Go Validation

During ledger implementation, use only the coding feedback allowed by
[Implementation](../spec-first-workflow/phases/implementation.md#feedback-during-coding).
For affected packages, `gopls check` or compile-only `go test -c` with the required
tags can expose type errors in production and test code. Keep binary outputs
outside source paths. `go test -run '^$'` is not compile-only: it can execute
package initialization and TestMain. Neither diagnostic establishes behavior.
`gopls references file.go:line:column` locates callers for semantic renames or
signature changes; it does not test or accept the code.

For standalone debugging or focused repair during final validation,
`make prove PKG=... FILES='...'`
is the lock-wrapped package check. Both fields are required; there is no
`./...` default. `make test-watch` and `make lint-fast` are also diagnostic
commands, not steps in ledger execution or substitutes for final evidence.

The integrated delivery owner starts with a matching build and relevant
unit tests on the assembled delivery candidate, not once per ledger unit.
[Evidence Contract](../spec-first-workflow/shared/evidence-contract.md#local-completion)
owns local completion and explicit additions. Task handoffs stay `Implemented`.
For the main service and the ordinary root-module test suite:

```bash
make build
ALLOW_FULL=1 make test-all
```

`make build` builds the service, not every worker. Use the existing
`build-worker`, `build-outbox-relay`, or `build-jobs-worker` targets when those
retained executables are part of the change. Include required build variants;
do not invent a matrix of template profiles.

For a bounded change, use `make test-package PKG=./path/to/package` and relevant
reverse importers selected through existing repository tooling. `make plan` may
diagnose that scope without executing its expanded plan. Do not build a new
package-selection mechanism. Use `test-all` when root `go.mod`/`go.sum` change,
the relevant closure is broad, or the package graph cannot reliably bound it.
A separate Go module needs its own relevant checks.

`ALLOW_FULL=1` on `test-all` permits the ordinary module-wide test suite; it does
not select `make check`, race, integration tags, or Docker. Confirm the required
tests actually ran or have valid reusable results. A zero-test selector, skipped
required test, or startup error is not passing behavior evidence.

Stop after the local criterion and any applicable final review pass. Formatting
and required generation remain normal implementation work. Do not append lint,
`make verify`, integration, race, runtime-image, or performance runs merely for
confidence. Existing CI gates remain unchanged.

Use `make verify` only for an explicitly selected expanded verification scope.
It formats changed handwritten files, lints changed packages, tests the
module-local reverse importer closure, and can select non-Go and heavy leaves.
Continue a failed plan through the Evidence Contract rather than automatically
rerunning its aggregate. Reuse valid scoped results after repair.
`ALLOW_FULL=1 make check` remains the explicit full formatting, `lint-all`,
`test-all`, root-module tidy, and generated-drift gate; do not run its leaves
beside it.

`make lint-changed` accepts one `PKG` or a space-separated `PKGS` batch;
`make test-package` requires `PKG` and remains package-scoped.
`lint-pr` is the PR correctness, architecture, resource, error, context, and
interface set and requires `PKG` or `PKGS`. Full-module leaves are `lint-all`
and `test-all`; they require `ALLOW_FULL=1` or `CI=true`. `make lint-deep`, `make test-race`,
`make test-integration`, and `make audit-full-manual` require `ALLOW_HEAVY=1`
or actual CI. These flags do not authorize scope expansion; never impersonate CI.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` only when a required race or environment claim needs fresh execution.
Final validation uses `VALIDATION_JOBS=2` and one Git-common process lock by
default; CI raises the shared budget explicitly.
