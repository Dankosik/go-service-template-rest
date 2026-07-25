# Build, Test, and Development Commands

The `Makefile` is the command index. Scripts exist only when they own behavior
that would be awkward or misleading inside a recipe.

## Prerequisites

- Go at the version declared in `go.mod`.
- Node.js and `npx` for the pinned Redocly OpenAPI lint command.
- Docker only for container-backed integration tests, migration rehearsal,
  runtime-image build and scan, Compose, real-PostgreSQL benchmarks, and k6 HTTP
  benchmarks.

Go tools are pinned through the `tool` block in `tools/go.mod` and invoked by
`scripts/run-go-tool.sh` from the repository root. Runtime and test
dependencies remain in the root `go.mod`. `make mod-check` verifies and checks
Go-version consistency for both modules. Container tools and test dependencies
are pinned at their owning command or test seam.

## Initialize a derived service

```bash
make template-init \
  MODULE=github.com/acme/orders \
  CODEOWNER=@acme/backend \
  DATABASE=none
make template-init-check
```

`template-init` is the only onboarding mutation. It validates the module,
owner, and profiles before editing; rewrites service identity, both Go module
paths, Go/proto imports, lint configuration, CODEOWNERS, OpenAPI title, and the
derived README; preserves an existing `.env`; and tidies both modules.
`DATABASE=none` removes PostgreSQL runtime, migration, test, image, deployment,
and tool surfaces. `DATABASE=postgres` retains them. The complete agent
workflow and its harness files are always retained and are not an
initialization profile.

The template source checkout may run the command without arguments for normal
local setup; it keeps the template module and CODEOWNERS unchanged while
creating a missing `.env` and tidying the module. A derived checkout must
provide a real module path and an owner in `@user` or `@org/team` form.

## Everyday validation

| Command | Meaning |
| --- | --- |
| `make project-structure-check` | Placement, naming, command, integration-test, migration-pair, and no-empty-placeholder contract |
| `make check` | Project structure, `fmt-check`, `lint`, and ordinary unit tests |
| `make ci-local` | Host-toolchain CI aggregate: module, initialization, project structure, format, lint, deep lint, race, coverage report, generated contracts, Go security, and secret scan |
| `make check-full` | `ci-local` plus required Docker integration, runtime image, migration, and image-security proof |
| `make pr-check BASE_REF=origin/main` | `check-full` plus OpenAPI breaking comparison when the base contains the spec |

`check-full` fails immediately when Docker is unavailable. It never converts a
missing container runtime into a successful skip.

## Tests

```bash
make test
make test-summary
make test-watch
make test-race
make test-cover
make test-report
make test-fuzz-smoke
make test-flake-smoke
make test-integration
```

`make test` uses the pinned `gotestsum` format; `make test-summary` is its
compatibility alias. The coverage job also executes the ordinary test suite, so
CI does not carry a duplicate standalone test job. `lint` owns the configured
Go analyzers, including vet-class checks.

Effective filtered coverage is the merge gate; raw coverage is informational.
The configured filter excludes generated OpenAPI and sqlc code, the
test-support `internal/infra/telemetry/telemetrytest` package, and `cmd`
composition roots. Integration-tag coverage is separate. Repository maintainers
own `COVERAGE_MIN` changes and must record the rationale. Treat
`COVERAGE_MIN` as a floor rather than a target; add tests for meaningful risk,
not to manufacture a fixed percentage-point margin.

Use standard Go selection flags for focused local work; no wrapper targets are
needed:

```bash
go test ./internal/config
go test ./internal/config -run '^TestLoadDefaults$'
go test ./internal/config -run '^TestResourceIdentityFieldsCannotBeEmpty$/app_version$'
go test -count=1 ./internal/config
go test ./internal/config -run '^FuzzParseDuration$'
go test ./internal/config -run '^$' -fuzz='^FuzzParseDuration$' -fuzztime=30s
```

The first fuzz command runs the seed corpus only; the second actively fuzzes
for the stated duration.

Integration tests use the `integration` build tag. Local focused execution may
skip an unavailable container dependency according to the test contract;
`REQUIRE_DOCKER=1 make test-integration` makes Docker availability mandatory,
as full CI does.

## Formatting and analysis

```bash
make fmt
make fmt-check
make mod-check
make lint
make lint-deep
make lint-fast LINT_BASE_REF=origin/main
make deadcode
make nilaway
make modernize-check
make test-parallelism-check
```

`make lint` runs golangci-lint with `.golangci.yml`. The linter's real config
load is the oracle; a second schema-download check would make local lint depend
on network availability without proving more.

`make lint-deep` runs the whole-program dead-code and NilAway analyses. They are
separate so `make check` stays cheap enough to run before every commit: on a
generated health-only service, a cold-cache `make check` measured about 26s and
`lint-deep` was 9s of it, while a warm edit-loop `make check` measured under 2s
against 2.6s for `lint-deep` alone. Most of a cold run is compiling the
dependency graph the analyzers need, not the analyzers themselves, so expect
both numbers to grow with the service rather than with the rule count.
`make ci-local` and the CI lint job run both targets, so nothing is optional on
the way to merge. Use the focused targets only when their narrower evidence is
the claim.

## OpenAPI, SQLC, and generated drift

```bash
make openapi-generate
make openapi-drift-check
make openapi-runtime-contract-check
make openapi-lint
make openapi-validate
make openapi-check

make sqlc-generate
make sqlc-check
```

