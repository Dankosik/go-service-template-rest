# Build, Test, and Development Commands

Use the narrowest command that can falsify the claim. The Makefile exposes
product commands and non-trivial leaf composition; ordinary tooling stays
direct.

## Tools

Repository Go tools are pinned by `tools/go.mod` and run without a wrapper:

```bash
go tool -modfile=tools/go.mod <tool> [args...]
```

Buf and its Protobuf plugins are pinned in the same tools module. Redocly is
pinned by the Makefile and invoked through `npx` with telemetry disabled.

## Initialize a service

```bash
make template-init \
  MODULE=github.com/acme/service \
  CODEOWNER=@acme/platform
ALLOW_HEAVY=1 make template-init-check
```

The default is the documented minimal service. Supported selections are the
individual choices and dependency combinations documented in the README:

- `DATABASE=postgres`;
- `DATABASE=postgres HTTP_IDEMPOTENCY=postgres`;
- `DATABASE=postgres JOBS=postgres`;
- `DATABASE=postgres JOBS=postgres WEBHOOKS=durable`;
- `DATABASE=postgres JOBS=postgres INBOUND_WEBHOOKS=standard-webhooks`;
- `DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream`;
- `MESSAGING=nats-jetstream`, `GRPC=enabled`, `AUTHN=oidc-jwt`,
  `OUTBOUND_HTTP=bounded`, `OBJECT_STORAGE=s3`, and
  `OUTBOUND_AUTH=oauth2-client-credentials` with HTTP, gRPC, or both.

Other cross-products are not a supported preset merely because the initializer
can mechanically resolve their markers.

`init-module.sh` validates selections and identity before mutation, records an
`initializing` journal, rewrites module/CODEOWNERS/service identity, physically
removes unselected profiles, regenerates retained outputs, tidies modules, and
marks `template.lock` complete only after postconditions pass. A same-selection
rerun resumes or returns without drift; another identity or profile fails
unchanged. The contract check captures the exact current working tree, clears
ambient profile variables, and exercises the smallest representative profile
rows plus partial-failure recovery.

## Add an outbound integration

```bash
make integration-init NAME=billing TRANSPORT=http \
  CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
```

While changing the initializer harness, select one named row and keep the
current complete matrix for final acceptance:

```bash
bash scripts/ci/integration-init-check.sh --list
INTEGRATION_INIT_ROWS=row_e1_http make integration-init-check
make integration-init-check
```

## Fast Go iteration

Use active gopls diagnostics first, then format and test only the changed files
and package:

```bash
go tool -modfile=tools/go.mod goimports -l path/to/changed.go
go tool -modfile=tools/go.mod gofumpt -l path/to/changed.go
go test -vet=off ./path/to/changed/package
make lint-fast PKG=./path/to/changed/package
```

When only the CLI exposes diagnostics, use `gopls check path/to/changed.go`.
Use `gopls references file.go:line:column` before a semantic rename or exported
signature change. `make test-watch` watches all Go directories but runs the
package containing the saved file; press `a` for an explicit full run.

## Ordinary Go leaves

Edit-loop proof is a focused package test, not a Makefile aggregate:

```bash
go test -vet=off ./internal/<package>
```

One acceptance-unit aggregate, with required package and file lists:

```bash
make unit-check \
  PKG=./internal/<package> \
  FILES="internal/<package>/a.go internal/<package>/a_test.go"
```

Inspect and run the surface-aware integrated plan:

```bash
make plan
make verify
```

`plan` collects base-branch, staged, unstaged, and untracked changes, explains
their surfaces and not-applicable gates, and prints the minimal command set.
`verify` rejects missing heavy authorization, Docker, or required binaries before
the first check. It batches changed Go formatting and lint packages, runs one
`test-all`, and serializes CPU/Docker work through the Git-common validation
lock. Its exact passing receipt is shared by related worktrees and binds the
resolved base, merge base, plan, execution inputs, tool versions, and candidate;
a candidate change during execution invalidates the run.

The explicit full-repository owner remains available when the claim spans it:

```bash
make check
```

`make check` runs formatting, `lint-all`, `test-all`, module tidy, and
generated-contract drift. Do not append it to ordinary completion or run
`fmt-check`, `lint-all`, or `test-all` beside it.

Package-scoped names refuse a missing `PKG`; they do not default to `./...`:

```bash
make test-package PKG=./internal/<package>
make lint-changed PKG=./internal/<package>
make lint-changed PKGS='./internal/one ./internal/two'
make lint-pr
make test-all
make lint-all
make fmt
```

Heavy leaves require `ALLOW_HEAVY=1` or `CI=true`:

```bash
ALLOW_HEAVY=1 make test-race
ALLOW_HEAVY=1 make lint-deep
ALLOW_HEAVY=1 make audit-full-manual
```

Use `make lint-deep` only for whole-program dead-code and nil analysis. Use
`make test-race` only when the claim spans concurrency-sensitive code.
`make audit-full-manual` is the rare template/release audit; it is not an
iteration or ordinary pre-commit target. It builds one image and reuses it for
migration rehearsal and image scanning.
While editing one module, use `make root-mod-check` or `make tools-mod-check`.
`tools-dependencies-check` adds registered-tool resolution for a tools-module
change; the combined `make mod-check` owns ordinary root/tools parity.

## Generated contracts

```bash
make openapi-generate
make openapi-check
make sqlc-generate
make sqlc-check
```

Checks run the canonical generator and a scoped Git diff. SQLC additionally
rejects source/output stem mismatch or output without sources.

<!-- profile:grpc:start -->
```bash
make proto-format
make proto-generate
make proto-check
make proto-breaking BASE_REF=origin/main
```

