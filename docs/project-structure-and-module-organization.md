# Project Structure And Module Organization

Use only when placement is not forced by current code, package documentation,
generated-source ownership, or [Component Boundaries](architecture/boundaries.md).
Do not create a directory before its first real artifact.

## Placement

| Responsibility | Owner |
| --- | --- |
| Business behavior, types, ports, invariants, domain errors | `internal/<feature>/` |
| Concrete provider/transport adapters | `internal/infra/<provider>/` |
| Process composition and lifecycle | `cmd/<binary>/internal/bootstrap/` |
| Client-visible REST contracts | `api/openapi/service.yaml` |
| Protobuf contracts when enabled | `api/proto/` |
| Generated Go | Derived directory named by the owning generator; never manual authority |
| Runtime configuration | The existing `internal/config/<section>_config.go` owner |
| Ordinary behavior and boundary tests | Beside their package owner |
| Real external-process/container proof | `test/` with integration tag and external test package |
| Task decisions | `specs/<task>/` |
| Repository agent policy and methods | The narrow `docs/` or `.agents/skills/` owner |

There is no public `pkg/` until reusable external API ownership is explicitly
accepted. Empty `api/proto/`, `migrations/`, query, or generated directories are
not reserved. A new binary requires an independent process lifecycle; a new
network boundary, cache, queue, store, or shared package requires its accepted
architecture force.

## Non-Obvious Repository Rules

- Feature REST implementations stay in their feature-owned package and satisfy
  `openapi.StrictServerInterface`; bootstrap injects them through
  `httpx.Handlers.API`. Business handlers never move into the shared
  `internal/infra/http` template adapter.
- Directory `internal/infra/http` uses package `httpx`, and
  `internal/infra/grpc` uses `grpcx`, to avoid collisions with `net/http` and
  `google.golang.org/grpc` in their callers.
- Reusable runtime option mapping belongs in `cmd/internal/runtimeopts/` only
  when more than one composition root currently consumes it.
- Configuration section type, defaults, and validation stay together in
  `<section>_config.go`; tagged types own decoding and known-key discovery.
- Adapter-owned instruments stay with that adapter; a caller-provided telemetry
  provider alone does not justify `metrics.go` or `telemetry.go`.
- PostgreSQL driver and generated SQL imports remain in the file families
  allowed by `.golangci.yml`. In particular `store_<stage>.go` is a load-bearing
  ownership name, not a cosmetic convention.
- Shared test support stays unexported in `harness_test.go` for one package,
  `<owner-package>/<name>test/` for multiple packages with one natural owner, or
  `internal/<name>test/` only when no single package owns current consumers.
- `examples/reference-service/` is an isolated feature-slice example, never a
  production route or shared runtime owner.

Go filenames use lowercase snake case and one present reason to exist. Name the
owned behavior, not chronology. Reject `*_additional_test.go`,
`*_part2_test.go`, `test_helpers_test.go`, production `*_helpers.go`, `util.go`,
`common.go`, and `misc.go`. Use `<owner>_test.go`,
`<owner>_contract_test.go`, `<feature>_integration_test.go`, and `goleak_test.go`
only for those actual proof roles. Split by independent lifecycle, audience,
authority, or operator flow; line count alone is not an owner.

## Removable Profile Packs

<!-- profile:outbound-auth-oauth2-client-credentials:start -->
`internal/infra/oauth2clientcredentials` owns the fixed token endpoint,
context-aware cache, and authenticated HTTP/gRPC client factories. Concrete
provider adapters close the owner and give feature packages only generated
authenticated clients. It remains one removable outbound-auth pack.
<!-- profile:outbound-auth-oauth2-client-credentials:end -->

<!-- profile:jobs-postgres:start -->
`cmd/jobs-worker`, River, and River's Goose migration are one removable
PostgreSQL-jobs pack. Business packages define their typed arguments and workers
without a second job framework.
<!-- profile:jobs-postgres:end -->

<!-- profile:outbox-postgres:start -->
`internal/infra/postgresoutbox` and `cmd/outbox-relay` are the removable
River-backed outbox pack. Event meaning owns no broker address; NATS routing
remains in `internal/infra/natsjs`.
<!-- profile:outbox-postgres:end -->

<!-- profile:messaging-nats-jetstream:start -->
`internal/domainevent` and `internal/infra/natsjs` form the removable typed
messaging pack. The domain package owns typed event identity and encoding; the
adapter owns composition routes and JetStream mechanics.
<!-- profile:messaging-nats-jetstream:end -->

<!-- profile:webhooks-durable:start -->
`internal/infra/postgreswebhook` and the enabled jobs-worker surfaces form the
removable durable-webhook pack. The always-retained `internal/outboundtrust`
predicate is shared with fixed-target HTTP and owns no URL, resolver, dialer,
HTTP, or config policy.
<!-- profile:webhooks-durable:end -->

<!-- profile:object-storage:start -->
`internal/objectstorage/` owns the provider-neutral port and
`internal/infra/s3/` one fixed-authority S3-compatible adapter.
<!-- profile:object-storage:end -->

A profile-owned source, generated output, wiring, tests, config, documentation,
and replacement cleanup leave together. Partial profile capability is not a
supported placement.

## Generated And Proof Boundaries

| Source | Derived output | Drift proof |
| --- | --- | --- |
| `api/openapi/service.yaml` and generation config | `internal/openapi/openapi.gen.go` | `make openapi-check` |
| Example OpenAPI and generation config | example generated bindings | `make openapi-check` |
| `migrations/*.sql` and SQL query source | PostgreSQL generated access | `make sqlc-check` |
| `.agents/roles/` | Codex, Claude, and Qwen role carriers | `make agent-roles-check` |

Use the narrowest real proof owner. `_contract_test.go` is for an executable
boundary invariant, not a stronger-sounding unit test. `test/` remains reserved
for real database, container, external process, or multi-package black-box
proof; ordinary package tests do not require Docker.

`make project-structure-check`, depguard, generator drift checks, migration
checks, and package contract tests own enforced facts. If a proposal still has
no owner after this fallback, reopen Repository Architecture instead of creating
a generic service, helper, or shared directory.
