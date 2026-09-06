# Go Validation

During ledger implementation, navigate source and references without running
validation commands, including `gopls check`, tests, lint, and watch mode.
`gopls references file.go:line:column` locates callers for semantic renames or
signature changes; it does not test or accept the code.

For standalone debugging outside a ledger, `make prove PKG=... FILES='...'`
is the lock-wrapped package check. Both fields are required; there is no
`./...` default. `make test-watch` and `make lint-fast` are also diagnostic
commands, not steps in ledger execution or substitutes for final evidence.

The integrated delivery owner runs `make verify` once on the final delivery
candidate, not once per ledger unit. Task handoffs are `Implemented`, without
intermediate proof; all required packet checks are consolidated
here before final acceptance. `make verify` formats changed handwritten files,
lints changed packages, and tests the module-local reverse importer closure instead of defaulting to `test-all`. `test-all` remains the
fallback when `go.mod`/`go.sum` change, the package graph cannot be built, or
the reverse closure is most of the module. Use `make plan` only to diagnose
that selection; `make verify` already prints the plan it will run.
`ALLOW_FULL=1 make check` remains the explicit full formatting, `lint-all`,
`test-all`, root-module tidy, and generated-drift gate; do not run its leaves
beside it.

`make lint-changed` accepts one `PKG` or a space-separated `PKGS` batch;
`make test-package` requires `PKG` and remains package-scoped.
`lint-pr` is the PR correctness, architecture, resource, error, context, and
interface set and requires `PKG` or `PKGS`. Full-module leaves are `lint-all`
and `test-all`; they require `ALLOW_FULL=1` or `CI=true`. `make lint-deep`, `make test-race`,
`make test-integration`, and `make audit-full-manual` require `ALLOW_HEAVY=1`
or `CI=true`.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` when a race or environment claim requires fresh execution.
Final validation uses `VALIDATION_JOBS=2` and one Git-common process lock by
default; CI raises the shared budget explicitly.
