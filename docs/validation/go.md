# Go Validation

Use existing gopls diagnostics and the narrowest package test while iterating.
Run `gopls references file.go:line:column` before a semantic rename or exported
signature change. `make lint-fast LINT_PACKAGES=<package>` adds changed-code
mechanical feedback without turning its `--new-from-rev` filter into a full
lint claim. `make test-watch` reruns the package containing the saved Go file;
press `a` only when an explicit full run is useful.

Add `make fmt-check`, `make lint`, and `make test` only when the completion
claim spans those separate surfaces. Repository-wide leaves are not a per-lane or per-sibling default.

Add `make lint-deep`, `make test-race`, or `make test-integration` only when the
changed surface or claim triggers them.

Formatting and linters own mechanical style; tests own behavior. Use
`-count=1` when a race or environment claim requires fresh execution.
