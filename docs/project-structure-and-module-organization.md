# Project Structure And Module Organization

<!-- profile:outbound-auth-oauth2-client-credentials:start -->

`internal/infra/oauth2clientcredentials` owns the fixed machine-credential
client, cache, wire contract, and transport adapters. Its HTTP adapter uses the
bounded `internal/infra/httpclient` path; its gRPC adapter wraps a concrete
`grpc.ClientConnInterface`. Bootstrap alone maps `internal/config` into this
runtime owner.

<!-- profile:outbound-auth-oauth2-client-credentials:end -->

<!-- profile:jobs-postgres:start -->
`internal/jobs`, `internal/infra/postgresjobs`, and `cmd/jobs-worker` are one
removable profile pack; generic job mechanics never move into feature packages.
<!-- profile:jobs-postgres:end -->

<!-- profile:webhooks-durable:start -->
`internal/outboundtrust` is a standard-library-only public-address predicate
shared by fixed-target HTTP and dynamic webhook transport. It owns no URL,
resolver, dialer, HTTP, or config policy. `internal/infra/postgreswebhook` and
`cmd/webhook-worker` own the removable durable webhook pack.
<!-- profile:webhooks-durable:end -->

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
│   ├── failure/
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
internal/infra/grpc/                         native gRPC server transport
internal/infra/grpcclient/                   bounded shared gRPC client connection
cmd/<worker>/                               additional process
cmd/internal/runtimeopts/                   adapter options more than one binary maps
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
| `cmd/internal/runtimeopts/` | the mapping from one loaded configuration onto the adapter options more than one composition root builds | startup flow, readiness, drain, or any option a single binary builds alone |
| `cmd/service/internal/bootstrap/` | composition, startup, admission, shutdown, dependency wiring | domain policy, HTTP mapping, persistence queries |
| `internal/<feature>/` | business types, use cases, ports, invariants, domain errors | generated OpenAPI types, runtime config, concrete adapters |
| `internal/health/` | service readiness/drain behavior and probe interface | HTTP responses, Postgres construction, config loading |
| `internal/config/` | config schema, defaults, loading, parsing, validation, immutable snapshot | feature behavior and adapter construction |
| `internal/failure/` | transport-neutral failure codes, safe classification, and mapper ordering | HTTP status catalogs, gRPC status codes, feature error identities, or I/O |
| `internal/openapi/` | generated Go bindings and generation config | hand-written handlers or business logic |
| `internal/infra/http/` (`package httpx`) | HTTP mapping, router, middleware, Problem responses | SQL, database repositories, business decisions |
| `internal/infra/httpclient/` (`OUTBOUND_HTTP=bounded`) | outbound fixed-authority transport safety, correlation enforcement, retry mechanism, and lifecycle | provider auth, concrete trust selection, retry eligibility, error mapping, or business policy |
<!-- profile:object-storage:start -->
| `internal/objectstorage/` and `internal/infra/s3/` (`OBJECT_STORAGE=s3`) | provider-neutral object port and one explicit-snapshot Amazon S3/Cloudflare R2 adapter | feature policy, ambient/refreshed credentials, bucket provisioning, interchangeable provider claims, or generated trust configuration |
<!-- profile:object-storage:end -->
| `internal/infra/grpc/` (`package grpcx`, `GRPC=enabled`) | native gRPC server composition, standard health, the interceptor policy chain, transport bounds, and status mapping | feature packages, generated handlers, domain policy, authentication decisions, or this repository's configuration shape |
| `internal/infra/grpcclient/` (`GRPC=enabled`) | bounded shared connections, correlation-policy enforcement, resolver metadata sanitization, address selection, standard-health eligibility, opt-in idle keepalive, and the connection lifecycle seam | provider auth, concrete trust selection, operation deadlines or application retries, generated-client ownership, or dependency readiness policy |
| `internal/infra/oidcjwt/` (`AUTHN=oidc-jwt`) | inbound caller identity: OIDC trust bootstrap, JWKS lifecycle, token admission, and the HTTP and gRPC authentication adapters | authorization, roles, tenant policy, sessions, user provisioning, or any decision past who the caller is |
| `internal/infra/postgres/` | pool, concrete repositories, query mapping | HTTP behavior, migration execution, and business policy |
<!-- profile:webhooks-durable:start -->
| `internal/outboundtrust/` | pure public/special IP classification shared by enabled outbound consumers | URLs, DNS, dialing, HTTP, configuration, or webhook policy |
| `internal/infra/postgreswebhook/` (`WEBHOOKS=durable`) | immutable acceptance values, PostgreSQL webhook state, claim/send/finality, bounded maintenance, signing, and telemetry | feature mutation, subscriber discovery, operator transport, receiver processing, or deployment |
| `cmd/webhook-worker/` (`WEBHOOKS=durable`) | webhook worker config mapping, diagnostics, readiness, drain, and dependency cleanup | webhook state policy or HTTP API routes |
<!-- profile:webhooks-durable:end -->
<!-- profile:inbox-postgres:start -->
| `internal/infra/postgresinbox/` (`INBOX=postgres`) | validate and insert one `(consumer identity, logical message ID)` claim through a caller-owned `pgx.Tx` | transaction lifecycle, feature effects, transport configuration, expiry, cleanup, telemetry, or ordering |
<!-- profile:inbox-postgres:end -->
| `internal/infra/postgresmigrate/` | Goose lifecycle, source/state admission, lock, and migration result metadata | service startup, domain policy, and production rollback commands |
| `internal/infra/telemetry/` | OpenTelemetry/Prometheus SDK setup and exporters | feature policy |
| `internal/observability/otelconfig/` | pure sampler/exporter policy values | SDK construction and repository runtime imports |
| `internal/observability/correlationpolicy/` | the outbound correlation policy enum and propagator shared by the HTTP and gRPC clients | carrier-specific stripping, transport construction, or repository runtime imports |
| `internal/observability/logctx/` | the shared process logger and the handler that attaches request/trace/span ids from context | feature log fields, sinks, or repository runtime imports |
| `internal/grpclimits/` (`GRPC=enabled`) | pure gRPC access-log and lifetime bound rules shared by config validation and the server adapter | capacity bounds, error wording, or repository runtime imports |
| `internal/authntrust/` (`AUTHN=oidc-jwt`) | pure trust-rule predicates shared by config validation and the verifier: fetchable provider URLs, trusted proxy prefixes | configured values, policy objects, credential verification, or repository runtime imports |
| `api/openapi/` | client-visible REST source of truth | generated Go or runtime handlers |
| `examples/reference-service/` | isolated runnable feature-slice example and its own generated contract | production service routes or shared runtime ownership |
| `examples/grpc-reference-service/` (`GRPC=enabled`) | isolated runnable example of the four gRPC method cardinalities, its own generated contract, and the benchmark server | production service registration, shared runtime ownership, or transport policy |
| `migrations/` | ordered schema changes | query logic and sample placeholder schema |
| `test/` | container/external-process integration proof | ordinary package unit tests |
| `.agents/skills/<skill>/` | canonical agent skill | harness-specific copies |
| `.claude/`, `.codex/`, and `.qwen/` | harness adapters to canonical skills | independent skill authority |

