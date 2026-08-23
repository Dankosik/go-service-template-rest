# Go Validation

Use existing gopls diagnostics and the narrowest package test while iterating.
Run `gopls references file.go:line:column` before a semantic rename or exported
signature change. `make lint-fast PKG=<package>` adds changed-code
mechanical feedback without turning its `--new-from-rev` filter into a full
lint claim. `make test-watch` reruns the package containing the saved Go file;
press `a` only when an explicit full run is useful.

The Acceptance-Unit Lead runs one package aggregate:

```bash
make unit-check PKG=./internal/<package> FILES='internal/<package>/a.go internal/<package>/a_test.go'
```

`PKG` and `FILES` are required. There is no `./...` default.

The integrated delivery owner runs `make verify` once. It selects changed-file
formatting, `lint-changed`, package tests, and any other applicable surface
owners without automatically appending a repository-wide aggregate. Use
`make plan` to inspect that selection first. `make check` remains the explicit
full formatting, `lint-all`, `test-all`, module-tidy, and generated-drift gate;
do not run its leaves beside it.

`make lint-changed` and `make test` require `PKG` and are package-scoped.
`lint-pr` is the PR correctness, architecture, resource, error, context, and
interface set. Full-module leaves are `lint-all` and `test-all`. `make lint-deep`, `make test-race`,
`make test-integration`, and `make audit-full-manual` require `ALLOW_HEAVY=1`
or `CI=true`.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` when a race or environment claim requires fresh execution.
