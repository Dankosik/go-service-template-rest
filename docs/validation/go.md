# Go Validation

Use existing gopls diagnostics and the narrowest package test while iterating.
Run `gopls references file.go:line:column` before a semantic rename or exported
signature change. `make lint-fast PKG=<package>` adds changed-code
mechanical feedback without turning its `--new-from-rev` filter into a full
lint claim. `make test-watch` reruns the package containing the saved Go file;
press `a` only when an explicit full run is useful.

Package-sized iteration is optional:

```bash
make prove PKG=./internal/<package> FILES='internal/<package>/a.go internal/<package>/a_test.go'
```

`PKG` and `FILES` are required. There is no `./...` default. `make prove` is
the lock-wrapped `unit-check`. Go's build/test cache and golangci-lint's cache
reuse unchanged inputs; do not layer a second package receipt on top of them.
Skip `make prove` when the change is already ready for completion.

The integrated delivery owner runs `make verify` once. It formats changed
handwritten files, lints changed packages, and tests the module-local reverse
importer closure instead of defaulting to `test-all`. `test-all` remains the
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