`api/openapi/service.yaml`, its adjacent generation config, migrations, and SQL
query sources are authoritative. When `examples/reference-service` exists, its
OpenAPI document and generation config are checked too. A derived service may
delete that example; service-owned generation, drift, package tests, lint, and
validation continue without it. The shared generated-drift script snapshots
the current derived output, runs the canonical generators, and fails with a
diff only when generation changes that output.
Uncommitted but already current generated files therefore pass; Git and CI own
the separate question of whether those files were committed.

For a PR comparison:

```bash
git show origin/main:api/openapi/service.yaml > /tmp/service-base.yaml
make openapi-breaking BASE_OPENAPI=/tmp/service-base.yaml
```

## Security

```bash
make govulncheck
make gosec
make go-security
make secret-scan
make container-security CONTAINER_IMAGE=service:ci
```

The first four commands use pinned Go tools. `container-security` scans the
actual runtime image with digest-pinned Trivy and requires Docker.

## Migrations and containers

```bash
make migration-validate
make docker-build
make docker-run
make compose-up
make compose-down
```

When owned migration files exist, `migration-validate` rehearses `up all`,
`down all`, and `up all`. With no migrations, the host and image migration
entrypoints return a successful explicit no-op. The command requires Docker
and creates an isolated Compose project on a dynamic host port; it never
accepts an operator-supplied database because the rehearsal rolls back every
migration. It exercises the host migration tool, then runs the runtime image's
`/migrate` entrypoint on the Compose network.
The migrator defaults to a `5m` overall budget, `2m` per statement, and `15s`
for lock acquisition. Override the typed `APP__POSTGRES__MIGRATION_*` values
only from rehearsal evidence. Dirty-state recovery is operator-controlled and
documented in `docs/railway-deployment-profile.md`; normal execution never
forces a migration version.
It starts the same image with a read-only filesystem and dropped capabilities,
waits for `/health/ready`, optionally checks `RUNTIME_EXPECTED_VERSION` in the
startup log, and requires a clean SIGTERM exit. Cleanup is registered before
the rehearsal begins.

`docker-build` and `docker-run` operate on the production Dockerfile. Compose
exists for runtime dependencies, not to emulate every native Make target.

## Run and build

```bash
make run
make build
make vendor
```

`run` loads `.env` when present. `build` writes `bin/service`.

## Benchmarking

Choose the execution environment before capturing results. DigitalOcean is the
default when `doctl` is installed and its selected context is authorized. Start
with the read-only preflight; after it succeeds and the user authorizes the paid
lifecycle operation, run the matching benchmark target through the remote
runner:

```bash
make benchmark-remote-check
scripts/dev/benchmark-remote.sh list
scripts/dev/benchmark-remote.sh image-list
scripts/dev/benchmark-remote.sh run -- make bench

scripts/dev/benchmark-remote.sh create
scripts/dev/benchmark-remote.sh sync
scripts/dev/benchmark-remote.sh exec make bench-baseline
# Change to the candidate source, then sync and measure again on this Droplet.
scripts/dev/benchmark-remote.sh sync
scripts/dev/benchmark-remote.sh exec make bench
scripts/dev/benchmark-remote.sh exec make bench-compare
scripts/dev/benchmark-remote.sh fetch
scripts/dev/benchmark-remote.sh destroy
```

If `doctl` is absent or the selected context is not authorized, do not start
DigitalOcean setup automatically. Use the matching local command instead:

```bash
make bench
make bench-baseline
make bench-compare
make bench-profile

make bench-db BENCH_DB_WORKLOAD_ID=fixture-10k-warm
make bench-db-baseline BENCH_DB_WORKLOAD_ID=fixture-10k-warm
make bench-db-compare

make bench-http
make bench-http-inspect
make benchmark-infra-check
```

Go and in-process HTTP benchmarks use the host toolchain of the selected
environment. Database benchmarks use the existing Testcontainers seam.
External HTTP load uses the digest-pinned k6 image owned by
`scripts/dev/benchmark.sh`. Workload, comparison, and evidence rules are in
[Benchmarking](benchmarking.md).

For faster fresh-Droplet startup, source the non-secret reference produced by
`benchmark-remote-image`, then return to the normal least-privilege context:

```bash
# Optional paid one-time DigitalOcean snapshot build:
DO_BENCH_CONTEXT=benchmarks-image-builder make benchmark-remote-image
source .artifacts/bench/remote/golden-image.env
export DO_BENCH_CONTEXT=benchmarks
make benchmark-remote-check
```

Read the repository
[`digitalocean-benchmark-runner`](../.agents/skills/digitalocean-benchmark-runner/SKILL.md)
skill before provisioning. It owns one-time `doctl`/SSH setup, Cloud Firewall,
current pricing checks, private source transfer, separate HTTP generator/target
topology, host telemetry, recovery, and mandatory cleanup.

## CI and repository settings

`.github/workflows/ci.yml` owns the current CI job graph and its stable
`ci-required` aggregate. GitHub Rulesets or organization policy own merge
admission: require `ci-required` plus independently managed code-scanning
evidence instead of coupling protection to every internal job name. Do not use
a repository script to rewrite its own protection policy.

`.github/workflows/cd.yml` owns release validation and runtime image
publication. It reports an immutable digest and promotes mutable tags only
after signature and attestation verification. `railway.toml` owns the generic,
non-secret Railway source-build profile; it does not connect the template to a
Railway project or make GHCR evidence apply to Railway's independent build.