`package httpx` intentionally differs from directory `http`, and `package grpcx`
from directory `grpc`, for the same reason: callers import each without colliding
with the library every file in it is written against — `net/http` and
`google.golang.org/grpc`.

## 3. Deterministic placement algorithm

Use the first matching rule.

1. Is it the source of a client-visible REST contract?
   Put it in `api/openapi/service.yaml`. Generated Go goes only to
   `internal/openapi/openapi.gen.go`.
2. Is it business behavior for one capability?
   Put it in `internal/<feature>/`.
   - command/use case: `<verb>_<noun>.go`;
   - business types and invariants: `model.go`;
   - domain errors and their transport-neutral classifier: `errors.go`;
   - consumer-owned persistence/outbound interface: `repository.go` or
     `client.go`.
3. Is it HTTP transport mapping?
   Endpoint methods do not go in this package. `internal/infra/http` is template
   surface every generated service shares; a service implements
   `openapi.StrictServerInterface` in its own package and injects it as
   `httpx.Handlers.API` from `cmd/service/internal/bootstrap`. See
   `examples/reference-service/internal/httpapi` for the worked pattern, and the
   `API` field comment in `internal/infra/http/handlers.go` for the seam itself.
   Only cross-route policy belongs here: `middleware_<concern>.go`; generated
   router composition stays in `router.go`, the reusable chain in `harden.go`,
   and Problem mapping in `problem.go`.
