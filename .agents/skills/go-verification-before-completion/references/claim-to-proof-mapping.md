# Claim To Proof Mapping

## When To Load
Load this when the requested completion claim is broader than a single obvious command, or when the wording is ambiguous: "fixed", "tests pass", "ready", "clean", "green", "safe", "done", or "validated".

## Mapping Rule
Choose proof from the claim's nouns and scope:
- behavior claims need tests or a reproducer
- build claims need build commands
- lint claims need lint commands
- contract and generated-code claims need generation plus drift checks
- migration claims need migration rehearsal checks
- readiness claims need the union of checks required by the changed surfaces

If a command only proves part of the claim, either run the missing proof or narrow the conclusion.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "I fixed `TestCreateUser`" | `go test ./internal/app/user -run '^TestCreateUser$' -count=1` now passes after the fix | `make lint` passes; an old test log says it passed |
| "The user package is green" | `go test ./internal/app/user/...` passes, or `go test ./internal/app/user/... -count=1` when uncached execution matters | `go test` from another directory; one targeted test passes |
| "Repository tests pass" | `make test` passes, or `go test ./...` with a note if packages were cached | one package-level `go test` passes |
| "Build succeeds" | `make build` passes | `make test` passes without a build target for the command binary |
| "Lint is clean" | `make lint` passes | `go test ./...` passes |
| "Race-sensitive worker path is checked" | `go test -race ./internal/app/worker/...` or `make test-race` passes and covers the changed path | non-race `go test` passes; race detector run skips the relevant test |
| "Ready for review" | focused fix proof plus triggered repo checks, such as `make test`, `make lint`, and surface-specific checks | subagent says it is ready; only `go test -run` passed |

## Exact Command Patterns

```bash
go test ./internal/app/user -run '^TestCreateUser$' -count=1
go test ./internal/app/user/...
go test ./... -count=1
make test
make lint
make build
make test-race
make openapi-check
make migration-validate
```

Use `-count=1` when the claim requires executed test bodies, not merely a cache-valid package result. If repository policy uses `make test`, report the actual signal honestly, including `(cached)` when present.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): `go build` compiles packages and dependencies; `go test` tests packages and uses package patterns.
- [testing package documentation](https://pkg.go.dev/testing): `go test` runs `TestXxx` functions and supports focused execution through test names and subtests.
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html): `-race` only detects races that occur in executed code paths.
- [Go command overview](https://go.dev/doc/cmd): Go commands operate on package-level source through the `go` program.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`
- `Makefile`
