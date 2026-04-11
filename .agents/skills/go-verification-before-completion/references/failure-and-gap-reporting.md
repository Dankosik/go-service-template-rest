# Failure And Gap Reporting

## When To Load
Load this when proof failed, was not run, was skipped, used cached results unexpectedly, required tools or services were unavailable, or the available evidence is weaker than the user's requested completion claim.

## Reporting Rule
Use "not verified" when the evidence cannot support the claim. Lead with the blocking or missing proof, then give the smallest concrete next verification action. Do not hide a failure under a positive summary.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "Tests pass" | `make test` exits 0 and the output is current for the workspace | `make test` failed in one package; old passing output exists |
| "Migration validated" | migration command actually ran a rehearsal and exited 0 | command skipped because Docker and `MIGRATION_DSN` were unavailable |
| "Race detector clean" | `go test -race ...` or `make test-race` exits 0 on the relevant scope | race command was not run because it is slow |
| "Ready" | every required surface proof passed or a named risk is explicitly accepted | one optional check failed and is omitted from the report |

## Exact Command Patterns

```bash
make test
go test ./internal/app/user -run '^TestCreateUser$' -count=1
go test -race ./internal/app/worker/...
make test-race
make lint
make openapi-check
MIGRATION_DSN='postgres://user:pass@localhost:5432/db?sslmode=disable' make migration-validate
make docker-migration-validate
```

## Report Templates

Use this shape for a failed command:

```text
Not verified: `make test` failed.
Signal: `FAIL ./internal/app/user`, with `TestCreateUser` failing on duplicate key handling.
Next verification action: fix the failing path, then rerun `go test ./internal/app/user -run '^TestCreateUser$' -count=1` and `make test`.
```

Use this shape for a missing or skipped command:

```text
Not verified: migration rehearsal did not run.
Signal: `make migration-validate` reported that `MIGRATION_DSN` was empty and Docker was unavailable, so it skipped migration validation.
Next verification action: provide `MIGRATION_DSN` or start Docker, then rerun `make migration-validate`.
```

Use this shape for weaker evidence:

```text
Partially verified: focused parser test passed, but repository tests were not run.
Evidence: `go test ./internal/app/parser -run '^TestParserRejectsTrailingJSON$' -count=1` passed.
Not proven: "all tests pass" or "ready for merge".
Next verification action: run `make test` and any changed-surface checks.
```

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): `go test` and `go build` evidence is bounded by command and package scope.
- [testing package documentation](https://pkg.go.dev/testing): focused test evidence comes from selected test and subtest names.
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html): race detector misses paths that are not executed.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`: skip behavior and prerequisites for integration, migration, OpenAPI, and Docker-backed targets.
- `Makefile`: target behavior and fallback paths.
