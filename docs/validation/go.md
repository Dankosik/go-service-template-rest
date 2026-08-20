# Go Validation

Use the narrowest package test while iterating, then `make check` for an
ordinary completed Go change. Add `make lint-deep`, `make test-race`, or
`make test-integration` only when the changed surface or claim triggers them.

`make check` owns project structure, format, mandatory lint, and ordinary unit
tests. Linters own mechanical style; tests still own behavior.
