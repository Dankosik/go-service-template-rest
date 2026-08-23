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

The integrated delivery owner or CI runs `make check` once. That target owns
full formatting, unit tests, the complete golangci-lint policy, module tidy,
and generated-contract drift. Do not also run `fmt-check`, `lint-all`, or
`test-all` beside it. Repository-wide leaves are not a per-lane or per-sibling default.

`make lint` and `make test` require `PKG` and are package-scoped. Full-module
leaves are `lint-all` and `test-all`. `make lint-deep`, `make test-race`,
`make test-integration`, and `make verify` require `ALLOW_HEAVY=1` or `CI=true`.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` when a race or environment claim requires fresh execution.