4. Is it Postgres implementation?
   - concrete feature adapter:
     `internal/infra/postgres/<feature>_repository.go`;
   - migration:
     `migrations/NNNNNN_<feature>_<change>.sql` with one Goose `Up` and one
     `Down` section;
   - sqlc source: `internal/infra/postgres/queries/<feature>.sql`;
   - generated result: `internal/infra/postgres/sqlcgen/`.
   <!-- profile:inbox-postgres:start -->
   - transport-neutral inbox claim bound to the caller's transaction:
     `internal/infra/postgresinbox/`.
   <!-- profile:inbox-postgres:end -->
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
   Add the typed field and `koanf` tag, its default when appropriate, and its
   validation to the section's own `internal/config/<section>_config.go`, plus
   the relevant example in `env/config/local.yaml` or `env/.env.example`. One
   key is one file, so the reason a value was chosen sits beside the rule that
   enforces it and a section a build profile removes leaves with its file.
   `types.go` and `defaults.go` hold only the `Config` shape, the merge over it,
   and the sections every profile keeps (`app`, `health`, `log`, `runtime`);
   `validate.go` owns only the order sections run in and the rules that hold
   between them.
   Snapshot decoding and known-key discovery derive from the tagged type; add
   manual parsing only for a genuinely custom value type.
9. Is it telemetry?
   SDK/exporter setup belongs in `internal/infra/telemetry/`. Adapter-owned
   instruments belong in `internal/infra/<adapter>/metrics.go` or `tracing.go`,
   or in one `telemetry.go` when that adapter's metrics, spans, and structured
   logging are a single type with one constructor and one snapshot, so that one
   record call moves a counter and closes a span together. An adapter that
   builds no instruments of its own owns neither name: when it only forwards a
   caller's `MeterProvider` to an instrumentation library, the delegation is
   proved in `telemetry_test.go` and no `metrics.go` or `telemetry.go` is
   created for it. Feature-owned signals belong in
   `internal/<feature>/telemetry.go`.
10. Is it a test?
    - package behavior: sibling `<owner>_test.go`;
    - executable boundary contract: sibling `<owner>_contract_test.go`;
    - container/external process: `test/<feature>_integration_test.go` with
      `//go:build integration` and `package integration_test`;
    <!-- profile:http-idempotency-postgres:start -->
    - sole package-local integration exception:
      `cmd/service/internal/bootstrap/startup_idempotency_integration_test.go`
      stays tagged `integration` in `package bootstrap` because it must exercise
      unexported service composition against the real PostgreSQL writer; no
      other `cmd/**/_integration_test.go` is permitted;
    <!-- profile:http-idempotency-postgres:end -->
    - fake used by one test file: keep it in that file;
    - fake or harness shared within one package: keep it unexported in
      `harness_test.go`, which does not collide with the `<owner>_test.go` of a
      same-named production file the way `<package>_test.go` would;
    - non-trivial test support used by two or more current packages:
      `<owner-package>/<owner-name>test/`, as in
      `internal/infra/telemetry/telemetrytest/`; production code must not import it;
    - the same support with no single owning package, because three or more
      unrelated packages need it: `internal/<name>test/`, as in
      `internal/waittest/`; the `test` suffix keeps the name self-describing the
      way `httptest` and `iotest` do, and nesting it under one arbitrary
      consumer would misname the owner;
    - integration tests under `test/`: every file needs the
      `_integration_test.go` suffix, including one that holds only fixtures, so
      name it `<feature>_fixtures_integration_test.go`;
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

The forms below are in two groups. The first is prescriptive: it names what a
service built on this template writes, and the template itself carries no
instance of those rows. The second is descriptive: it names the prefix families
this repository's own packages already use, so that adding a file to one of them
has a rule to follow rather than a neighbour to imitate.

