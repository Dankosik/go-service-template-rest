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

The integrated delivery owner runs `make verify` once. It selects one changed-file
format check, one batch `lint-changed`, one cached `test-all`, and any other
applicable surface owners. Deleted Go files still select the surviving package
for lint and always select `test-all`. Use
`make plan` to inspect that selection first. `make check` remains the explicit
full formatting, `lint-all`, `test-all`, module-tidy, and generated-drift gate;
do not run its leaves beside it.

`make lint-changed` accepts one `PKG` or a space-separated `PKGS` batch;
`make test` requires one `PKG` and remains package-scoped.
`lint-pr` is the PR correctness, architecture, resource, error, context, and
interface set. Full-module leaves are `lint-all` and `test-all`. `make lint-deep`, `make test-race`,
`make test-integration`, and `make audit-full-manual` require `ALLOW_HEAVY=1`
or `CI=true`.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` when a race or environment claim requires fresh execution.
Final validation uses `VALIDATION_JOBS=2` and one Git-common process lock by
default; CI raises the shared budget explicitly.
