# Project Structure And Module Organization

This document is the normative owner of repository placement and file naming.
When a new artifact is needed, follow the decision tree below. Do not create a
directory until the first real artifact has an owner.

## 1. Structural model

The repository is feature-first for business behavior and layer-first only for
shared technical adapters:

```text
.
├── .agents/skills/
├── .claude/
├── .codex/
├── .qwen/
├── api/
│   └── openapi/service.yaml
├── build/
│   └── docker/Dockerfile
├── cmd/
│   ├── migrate/
│   │   ├── main.go
│   │   ├── run.go
│   │   └── run_test.go
│   └── service/
│       ├── main.go
│       └── internal/bootstrap/
├── docs/
├── env/
│   ├── config/
│   └── docker-compose.yml
├── examples/
│   └── reference-service/
├── internal/
│   ├── config/
│   ├── health/
│   ├── infra/
│   │   ├── http/
│   │   ├── postgres/
│   │   └── telemetry/
│   ├── observability/otelconfig/
│   └── openapi/
├── scripts/
├── specs/
└── test/
    └── performance/http/
```

Conditional paths are created only by the first real owner:

```text
internal/<feature>/                         business feature
migrations/                                schema migration source
internal/infra/postgres/queries/            sqlc query source
internal/infra/postgres/sqlcgen/            generated sqlc output
internal/infra/<outbound-system>/            outbound adapter
internal/infra/grpcclient/                   bounded shared gRPC client connection
cmd/<worker>/                               additional process
api/proto/                                  protobuf source
test/performance/http/<feature>.js           multi-step k6 scenario
```

There is no `pkg/`: this module does not currently promise reusable public Go
packages. There is no reserved empty `api/proto/`, `migrations/`, `queries/`, or
`sqlcgen/`. Add one only with its first owned source file.

## 2. Package ownership

| Path | Owns | Must not own |
| --- | --- | --- |
| `cmd/<binary>/` | process entry point; `main.go` plus binary-local orchestration | reusable business or adapter behavior |
| `cmd/service/internal/bootstrap/` | composition, startup, admission, shutdown, dependency wiring | domain policy, HTTP mapping, persistence queries |
| `internal/<feature>/` | business types, use cases, ports, invariants, domain errors | generated OpenAPI types, runtime config, concrete adapters |
| `internal/health/` | service readiness/drain behavior and probe interface | HTTP responses, Postgres construction, config loading |
| `internal/config/` | config schema, defaults, loading, parsing, validation, immutable snapshot | feature behavior and adapter construction |
| `internal/openapi/` | generated Go bindings and generation config | hand-written handlers or business logic |
| `internal/infra/http/` (`package httpx`) | HTTP mapping, router, middleware, Problem responses | SQL, database repositories, business decisions |
| `internal/infra/httpclient/` (`OUTBOUND_HTTP=bounded`) | outbound fixed-authority transport safety, correlation enforcement, and lifecycle | provider auth, concrete trust selection, retries, error mapping, or business policy |
| `internal/infra/grpcclient/` (`GRPC=enabled`) | bounded shared connections, correlation-policy enforcement, resolver metadata sanitization, and connection lifecycle seam | provider auth, concrete trust selection, operation deadlines or retries, generated-client ownership, or readiness policy |
| `internal/infra/postgres/` | pool, migrations, concrete repositories, query mapping | HTTP behavior and business policy |
| `internal/infra/telemetry/` | OpenTelemetry/Prometheus SDK setup and exporters | feature policy |
| `internal/observability/otelconfig/` | pure sampler/exporter policy values | SDK construction and repository runtime imports |
| `api/openapi/` | client-visible REST source of truth | generated Go or runtime handlers |
| `examples/reference-service/` | isolated runnable feature-slice example and its own generated contract | production service routes or shared runtime ownership |
| `migrations/` | ordered schema changes | query logic and sample placeholder schema |
| `test/` | container/external-process integration proof | ordinary package unit tests |
| `.agents/skills/<skill>/` | canonical agent skill | harness-specific copies |
| `.claude/`, `.codex/`, and `.qwen/` | harness adapters to canonical skills | independent skill authority |

`package httpx` intentionally differs from directory `http`: callers can import
it as `httpx` without colliding with the standard library `net/http`.

## 3. Deterministic placement algorithm

Use the first matching rule.

1. Is it the source of a client-visible REST contract?
   Put it in `api/openapi/service.yaml`. Generated Go goes only to
   `internal/openapi/openapi.gen.go`.
2. Is it business behavior for one capability?
   Put it in `internal/<feature>/`.
   - command/use case: `<verb>_<noun>.go`;
   - business types and invariants: `model.go`;
   - domain errors: `errors.go`;
   - consumer-owned persistence/outbound interface: `repository.go` or
     `client.go`.
3. Is it HTTP transport mapping?
   Put endpoint methods in `internal/infra/http/<feature>_handlers.go`.
   Put cross-route policy in `middleware_<concern>.go`; router composition stays
   in `router.go`; Problem mapping stays in `problem.go`.