Forms a service writes:

| Artifact | Required form |
| --- | --- |
| feature HTTP handlers | `<feature>_handlers.go`, in the service's own package behind `httpx.Handlers.API` — never in `internal/infra/http`. The platform's own probe handlers are not feature handlers; `internal/infra/http/health_handlers.go` is the one file of this shape that belongs there |
| Postgres repository | `<feature>_repository.go` |
| domain types | `model.go`, or `<feature>.go` when one file already owns the feature's types and behavior — as `examples/reference-service/internal/article/article.go` does |

Forms this repository's own packages use:

| Artifact | Required form |
| --- | --- |
| middleware | `middleware_<concern>.go` and matching `_test.go` |
| startup policy | `startup_<concern>.go` and matching `_test.go` |
| gRPC interceptor | `interceptor_<concern>.go`, one policy per file; `chain.go` is the single owner of their order |
| configuration section | `<section>_config.go`, holding that section's type, its defaults, and its validation together — see rule 8 |
| durable-store access | `store_<stage>.go`. In `internal/infra/postgres*` this prefix is load-bearing, not stylistic: `.golangci.yml` exempts driver and `sqlc` imports by the `store*.go` and `notify*.go` globs, and `docs_test.go` fails if a driver-importing file matches no glob |
| relay cycle stage | `relay_<stage>.go`, the driver-free half of the same package |
| broker worker stage | `worker_<stage>.go`, split by lifetime: construction-time topology proof, per-message delivery, and the run loop are separate files |
| broker wire contract | `message_<aspect>.go`, with the encode/decode pair and every header constant in one file |
| configuration loader | `load_<source>.go`, one input source per file |
| closed label vocabulary | `vocabulary.go` — every literal the package puts on a metric attribute or an operator log field, in one place because an unrecognized value mints a time series |
| normal unit test | `<owner>_test.go` |
| executable boundary contract | `<owner>_contract_test.go` |
| package-wide test helpers or fakes | `harness_test.go` |
| cross-package test support | `<owner-name>test/` under its owning package, or `internal/<name>test/` when no single package owns it |
| integration fixtures shared by one `test/` family | `<feature>_fixtures_integration_test.go` |
| package-wide goroutine leak gate | `goleak_test.go` |
| integration test | `<feature>_integration_test.go` |
| package documentation/generation directive | `doc.go` |

Forbidden names:

- `*_additional_test.go`, `*_part2_test.go`, and other ordinal/history suffixes;
- `test_helpers_test.go`;
- production `*_helpers.go`, `util.go`, `common.go`, or `misc.go`.

Give each Go file one present reason to exist: its declarations change for the
same reason and are normally read together on one execution or proof path. A
shared package or receiver is not enough. Split declarations whose lifecycle
stage, audience, authority, or operator flow changes independently; when those
pressures coexist in one file, record why co-location makes the path easier to
trace. Treat line count only as a signal to inspect ownership, never as a split
criterion. `network_policy.go`, `network_policy_parsing.go`, and
`network_policy_enforcement.go` are separate because state, parsing, and
enforcement are independent policies.

Add `doc.go` when a package has multiple reader audiences, lifecycle stages,
extension paths, or a deliberately absent seam that a maintainer could
reasonably expect. State that package contract and the non-obvious constraint;
omit `doc.go` when exported APIs and filenames already make both apparent.
Package documentation supplements the structure: when it is the only way to
discover which file owns ordinary behavior, or it compensates for unclear names
or placement, repair the file map instead.

Bootstrap is intentionally one package because its declarations jointly build
one process. `cmd/service/internal/bootstrap`'s production files are:

```text
run.go
runtime_memory_limit.go
runtime_request_buffer_budget.go
shutdown.go
startup.go
startup_admission.go
startup_authn.go
startup_config.go
startup_dependencies.go
startup_diagnostics.go
startup_grpc.go
startup_grpc_tls.go
startup_http.go
startup_logging.go
startup_messaging.go
startup_readiness.go
startup_server.go
startup_telemetry.go
startup_timing.go
```

