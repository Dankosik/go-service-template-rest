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

`init-module.sh` validates selections and identity before mutation, rewrites
module/CODEOWNERS/service identity, physically removes unselected profiles,
regenerates retained outputs, tidies modules, writes `template.lock`, and is
idempotent for the same selection. The contract check uses invalid synthetic
inputs and one default generated checkout; profile-owned Go tests retain their
own behavior.

## Add an outbound integration

```bash
make integration-init NAME=billing TRANSPORT=http \
  CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
```

While changing the initializer harness, select one named row and keep the
25-row default for final acceptance:

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
`verify` runs that set sequentially, reports durations and scope, and reuses an
exact passing receipt from the Git worktree metadata.

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
migration rehearsal, and vulnerability scanning.

## Run and build

```bash
make run
make build
make build-pgo PGO_PROFILE=<representative-cpu.pprof>
make docker-build
make docker-run
```

`run` sources `.env` when present. PGO accepts only an explicit readable CPU
profile; `off` is the default and rollback.

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

Use direct `go test -bench`, `benchstat`, profile, integration-tagged
PostgreSQL, or pinned k6 commands from [Benchmarking](benchmarking.md).
Benchmarks are explicit evidence, not default CI gates.

## CI and publication

`ci.yml` classifies the changed surface once, runs only the matching quality,
security, delivery, contract, instruction, and integration leaves, and exposes
one always-reported `required` context. Go module and build caches use separate
quality, security, secret-scan, and integration keys; Node is set up only for
OpenAPI work. `codeql.yml` uses the same classifier for pull requests and runs
both languages on main, tags, schedule, and manual dispatch. `cd.yml` waits for
exact-SHA CI and CodeQL before publishing through the digest-bound
`publish-image` action. GitHub Rulesets remain external repository policy.
