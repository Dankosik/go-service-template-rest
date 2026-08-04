# Claim To Proof Mapping

## Behavior Change Thesis

When loaded for a "fixed", "green", "ready", or "done" claim, this file supplies
what the aggregate target names do not say — what each gate leaves unproven, and
which commands in this repository exit 0 without executing the behavior the claim
depends on.

## When To Load

Load this when a positive claim names a gate, a scope ("the package", "the repo"),
or readiness, and the honest boundary of that evidence is not obvious.

## Decision Rubric

The aggregates are not nested supersets. Each proves its own composition and is
silent about the rest:

| Gate | Composition | Silent about |
| --- | --- | --- |
| `make check` | `project-structure-check fmt-check lint test` | build, `lint-deep`, generated drift, security, anything Docker-backed |
| `make ci-local` | the host-toolchain CI aggregate: adds `lint-deep`, `test-race`, `test-report` with the coverage floor, `sqlc-check`, `openapi-check`, `proto-check`, `go-security`, `secret-scan` | integration, migration, runtime image, container security — everything needing Docker |
| `make check-full` | `delivery-quality`, `ci-local`, `REQUIRE_DOCKER=1 test-integration`, runtime image, `migration-validate`, `container-security` | base-relative proof: template init, `mod-verify`, OpenAPI/Protobuf breaking — those are `make pr-check BASE_REF=…` |

`check-full` and `migration-validate` **exit 1 when Docker is unreachable**; they
never convert a missing runtime into a passing skip, so a green run of either is
real Docker proof. `make build` is not implied by any test target — tests compile
the packages under test, not the command binaries. `make lint` is
`golangci-lint run` alone; `deadcode` and `nilaway` live in `lint-deep`, which
`make check` does not reach. A security claim maps to `go-security`
(govulncheck **and** gosec), with `secret-scan` and `container-security` as
separate surfaces. A performance claim is out of scope here — `docs/benchmarking.md`
owns its proof level and completion policy.

### Commands that exit 0 without proving anything

Verified on the pinned toolchain (go1.26.5); exit status alone will not tell you:

- A `-run` pattern matching no test prints `ok … [no tests to run]` and exits 0.
  This is why `openapi-runtime-contract-check` runs `-json` and greps for an actual
  `"Action":"run"` event rather than trusting the exit code — copy that check
  whenever a claim rests on one named test.
- A package with no test files prints `? … [no test files]` and exits 0, so
  "the package is green" can mean "the package has no tests".
- Integration tests **skip** when the Docker provider is unhealthy unless
  `REQUIRE_DOCKER=1` is set (`internal/infra/postgres/pgtest`). Bare
  `make test-integration` can pass having run nothing.

### What a cached result is still worth

`make test` carries no `-count=1`, so packages return `(cached)`; `-vet=off` does
not disable the cache. Go re-runs a package when the test binary changes, when a
cacheable flag changes, or when a file inside the module or an environment
variable **the test itself read** changes. It does not re-run because the world
changed around it: a started Docker daemon, a reseeded database, a rebuilt image,
or an edited file the test never opened all leave the cached PASS in place. A
cached result therefore supports "this code still passes its tests" and never
"this just ran against the current environment". `test-integration` pins
`-count=1` for exactly this reason; add it yourself when the claim is about a
dependency's current state rather than about the code.

## Reject

Reject a scope promotion the command did not earn: a package pattern says nothing
about unrelated packages, a focused reproducer says nothing about lint, drift, or
regressions, and race conditions surface only on race-instrumented paths that
actually executed. Reject `git diff` as behavioral proof, and reject another
agent's report or a prior session's log as proof of the current tree — a
worktree-isolated worker's changes are absent from the root checkout's
`git status` until integration, so re-run the claim-scoped command here.

## Validation Shape

Name the command, its exit result, whether the run was cached or fresh, and the
exact scope it proves. Keep the conclusion no broader than that scope, and when
the evidence is narrower than the claim, state what is not proven and the single
command that would close the gap.