Several of those files belong to one build profile and leave with it, so a
generated service holds a subset. `run.go` keeps the startup order and the
defer-ordered teardown, because that sequence is the behavior: a stage moved
behind a helper stops being checkable against the order it must run in. Each
`startup_<concern>.go` beside it owns one stage's construction, and the two
transports are deliberately symmetric — `startup_http.go` and `startup_grpc.go`
each pair a bindings type with the composition that consumes it.

The `startup_` prefix is what this package needs, not a repository-wide rule. It
earns its place here because `run.go` and `shutdown.go` sit in the same
directory, so the prefix is what says a file builds the process rather than
tears it down. A background binary's bootstrap package draws the same
one-concern-per-file line with bare nouns — `config.go`, `lifecycle.go`,
`diagnostics.go`, `telemetry.go` — because it has no separate `shutdown.go` to
disambiguate against: its drain lives in `lifecycle.go` beside the run loop that
owns it. A bootstrap package that splits startup from teardown across files
takes the prefix; one that does not names each file for its concern alone.
Runtime wiring that more than one binary needs belongs in
`cmd/internal/runtimeopts/` rather than being written once per composition root.

Do not create a bootstrap subpackage unless behavior becomes reusable outside
that binary; file size alone is not a package boundary.

## 5. Generated authority

| Source of truth | Generated/derived output | Proof |
| --- | --- | --- |
| `api/openapi/service.yaml` + `internal/openapi/oapi-codegen.yaml` | `internal/openapi/openapi.gen.go` | `make openapi-check` |
| `examples/reference-service/api/openapi.yaml` + example generation config | `examples/reference-service/internal/openapi/openapi.gen.go` | `make openapi-check` |
| Goose `Up` sections in `migrations/*.sql` + `internal/infra/postgres/queries/*.sql` | `internal/infra/postgres/sqlcgen/` | `make sqlc-check` |
| `.agents/skills/` | harness adapters | `make template-init-check` where applicable |

Never hand-edit generated Go. With no owned migrations or queries, their source
and generated directories are absent. SQLC reports that there is nothing to
generate; migration source checks accept the pre-first-migration state, while
runtime rehearsal still connects and verifies that database state is empty.

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
| `authntrust` stays a pure rule leaf (`AUTHN=oidc-jwt`) | depguard |
| the `oidcjwt` core stays transport-neutral outside its two adapter files (`AUTHN=oidc-jwt`) | depguard |
| no historical/helper filenames | `make project-structure-check` |
| command directories contain `main.go` | `make project-structure-check` |
| integration suffix/tag/package | `make project-structure-check` |
| no empty speculative source/generated directories | `make project-structure-check` |
| canonical Goose source and append-only history | `make migration-check` |
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
| `POST /orders` handler | `internal/httpapi/orders_handlers.go`, injected as `httpx.Handlers.API` — never `internal/infra/http` |
| order Postgres adapter | `internal/infra/postgres/orders_repository.go` |
| create orders table | `migrations/000001_orders_create.sql` with Goose `Up` and `Down` sections |
| order sqlc operations | `internal/infra/postgres/queries/orders.sql` |
| service-owned request middleware | `internal/infra/http/middleware_<name>.go` (with `AUTHN=oidc-jwt`, caller authentication is already owned by `internal/infra/oidcjwt/`) |
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
5. If persistence is required, add the first single-file Goose migration,
   query source, generated sqlc output, and hand-written repository in that
   order.
6. Add config, telemetry, startup wiring, and cleanup only for dependencies the
   feature actually uses.
   For an outbound gRPC dependency, bootstrap owns one reusable
   `grpcclient` connection and closes it; the dependency adapter owns the
   generated client and selects `PropagationNone`, `PropagationTraceContext`,
   or `PropagationTrustedService` from the accepted trust boundary. No policy
   propagates baggage, and the shared connection deliberately supports neither
   environment proxies nor resolver-provided service configs. Round robin
   follows standard whole-process health by default; disable it only for a
   dependency that lacks that contract. If the dependency protects
   `Health/Watch`, supply its provider-owned credential through the connection;
   a per-call credential cannot authenticate that control stream. Enable idle keepalive only with a
   named intermediary timeout and a complete peer-compatible interval/timeout
   pair.
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
