# Go Test Build Race And Lint Evidence

## When To Load
Load this for claims about Go tests, builds, race detector coverage, vet, lint, or package-level command evidence.

## Command Semantics To Remember
- `go test` proves the packages it is asked to test; a focused `-run` pattern proves only matching tests.
- `go build` proves compile success for packages and dependencies; it does not run tests and ignores `_test.go` files.
- `go test -race` adds race detector instrumentation, but only executed code paths can reveal races.
- `make lint` in this repo verifies the golangci-lint config before running linters.
- `make test-race` is the repo-wide race detector target.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "The new unit test passes" | `go test ./internal/config -run '^TestLoadDefaults$' -count=1` | `go test ./...` output from before the test was added |
| "All unit tests pass" | `make test` or `go test ./...` with current output | `go test` in the current package only |
| "Build succeeds" | `make build` | `go test ./...` alone |
| "Lint clean" | `make lint` | `go vet ./...`; `go test ./...` |
| "Race detector is clean for the worker package" | `go test -race ./internal/app/worker/...` | non-race `go test ./internal/app/worker/...` |
| "Repo race detector is clean" | `make test-race` | targeted package race run |

## Exact Command Patterns

```bash
go test ./internal/config -run '^TestLoadDefaults$' -count=1
go test ./internal/config/...
go test ./... -count=1
make test
go vet ./...
make vet
make lint
make build
go test -race ./internal/app/worker/...
make test-race
```

Prefer repository `make` targets when the claim uses repository language or when the local docs define a target for that proof. Prefer raw `go test` with a focused package and `-run` when the claim is narrowly about one package or test. Add `-count=1` for uncached execution when freshness depends on executing the test body.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): `go build`, `go test`, `go vet`, package patterns, build flags, and `-race`.
- [testing package documentation](https://pkg.go.dev/testing): `go test` executes test functions and supports `-run` for focused tests and subtests.
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html): `-race` usage, runtime-coverage limits, and platform/runtime costs.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`: `make test`, `make vet`, `make test-race`, `make lint`, and `make build`.
- `Makefile`: concrete target recipes.
