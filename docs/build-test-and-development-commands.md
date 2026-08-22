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
make template-init-check
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
  `OBJECT_STORAGE=s3`, and
  `OUTBOUND_AUTH=oauth2-client-credentials` with HTTP, gRPC, or both.

Other cross-products are not a supported preset merely because the initializer
can mechanically resolve their markers.

`init-module.sh` validates selections and identity before mutation, rewrites
module/CODEOWNERS/service identity, physically removes unselected profiles,
regenerates retained outputs, tidies modules, writes `template.lock`, and is
idempotent for the same selection. The contract check uses invalid synthetic
inputs and one default generated checkout; profile-owned Go tests retain their
own behavior.

## Ordinary Go leaves

```bash
make fmt-check
make lint
make test
```

Authoring commands remain separate:

```bash
make fmt
make tidy
make mod-tidy-check
make mod-verify
make test-race
```

Use `make lint-deep` only for whole-program dead-code and nil analysis. Use
`make test-race` only when the claim spans concurrency-sensitive code.

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
make test-integration
REQUIRE_DOCKER=1 make test-integration
```

The second form is required evidence: Docker unavailability fails instead of
skipping.

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
bash scripts/ci/instruction-evals-check.sh
```

`scripts/harness-skills-sync.sh` is the one Claude/Qwen skill-view owner.
`scripts/template-sync.sh` checks or applies one committed template snapshot to
one target and leaves the result in that target's working tree.

## Delivery and security

```bash
make actionlint
make shellcheck
make dockerfile-check
make govulncheck
make gosec
make secret-scan BASE_REF=origin/main
make secret-scan-history
```

Gitleaks runs directly from the tools module. Pull-request proof scans both the
current tree and exact base-to-HEAD history; push/release proof scans full
history.

## Runtime images

```bash
make runtime-image-build RUNTIME_IMAGE=service:ci
make container-security CONTAINER_IMAGE=service:ci
```

The source template builds one documented PostgreSQL generated output. A
derived repository builds its own exact source. Reuse the same image tag for
migration rehearsal and vulnerability scanning.

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

`ci.yml` owns fast quality, security, and delivery contexts. `integration.yml`
uses native path filters for PostgreSQL/image owners. `cd.yml` consumes
successful exact-SHA CI and publishes through the single digest-bound
`publish-image` action. GitHub Rulesets remain external repository policy.
