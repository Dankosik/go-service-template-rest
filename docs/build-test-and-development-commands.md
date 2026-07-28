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

Keep the Go and BuildKit caches between runs. `gosec` uses the normal Go build
cache. GitHub's race, coverage, and integration jobs key their cache only from
the runtime `go.sum`; jobs that execute pinned tools also include
`tools/go.sum`. Every workflow job that runs golangci-lint derives the exact
version from `tools/go.mod`, installs its release binary through the same
SHA-pinned action, and still invokes the Make or template-check owner. Local and
CI configuration therefore have the same owner without compiling the linter in
every cold job. The action restores its golangci-lint analysis cache before its
install-only step returns and saves it after the owning Make or template check;
no second `actions/cache` step owns the same directory.

The production Dockerfile persists module and compiler caches across BuildKit
builds, but bind-mounts the source only for compilation. `.dockerignore`
excludes API source, examples, tool sources, test files, and testdata from that
runtime build context, so changes to those files do not create source layers or
invalidate the production binary. Scripts inherit `go env GOCACHE` and the
linter's canonical cache instead of creating repository-local copies that CI
does not restore. Do not use `go clean -cache`, `docker build --no-cache`, or
automatic BuildKit pruning as an edit-loop speed technique.

Local Trivy scans reuse the shared `trivy-cache` Docker volume for the
vulnerability database. The target suppresses progress/update noise but not
findings. Do not prune this volume routinely; remove it manually only when a
corrupt database is diagnosed or reclaiming its roughly gigabyte-scale disk
cost is worth making the next scan cold. See [Trivy cache
management](https://trivy.dev/latest/docs/configuration/cache/).

The pinned Redocly command prefers the local npm cache and disables update
notices and telemetry. It does not replace the exact version pin or turn a
missing package into a successful offline skip.

GitHub's Docker cache backend is appropriate only after the repository has
durable cache-budget headroom and a compatible non-default Buildx driver. See
the [GitHub cache limits](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching#usage-limits-and-eviction-policy)
and [Docker backend requirements](https://docs.docker.com/build/cache/backends/gha/).

### Temporary Docker resource lifecycle

The command or test that creates a temporary Docker resource owns its normal
cleanup. Use [`docker run --rm`](https://docs.docker.com/reference/cli/docker/container/run/#rm)
for throwaway containers. Register a trap before ephemeral Compose work and run
[`docker compose down -v --remove-orphans`](https://docs.docker.com/reference/cli/docker/compose/down/)
from that trap. For Testcontainers, register
[`CleanupContainer` or `Terminate`](https://golang.testcontainers.org/features/garbage_collector/#terminate-function)
immediately after a successful `Run`. Named volumes require the matching
`RemoveVolumes` termination option or owner cleanup.

Resources that may survive an interrupted owner can opt into the workstation
janitor only when deletion is safe and all four labels are present:

```text
codex.cleanup=auto
codex.data=ephemeral
codex.owner=<repository-or-task>
codex.expires_at=<Unix-seconds>
```

Attach those labels when an explicitly created volume, network, image, or
container is created. The owner still removes it on success; the janitor is
only crash recovery. Never apply the labels to database state, operator data,
shared caches, reusable environments, or any resource whose loss would require
recovery rather than regeneration. An old or currently unreferenced resource
is not deletion proof.

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
| `make claude-skills-check` | Every `.agents/skills/` entry is exposed to Claude Code by a matching `.claude/skills/` symlink, and no link outlives its skill |
| `make check` | Project structure, `fmt-check`, `lint`, and ordinary unit tests |
| `make ci-local` | Fast host-toolchain CI aggregate: manifest drift, project structure, format, lint, deep lint, race, coverage report, generated contracts, Go security, and secret scan |
| `make check-full` | `ci-local` plus required Docker integration, runtime image, migration, and image-security proof |
| `make pr-check BASE_REF=origin/main` | `check-full` plus template initialization, downloaded-module verification, and OpenAPI breaking comparison when the base contains the spec |

Module validation is split without weakening the aggregate:

- `make mod-tidy-check` checks root/tools manifest drift and Go-version parity;
- `make mod-verify` verifies downloaded root/tools module content;
- `make mod-check` runs both and remains the CI/release gate.

`ci-local` and `check-full` intentionally use `mod-tidy-check`; downloaded
content and the generator contract belong to `pr-check` and their focused
targets. Merge CI still runs `mod-check`, and its minimal and PostgreSQL
generator proofs run as independent jobs. Generated services no longer own
`scripts/profiles`, so those jobs stop after checkout instead of installing Go,
Node, or golangci-lint. Run `make mod-verify` or
`make template-init-check` while changing their owning inputs, and run
`make pr-check BASE_REF=origin/main` for the complete pre-PR proof.

Merge CI skips both generated-service jobs when every changed path is an
ordinary Go or API implementation path that initialization does not transform.
Changes to generator scripts, profiles, bootstrap/config/PostgreSQL ownership,
modules, Make, Docker, environment, workflows, or any unknown path run both
profiles. An empty or unresolvable comparison also runs both profiles.

`check-full` fails immediately when Docker is unavailable. It never converts a
missing container runtime into a successful skip. `test-integration` disables
Go's result cache because current dependency startup, health, behavior, and
cleanup are the proof; ordinary unit-test and build caches remain enabled.

CI classifies the base-to-HEAD path set before starting runtime-heavy jobs.
Markdown-only changes under the repository instructions, `docs/`, or `specs/`
run repository integrity, secret scanning, and the stable `ci-required`
aggregate while skipping repo-integrity Go setup, module/format/generated
checks, Dependency Review, Go runtime, integration, OpenAPI, and container
jobs. The secret job keeps Go setup for its pinned scanner. Dependency Review
remains active for non-doc pull requests. An empty, mixed, unrecognized,
manually dispatched, or unresolvable comparison runs the full matrix.
`make ci-change-scope-check` locks this fail-closed policy locally.

Manual CI exposes `go_cache_enabled` solely for same-ref cache A/B runs. Push
and pull-request CI always keep the `actions/setup-go` module/build cache
enabled. Compare sequential manual runs on the same ref; concurrent runs cancel
each other by design.

The container job builds the production runtime image once. Trivy scans that
tag and migration validation reuses it when a migration, runtime, dependency,
Dockerfile, Makefile, or CI owner changed.

### Keep the laptop responsive

The normal commands favor throughput and retain all caches. On a 10-core,
16-GiB development Mac, this bounded profile leaves capacity for the desktop
while preserving the same test and analysis coverage:

```bash
make check-gentle
make ci-local-gentle
GOMAXPROCS=6 make test
GOMAXPROCS=6 make check-full
```

The gentle targets set `GOMAXPROCS` to `GENTLE_GOMAXPROCS=6` and run at
`GENTLE_NICE=10` by default while executing exactly the normal target graph.
Go uses `GOMAXPROCS` for runtime and default package-build parallelism, and
gosec follows that explicit bound. golangci-lint remains independently capped
at `LINT_CONCURRENCY=4`: an A/B against automatic concurrency found no wall-time
benefit, so the smaller worker count wins. [`nice`](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man3/nice.3.html)
only lowers CPU scheduling priority when other work competes for the host; it
does not impose an idle-host sleep or memory limit. All values remain
overrideable when another machine has a measured reason.

Broad Make aggregates serialize their heavyweight prerequisites even when
`MAKEFLAGS=-j` is inherited, and golangci-lint callers wait on its native
runner lock instead of bypassing it. On the 10-core/16-GiB reference Mac, one
same-target A/B measured 138.7s sequentially and 294.6s with `make -j4`, a
2.1x slowdown. Re-measure after the host, Go toolchain, or aggregate membership
changes; until then, do not start another broad Go or Docker gate on the same
host while one is active. Use focused checks or wait. See
[GNU Make `.NOTPARALLEL`](https://www.gnu.org/software/make/manual/html_node/Parallel-Disable.html)
and [golangci-lint runner flags](https://golangci-lint.run/docs/configuration/cli/).

OrbStack limits are machine-wide. Record the current values, stop or preserve
unrelated workloads, then use this balanced profile only when a restart is
safe:

```bash
orb config get cpu
orb config get memory_mib
orb config set cpu 6
orb config set memory_mib 6144
```

Some settings take effect only after restarting OrbStack. Restore the recorded
values with the same `orb config set` commands when maximum local throughput is
more important than responsiveness. Do not hardcode Compose or builder
CPU/memory limits from laptop capacity alone. First record peak use for the
representative integration/build workload and retain headroom; otherwise the
limit can turn a healthy test into throttling, swapping, OOM, or flaky proof.

## Tests

Mandatory golangci-lint owns `govet` for the current repository. Repeated unit,
race, coverage, fuzz, flake, and OpenAPI test commands therefore pass
`-vet=off`; this removes duplicate analyzer work without reducing the gate.
Disposable generated-template tests and integration-tag tests retain Go's
default vet because they are not equivalent to the current-tree lint run.

```bash
make test
make test-watch
make test-race
make test-cover
make test-report
make test-fuzz-smoke
make test-flake-smoke
make test-integration
```

`make test` uses the pinned `gotestsum` format. The coverage job also executes
the ordinary test suite, so CI does not carry a duplicate standalone test job.
`lint` owns the configured Go analyzers, including vet-class checks.

Effective filtered coverage is the merge gate; raw coverage is informational.
Coverage uses `-covermode=set`: the gate asks whether each statement executed,
not how many concurrent executions occurred, so the more expensive atomic
counter mode adds no evidence. See [Go coverage modes](https://go.dev/blog/cover).
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
on network availability without proving more. `GOLANGCI_LINT` is an
execution-path override used by CI after its pinned binary installer; local
commands default to the version in `tools/go.mod`. The template check accepts
the narrower `GOLANGCI_LINT_BIN` path override. Every invocation uses
`--allow-serial-runners`, so concurrent repository sessions queue on the
linter's native lock instead of running analyzers twice.

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

`make test-watch` passes `-vet=off` to the same current-tree test path as
`make test`; mandatory lint remains the single `govet` owner.

## OpenAPI, SQLC, and generated drift

```bash
make openapi-generate
make openapi-drift-check
make openapi-reference-compile
make openapi-runtime-contract-check
make openapi-lint
make openapi-validate
make openapi-check

make sqlc-generate
make sqlc-check
```

The runtime contract compiles the main generated OpenAPI package through the
real router. `openapi-reference-compile` separately compiles the optional
reference package; `openapi-check` composes both owners without compiling the
main generated package twice.

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
make secret-scan-history
make container-security CONTAINER_IMAGE=service:ci
```

`secret-scan` checks the current reviewable worktree and commits after the
merge base with `SECRET_SCAN_BASE_REF=origin/main` by default. It scans full
history when that ref is unavailable, so a configuration mistake cannot skip
proof. `secret-scan-history` is the explicit full-history gate used on main and
release. The Go tools are pinned. `gosec` receives the repository
Go version directly, avoiding its fallback `go list`, and follows an explicit
`GOMAXPROCS` concurrency bound when one is present; see [gosec
performance](https://github.com/securego/gosec#performance). `container-security`
scans the actual runtime image with digest-pinned Trivy and requires Docker.
Its database cache persists in the shared `trivy-cache` Docker volume, so
repeated scans do not download the same database again.

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
exists for runtime dependencies, not to emulate every native Make target. Its
PostgreSQL healthcheck probes every second during startup, then returns to the
30-second steady-state interval, avoiding a permanent 12-probes-per-minute
wake-up loop without delaying readiness.

Integration packages pay for one PostgreSQL container per test binary, while
each test still owns and drops an isolated database.
The disposable cluster passes `POSTGRES_INITDB_ARGS=--no-sync`; together with
Testcontainers' built-in `fsync=off`, this skips crash-durability work that the
suite does not claim to test. Do not copy either setting to a persistent
database: PostgreSQL explicitly warns that an interrupted `initdb --no-sync`
cluster may be corrupt. See the [official image initialization
contract](https://github.com/docker-library/docs/blob/master/postgres/README.md)
and [`initdb --no-sync`](https://www.postgresql.org/docs/17/app-initdb.html).

All Testcontainers runs keep Ryuk enabled so an interrupted process still has a
cleanup owner. A same-host A/B found no wall-time benefit from disabling it, so
workflows do not trade that cleanup for an unmeasured optimization. Cross-run
container reuse remains disabled because it weakens freshness and cleanup
ownership. See the
[Testcontainers cleanup contract](https://golang.testcontainers.org/features/garbage_collector/)
and [configuration reference](https://golang.testcontainers.org/features/configuration/).

The pinned Distroless runtime already supplies `/usr/share/zoneinfo`, so the
Docker build does not also link `time/tzdata` into each binary. Reopen that
choice if the final image changes to `scratch` or another runtime without a
system timezone database.

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
