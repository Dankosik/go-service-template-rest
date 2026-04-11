# Claim To Proof Mapping

## Behavior Change Thesis
When loaded for an ambiguous completion, readiness, or command-scope claim, this file makes the model bind the claim to an exact proof surface and choose the narrowest sufficient command set instead of either generalizing a focused pass to repo readiness or reflexively running unrelated broad checks.

## When To Load
Load this for fuzzy positive claims such as "fixed", "tests pass", "green", "ready", "safe", "done", "builds", "lint clean", race checked, package green, or repository green.

## Decision Rubric
- First quote the exact positive claim you are about to make.
- Bind its nouns to a proof dimension: focused behavior, package behavior, repository tests, build, lint, vet, race detector, or composite readiness.
- Choose the smallest command that directly proves that dimension for the named scope.
- For "ready", "merge", "handoff", or "green" claims, use the union of changed-surface checks. A readiness claim is rarely proven by one command.
- If the command proves only part of the claim, either run the missing proof or narrow the conclusion.
- Use `-count=1` when freshness requires executed test bodies, not merely a cache-valid package result.

## Imitate
| Claim | Choose | Copy this behavior |
|---|---|---|
| "I fixed `TestCreateUser`" | `go test ./internal/app/user -run '^TestCreateUser$' -count=1` | Match the failing test and force a fresh body execution. |
| "The parser package is green" | `go test ./internal/app/parser/...` | Expand from one focused test to the package tree named by the claim. |
| "Repository tests pass" | `make test` or `go test ./...` with honest cache reporting | Use repository-wide proof for repository-wide wording. |
| "Build succeeds" | `make build` | Do not let tests stand in for building the command binary. |
| "Lint is clean" | `make lint` | Use the repo target that verifies golangci-lint config before linting. |
| "Worker race path is checked" | `go test -race ./internal/app/worker/...` or `make test-race` | Race proof requires race instrumentation on the relevant executed path. |
| "Ready for review" | focused fix proof plus triggered repo checks such as `make test`, `make lint`, and any surface-specific checks | Treat readiness as a composite claim. |

## Reject
| Plausible bad conclusion | Why it fails |
|---|---|
| "All tests pass" after `go test ./internal/app/parser/...` | A package pattern does not prove unrelated packages. |
| "Build is good" after `go test ./...` | Tests compile packages under test, but this repo's build claim maps to `make build`. |
| "Race safe" after non-race `go test` | Races are only detected in race-instrumented executed paths. |
| "Ready to merge" after one focused `-run` test | A focused reproducer proves the fix path, not lint, build, generated drift, migrations, or broader regressions. |

## Agent Traps
- `go test` without `./...` can only prove the current package context.
- `go test ./...` output may include cached packages. That can support a cache-valid broad test claim, but not a claim that every test body just executed.
- `make check` proves this repo's quick fmt, lint, and test set. It is not the same as generated API, migration, security, or full CI-like proof.
- `make check-full` can still print local skip messages for Docker-backed checks when Docker is unavailable. Carry those gaps into the conclusion.
- If command names feel stale, inspect `Makefile` and `docs/build-test-and-development-commands.md` instead of guessing.

## Validation Shape
Report the command, exit result, key signal, and exact scope it proves. Keep the conclusion no broader than that scope.
