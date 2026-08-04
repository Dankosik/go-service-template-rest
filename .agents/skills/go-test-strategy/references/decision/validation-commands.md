# Validation Commands

## Behavior Change Thesis
When loaded for symptom "a strategy or a piece of reported evidence must name executable validation", this file makes the model pick the command that would fail for this regression instead of likely mistake "offer `go test ./...` as the gate, or demand the full CI aggregate by reflex."

## When To Load
Load this when a proof obligation, a review finding, or a readiness claim needs a named command, or when reported evidence may not exercise the changed surface.

## Decision Rubric
- A command is fit when it would fail for the regression under discussion. Breadth is not fitness: `make check-full` proves nothing in particular about the changed behavior.
- [Build, test, and development commands](../../../../../docs/build-test-and-development-commands.md) and [test/README.md](../../../../../test/README.md) own the full map and stay current. Read them before naming a target instead of re-deriving the list here.
- Prefer the narrowest command that would fail — `go test ./internal/<pkg> -run '<Test>/<subtest>' -count=1` — and add a repository gate only when the surface is wider than the package.
- Add `-count=1` when a cached pass could hide whether the changed code ran at all.

| Changed surface | Gate that can see it |
| --- | --- |
| OpenAPI or generated bindings | `make openapi-check` — drift, reference compile, runtime contract, lint, validate |
| Docker-backed or multi-package behavior | `make test-integration` over `./test/...` |
| Shared-memory races | `make test-race`, or `go test -race` on the package that executes the path |
| Order or scheduling sensitivity | `make test-flake-smoke` — `-count=5 -shuffle=on ./...` |
| `t.Parallel()` correctness | `make test-parallelism-check` — `paralleltest,tparallel` only |
| SQL source drift, migrations | `make sqlc-check`, `make migration-validate` |
| Coverage threshold and artifacts | `make test-report COVERAGE_MIN=<value>` |

## Imitate
Evidence reads `go test ./...` for a change that edited `api/openapi/service.yaml`. The command passes and proves nothing about the change: spec/runtime drift and an invalid document both survive it. `make openapi-check` is the answer, and the reason is the fitness test, not that it is the bigger command.

## Reject
- `go test ./...` offered as evidence for integration, OpenAPI drift, race, fuzz, or coverage. It cannot see `integration`-tagged packages, generated-artifact drift, or a fuzz target, and it exits zero without them.
- A local `make test-integration` that skipped Docker, reported as integration proof. CI sets `REQUIRE_DOCKER=1` so absence fails the job; the same command locally skips quietly, and a skip is not a pass.
- "Coverage is fine" as a finding or as a gate substitute. The merge gate is *effective filtered* coverage — generated OpenAPI, sqlc, and protobuf code, test-support packages, and `cmd` composition roots are excluded — against a maintainer-owned `COVERAGE_MIN` floor, currently `80.0`. Raw coverage is informational, and the floor is a floor rather than a target.
- "CI passed before" for a surface that has changed since.

## Agent Traps
- `make test-fuzz-smoke` discovers targets itself and prints a skip when none exist. That skip is an honest record, not robustness evidence.
- Makefile test targets pass `-vet=off` because mandatory lint owns `govet`; a focused local `go test` has no reason to imitate that flag.

## Validation Shape
State: obligation or finding → narrowest command that would fail → broader gate when the surface is wider → the residual limit the command does not cover.