Buf owns formatting, lint, generation, and breaking comparison. Generated Go
is never edited manually.
<!-- profile:grpc:end -->

## PostgreSQL and integration

```bash
ALLOW_HEAVY=1 make test-integration-db
ALLOW_HEAVY=1 make test-integration-messaging
ALLOW_HEAVY=1 make test-integration-process
ALLOW_HEAVY=1 make test-integration-race
ALLOW_HEAVY=1 make test-integration
```

The first four targets own database, messaging, process, and concurrency
surfaces. `test-integration` remains the full non-race pack; race is always an
explicit separate claim. Add `REQUIRE_DOCKER=1` when Docker-backed proof must
fail rather than skip.

<!-- profile:database-postgres:start -->
```bash
make migration-check
make migration-validate
```

`migration-check` combines Goose validation with append-only Git history in a
derived repository. `migration-validate` uses disposable PostgreSQL, focused
real-database tests, the production image migrator, readiness/version readback,
and clean SIGTERM shutdown.
<!-- profile:database-postgres:end -->

<!-- profile:messaging-nats-jetstream:start -->
`make test-messaging-race` owns the focused NATS lifecycle/race pack.
<!-- profile:messaging-nats-jetstream:end -->
<!-- profile:outbox-postgres:start -->
`make test-outbox-race` owns the real-PostgreSQL outbox race pack.
<!-- profile:outbox-postgres:end -->
<!-- profile:webhooks-durable:start -->
`make test-webhook-race` owns durable outbound-webhook race proof.
<!-- profile:webhooks-durable:end -->

## Instructions and template propagation

```bash
make agent-roles-check
make codex-agents-check
make claude-skills-check
make qwen-skills-check
make template-owned-purity-check
```

Use the single matching carrier check from
[`validation/instructions.md`](validation/instructions.md) while iterating.
The template purity target already runs every carrier check, so do not pair it
with those leaves in the same final bundle.

`scripts/harness-skills-sync.sh` is the one Claude/Qwen skill-view owner.
`scripts/template-sync.sh` checks or applies one committed template snapshot to
one target and leaves the result in that target's working tree.

## Delivery and security

When the installed native versions match the Makefile pins, use the local-only
fast targets during iteration and retain the containerized targets for final
proof:

```bash
make actionlint-fast
make shellcheck-fast
```

```bash
make actionlint
make shellcheck
make dockerfile-check
make govulncheck
make gosec
make secret-scan BASE_REF=origin/main
make secret-scan-history
```

Gitleaks consumes the reviewed baseline. Local commands use the tools module;
CI downloads a checksum-pinned binary whose version is read from that same
module. Local review scans the current tree and exact base-to-HEAD history.
Clean pull-request CI scans only that commit range; push and release admission
scan full history.

## Runtime images

```bash
make runtime-image-build RUNTIME_IMAGE=service:ci
make runtime-image-check RUNTIME_IMAGE=service:ci
make container-security CONTAINER_IMAGE=service:ci
```

The source template builds one documented PostgreSQL generated output. A
derived repository builds its own exact source. The lifecycle check starts a
disposable PostgreSQL only when that current profile requires it; migration
rehearsal reuses its own database. Reuse the same image tag for lifecycle,
migration rehearsal, and vulnerability scanning. The Dockerfile fixes output
timestamps with `SOURCE_DATE_EPOCH=0`, so identical inputs rebuild to the same
local image digest; application version and commit remain explicit build inputs.

## Run and build

```bash
make run
make build
make pgo-manifest PGO_PROFILE=<representative-cpu.pprof> # requires provenance env; see Benchmarking
make build-pgo PGO_PROFILE=<representative-cpu.pprof> PGO_MANIFEST=<representative-cpu.pprof.meta>
make docker-build
make docker-run
```

`run` sources `.env` when present. PGO requires an explicit CPU profile and its
provenance manifest; `off` is the default and rollback. Off and PGO local
binaries have distinct paths, `bin/service` and `bin/service-pgo`.

<!-- profile:messaging-nats-jetstream:start -->
`make run-worker` and `make build-worker` own the messaging worker.
<!-- profile:messaging-nats-jetstream:end -->
<!-- profile:outbox-postgres:start -->
`make run-outbox-relay` and `make build-outbox-relay` own the outbox process.
<!-- profile:outbox-postgres:end -->
<!-- profile:jobs-postgres:start -->
`make run-jobs-worker` and `make build-jobs-worker` own the jobs process.
<!-- profile:jobs-postgres:end -->

## Benchmarking

Use `make benchmark-capture`, `benchmark-compare`, `benchmark-http`, and the
profile commands from [Benchmarking](benchmarking.md). The capture owner records
comparability metadata and rejects mismatched inputs before `benchstat`.
Benchmarks are explicit evidence, not default CI gates; CI only runs the
non-load `performance-harness-check` when its harness changes.

## CI and publication

`ci.yml` classifies the exact PR, merge-group, or main-push diff once, rejects
unclassified paths, runs only matching leaves, and exposes one always-reported
`required` context. PR jobs restore but do not save the separate root-module,
tool-module, build, and lint caches; the main quality owner refreshes them.
Generated Go, compose, publication metadata, and the messaging/outbox/webhook
race owners route independently, and image-only work skips Go setup.
`codeql.yml` uses the same exact-diff classifier and restores downloaded modules
without restoring the build cache CodeQL must observe. `cd.yml` waits for
exact-SHA CI and CodeQL before publishing through the digest-bound
`publish-image` action. GitHub Rulesets remain external repository policy.