4. Is it Postgres implementation?
   - concrete feature adapter:
     `internal/infra/postgres/<feature>_repository.go`;
   - migration:
     `migrations/NNNNNN_<feature>_<change>.up.sql` and matching `.down.sql`;
   - sqlc source: `internal/infra/postgres/queries/<feature>.sql`;
   - generated result: `internal/infra/postgres/sqlcgen/`.
5. Is it another outbound system?
   Put the concrete adapter in
   `internal/infra/<outbound-system>/client.go`; add
   `<feature>.go` only when several features map through that adapter.
6. Is it process construction or lifecycle?
   Existing service wiring goes in `cmd/service/internal/bootstrap/` under the
   matching `startup_<concern>.go`, `run.go`, or `shutdown.go`.
7. Is it a background process with an independent deployment/lifecycle?
   Create `cmd/<worker>/main.go`; keep binary-only wiring under
   `cmd/<worker>/internal/bootstrap/`. Reuse feature packages and adapters
   rather than duplicating them.
8. Is it runtime configuration?
   Add the typed field and `koanf` tag to `internal/config/types.go`, its
   default to `defaults.go` when appropriate, validation in `validate.go`, and
   the relevant example in `env/config/local.yaml` or `env/.env.example`.
   Snapshot decoding and known-key discovery derive from the tagged type; add
   manual parsing only for a genuinely custom value type.
9. Is it telemetry?
   SDK/exporter setup belongs in `internal/infra/telemetry/`. Adapter-owned
   instruments belong in `internal/infra/<adapter>/metrics.go` or `tracing.go`.
   Feature-owned signals belong in `internal/<feature>/telemetry.go`.
10. Is it a test?
    - package behavior: sibling `<owner>_test.go`;
    - executable boundary contract: sibling `<owner>_contract_test.go`;
    - container/external process: `test/<feature>_integration_test.go` with
      `//go:build integration` and `package integration_test`;
    - fake used by one test file: keep it in that file;
    - fake shared within one package: keep it unexported in `<package>_test.go`;
    - non-trivial test support used by two or more current packages:
      `<owner-package>/<owner-name>test/`, as in
      `internal/infra/telemetry/telemetrytest/`; production code must not import it;
    - package-wide goroutine leak gate: sibling `goleak_test.go` with `TestMain`.
11. Is it HTTP load proof?
    A single configurable request uses
    `test/performance/http/single-flow.js`. A genuinely multi-step feature flow
    goes to `test/performance/http/<feature>.js`.
12. Is it agent guidance?
    A reusable skill goes to `.agents/skills/<skill>/SKILL.md`. Repository
    policy goes to the owning `docs/<topic>.md`; task decisions go to
    `specs/<task>/`. Do not duplicate canonical instructions into harness
    adapters.

If none matches, the artifact has no accepted owner. Stop and update this
document plus its enforcement before adding the file.

## 4. File naming and granularity

All Go filenames use lowercase snake case. Name files after owned behavior, not
chronology or editing history.

| Artifact | Required form |
| --- | --- |
| middleware | `middleware_<concern>.go` and matching `_test.go` |
| startup policy | `startup_<concern>.go` and matching `_test.go` |
| feature HTTP handlers | `<feature>_handlers.go` |
| Postgres repository | `<feature>_repository.go` |
| normal unit test | `<owner>_test.go` |
| executable boundary contract | `<owner>_contract_test.go` |
| package-wide test helpers | `<package>_test.go` |
| cross-package test support | `<owner-name>test/` under its owning package |
| package-wide goroutine leak gate | `goleak_test.go` |
| integration test | `<feature>_integration_test.go` |
| package documentation/generation directive | `doc.go` |

Forbidden names:

- `*_additional_test.go`, `*_part2_test.go`, and other ordinal/history suffixes;
- `test_helpers_test.go`;
- production `*_helpers.go`, `util.go`, `common.go`, or `misc.go`.

Split a file when its declarations have different owners named by this table.
Do not split merely to hit a line-count target. `network_policy.go`,
`network_policy_parsing.go`, and `network_policy_enforcement.go` are separate
because state, parsing, and enforcement are independent policies.

Bootstrap is intentionally one package because its declarations jointly build
one process. Its production files are:

```text
network_policy.go
network_policy_enforcement.go
network_policy_parsing.go
run.go
shutdown.go
startup.go
startup_admission.go
startup_config.go
startup_dependencies.go
startup_logging.go
startup_rejections.go
startup_server.go
startup_span.go
startup_telemetry.go
startup_timing.go
```

Do not create a bootstrap subpackage unless behavior becomes reusable outside
that binary; file size alone is not a package boundary.

## 5. Generated authority

