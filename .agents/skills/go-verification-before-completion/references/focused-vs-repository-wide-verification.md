# Focused Vs Repository-Wide Verification

## When To Load
Load this when deciding whether a focused command is enough, or when the user asks for broad wording such as "all tests pass", "repo is green", "ready to merge", or "everything is fixed".

## Scope Rule
Match proof breadth to claim breadth:
- A focused reproducer proves that focused behavior, not the repository.
- A package pattern proves that package pattern, not unrelated packages.
- A repository-wide target proves the repository dimension named by the target.
- "Ready" claims are composite and must include every changed surface that could break independently.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "The parser fix works" | `go test ./internal/app/parser -run '^TestParserRejectsTrailingJSON$' -count=1` | `make build` passes |
| "The parser package is green" | `go test ./internal/app/parser/...` | only `TestParserRejectsTrailingJSON` passes |
| "All Go tests pass" | `make test`, or `go test ./...` with honest cache reporting | `go test ./internal/app/parser/...` |
| "This API change is ready for handoff" | focused HTTP/API tests plus `make openapi-check`, and usually `make test` or the planned repo quality target | `make openapi-runtime-contract-check` alone |
| "This migration change is ready" | migration rehearsal plus affected data-access tests and generated SQL checks when query generation changed | `make test` alone |

## Exact Command Patterns

```bash
go test ./internal/app/parser -run '^TestParserRejectsTrailingJSON$' -count=1
go test ./internal/app/parser/...
make test
make lint
make check
make check-full
make openapi-check
make sqlc-check
make migration-validate
```

Use `make check` for a quick repo quality claim when fmt, lint, and tests are all in scope. Use `make check-full` only when the claim is full local CI-like readiness and the runtime prerequisites are acceptable.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): package patterns define which packages a command compiles or tests.
- [testing package documentation](https://pkg.go.dev/testing): `-run` selects tests and subtests by name pattern.
- [Go command overview](https://go.dev/doc/cmd): Go commands usually operate at package level through the `go` program.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`: defines `make check`, `make check-full`, `make test`, `make lint`, API, sqlc, and migration targets.
- `Makefile`: defines the concrete command recipes behind those targets.