| Source of truth | Generated/derived output | Proof |
| --- | --- | --- |
| `api/openapi/service.yaml` + `internal/openapi/oapi-codegen.yaml` | `internal/openapi/openapi.gen.go` | `make openapi-check` |
| `examples/reference-service/api/openapi.yaml` + example generation config | `examples/reference-service/internal/openapi/openapi.gen.go` | `make openapi-check` |
| `migrations/*.up.sql` + `internal/infra/postgres/queries/*.sql` | `internal/infra/postgres/sqlcgen/` | `make sqlc-check` |
| `.agents/skills/` | harness adapters | `make template-init-check` where applicable |

Never hand-edit generated Go. With no owned migrations or queries, their source
and generated directories are absent and the migration/sqlc commands
intentionally no-op.

## 6. Test level

Use the narrowest real owner:

- a pure function, use case, mapper, middleware, or adapter policy is tested
  beside its package;
- `_contract_test.go` is reserved for an executable boundary or architecture
  invariant, not stronger-sounding unit tests;
- `test/` is reserved for proof requiring a real database, container, external
  process, or multiple packages as black boxes;
- build-tagged integration tests must be deterministic and isolated; ordinary
  package tests must not require Docker.

## 7. Enforcement

| Rule | Enforcement |
| --- | --- |
| feature/config/health import direction | depguard in `.golangci.yml` |
| HTTP cannot import Postgres | depguard |
| sqlc types stay behind Postgres adapter | depguard |
| chi stays in OpenAPI/HTTP owners | depguard |
| `otelconfig` stays pure and SDK-free | depguard |
| no historical/helper filenames | `make project-structure-check` |
| command directories contain `main.go` | `make project-structure-check` |
| integration suffix/tag/package | `make project-structure-check` |
| no empty speculative source/generated directories | `make project-structure-check` |
| migration up/down pairing | `make project-structure-check` |
| OpenAPI/sqlc generated drift | `make openapi-check`, `make sqlc-check` |
| runtime OpenAPI route ownership | package `_contract_test.go` tests |
| all remaining placement and naming decisions | this document; review risk |

The structure check runs in local aggregates and CI. Prose-only rules are an
explicit residual risk; add executable enforcement when a repeated drift is
observed.

## 8. Ten placement examples

| Requested artifact | One result |
| --- | --- |
| `CreateOrder` use case | `internal/orders/create_order.go` |
| order domain error | `internal/orders/errors.go` |
| `POST /orders` handler | `internal/infra/http/orders_handlers.go` |
| order Postgres adapter | `internal/infra/postgres/orders_repository.go` |
| create orders table | `migrations/000001_orders_create.up.sql` plus `.down.sql` |
| order sqlc operations | `internal/infra/postgres/queries/orders.sql` |
| request authentication middleware | `internal/infra/http/middleware_authentication.go` |
| Stripe outbound client | `internal/infra/stripe/client.go` |
| order DB integration proof | `test/orders_integration_test.go` |
| order-processing worker | `cmd/order-worker/main.go` |

Every example resolves through exactly one first-matching rule. A proposal such
as “shared helpers” or “generic service layer” does not resolve and is rejected
until it names a concrete owner.

## 9. First production feature checklist

1. Name one feature owner and create `internal/<feature>/` only with its first
   use case.
2. Decide the public/security behavior in the task specification, then update
   `api/openapi/service.yaml`.
3. Add feature behavior and consumer-owned ports before concrete adapters.
4. Add `<feature>_handlers.go`; do not register a manual business route.
5. If persistence is required, add the first paired migration, query source,
   generated sqlc output, and hand-written repository in that order.
6. Add config, telemetry, startup wiring, and cleanup only for dependencies the
   feature actually uses.
   For an outbound gRPC dependency, bootstrap owns one reusable
   `grpcclient` connection and closes it; the dependency adapter owns the
   generated client and selects `PropagationNone`, `PropagationTraceContext`,
   or `PropagationTrustedService` from the accepted trust boundary. No policy
   propagates baggage, and the shared connection deliberately supports neither
   environment proxies nor resolver-provided service configs.
7. Add sibling unit/contract tests and `test/<feature>_integration_test.go` only
   when real infrastructure is part of the claim.
8. Run the matching generators, `make project-structure-check`, focused tests,
   and the publication gate required by the change.

## 10. Source basis

Primary guidance:

- [Organizing a Go module](https://go.dev/doc/modules/layout)
- [Go Wiki: package names](https://go.dev/wiki/CodeReviewComments#package-names)
- [Google Go Style Guide: package names](https://google.github.io/styleguide/go/decisions#package-names)

Community implementations are evidence, not authority:

- [ardanlabs/service](https://github.com/ardanlabs/service) groups business
  capabilities separately from platform adapters.
- [Grafana k6](https://github.com/grafana/k6) keeps commands and internal
  implementation ownership explicit in a large production Go module.
- [GoogleCloudPlatform/microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo)
  demonstrates per-service ownership rather than one repository-wide framework
  layer stack.

`golang-standards/project-layout` is not an official Go standard and is not a
source of authority for this repository.
