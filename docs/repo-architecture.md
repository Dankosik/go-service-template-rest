# Repository Architecture Baseline

This document is the stable repository-wide current-state architecture baseline for `go-service-template-rest`.
Use it as evidence to recover standing constraints, owners, reusable capabilities, and runtime flows before writing task-local design in `specs/<feature-id>/design/`. [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) owns deciding which affected surfaces the target retains, replaces, or removes.

This file is intentionally narrower than:
- [Project Structure & Module Organization](./project-structure-and-module-organization.md)
- [Configuration Source Policy](./configuration-source-policy.md)
- [Build, Test, and Development Commands](./build-test-and-development-commands.md)
- [CI/CD Production-Ready Checklist](./ci-cd-production-ready.md)

It does not restate the full tree, every command, or task-local design choices.

## Stable Component Boundaries

| Area | Owns | Does not own |
| --- | --- | --- |
| `cmd/service/main.go` | Thin process entrypoint. Delegates immediately to bootstrap. | Business logic, request handling, dependency details. |
| `cmd/service/internal/bootstrap/` | Service composition root, startup/shutdown flow, config/bootstrap lifecycle, dependency admission, runtime policy. | Use-case semantics, transport contract definition, persistence logic. |
| `internal/config/` | Building one validated, immutable runtime config snapshot from defaults, config files, env, and flags. | Feature behavior, dependency wiring, request handling. |
| `api/openapi/service.yaml` | Source of truth for the REST contract. | Hand-written runtime logic or transport implementation. |
| `internal/openapi/` | Generated Go bindings derived from the OpenAPI contract. | Manual business logic; hand-editing should not become the source of truth. |
| `internal/<feature>/` | Feature-owned use cases, business types, ports, invariants, and domain errors. | HTTP details, driver details, runtime config, process lifecycle. |
| `internal/failure/` | Transport-neutral client-visible failure codes and mapper ordering shared by features and transports. | Feature error identities, HTTP envelopes, gRPC statuses, or I/O. |
| `internal/infra/http/` | HTTP server, middleware, request/response mapping, route policy, and observability at the transport edge. | Core business rules or config loading. |
| `internal/infra/httpclient/` (`OUTBOUND_HTTP=bounded`) | Optional shared outbound target validation, transport bounds, explicit correlation-policy enforcement, and idle-pool cleanup. | Provider authentication, concrete trust selection, operation budgets, retries, error mapping, or readiness policy. |
| `internal/infra/postgres/` | Optional Postgres connection/pool lifecycle and repository code. | Process lifecycle, migrations, HTTP behavior, config precedence rules. |
| `internal/infra/postgresmigrate/` | Optional migration execution used by `cmd/migrate`. | Runtime pool ownership or application startup. |
| `internal/infra/telemetry/` | OpenTelemetry tracing/metrics SDK setup and Prometheus export. | Feature semantics, startup logging, or request routing decisions. |
| `internal/observability/otelconfig/` | Narrow shared OTel config vocabulary, defaults, and pure validation helpers used by config and telemetry. | Config loading, OTel SDK construction, exporter setup, or generic observability helpers. |
<!-- profile:authn-oidc-jwt:start -->
| `internal/authntrust/` | The two deployment trust rules an authentication boundary is configured with — which provider URLs may be fetched, which peers may state a request's transport — as pure predicates shared by `internal/config` and `internal/infra/oidcjwt`. | Config loading, credential verification, policy objects, or any configured value of its own. |
<!-- profile:authn-oidc-jwt:end -->
| `migrations/` | SQL schema migration source of truth. | Runtime repository logic or generated Go bindings. |
<!-- profile:outbox-postgres:start -->
| `internal/infra/postgresoutbox/` | PostgreSQL outbox envelope, transactional append, claims, retries, poison/redrive, retention, and broker-neutral relay loop. | Domain event selection, a broker adapter, inbox processing, or exactly-once delivery. |
| `cmd/outbox-relay/` | Separate relay composition, readiness, drain, and dependency cleanup. | API routes or a fallback/noop publisher. |
<!-- profile:outbox-postgres:end -->
<!-- profile:inbox-postgres:start -->
| `internal/infra/postgresinbox/` | Stateless claim binding for one caller-owned PostgreSQL transaction. | Transaction lifecycle, feature effects, transport identity construction, cleanup, or ordering. |
<!-- profile:inbox-postgres:end -->

<!-- profile:grpc:start -->
The gRPC profile adds four boundaries: `api/proto/` owns protobuf contracts,
`internal/gen/proto/` contains derived messages and interfaces,
`internal/infra/grpc/` owns native server policy and lifecycle, and
`internal/infra/grpcclient/` constructs bounded shared client connections and
enforces explicit outbound correlation and client-owned routing policy. Its
zero propagation policy retains local telemetry but emits no remote correlation;
the trace-only and trusted-service policies are selected per dependency, and
none propagates baggage. Round robin consumes standard whole-process health by
default; idle keepalive is an explicit complete interval/timeout pair. The
client also disables environment proxies and resolver-provided service configs
so they cannot bypass metadata enforcement or install another routing/retry
policy. None of these packages owns feature semantics, storage schemas,
authentication policy, per-operation deadlines, application retry eligibility,
dependency criticality, or the trust decision for a concrete neighbor.
<!-- profile:grpc:end -->

## Domain Vocabulary

Keep only accepted cross-task terms whose interpretation changes behavior,
violation outcome, authority, proof, or handoff. Task-local or unsettled terms
stay in their owning specification; endpoint, table, queue, and package names
do not become domain vocabulary merely because they recur.

| Term | Means here | Does not mean | Authority source | Semantic owner | Decision affected |
| --- | --- | --- | --- | --- | --- |

The template defines no service-specific terms. A derived service adds rows as
stable domain decisions emerge.

## Source-Of-Truth Ownership

Keep these ownership rules stable across tasks:

| Source of truth | Derived or consuming surfaces |
| --- | --- |
| `api/openapi/service.yaml` | `internal/openapi/` generated bindings and `internal/infra/http/` transport wiring |
| `internal/config/` snapshot build + validation rules | Runtime config consumed by bootstrap and adapters |
| `env/config/*.yaml`, `APP__...`, and runtime flags | Inputs to `internal/config/`; precedence and secret rules live in [Configuration Source Policy](./configuration-source-policy.md) |
| `migrations/*.sql` | Database shape used by Postgres runtime code and any generated SQL access layer |
| `internal/<feature>/*` behavior | Consumed by HTTP handlers now; reusable by future binaries or async workers |
| `cmd/service/internal/bootstrap/*` lifecycle logic | Consumed by the `service` binary only; future binaries should own their own bootstrap flow |

<!-- profile:grpc:start -->
For native gRPC, `api/proto/**/*.proto` is the source of truth;
`internal/gen/proto/` and bootstrap-owned service registration consume it.
<!-- profile:grpc:end -->

Two repository-wide rules matter most:
1. Generated code is derived code. Edit the contract or generation inputs first, then regenerate.
2. Concrete adapter wiring belongs in the composition root, not inside `internal/<feature>`.

## Stable Dependency Direction

The default dependency direction is inward toward business behavior and outward only at the composition root:

```text
cmd/service/main.go
  -> cmd/service/internal/bootstrap
     -> internal/config
     -> internal/<feature>/*
     -> internal/infra/*

internal/infra/http
  -> internal/openapi
  -> internal/<feature>/*

internal/infra/postgres, internal/infra/telemetry
  -> external libraries

internal/config, internal/infra/telemetry
  -> internal/observability/otelconfig
```

<!-- profile:authn-oidc-jwt:start -->
The authentication profile adds one more shared leaf, for the same reason:

```text
internal/config, internal/infra/oidcjwt
  -> internal/authntrust
```
<!-- profile:authn-oidc-jwt:end -->

<!-- profile:grpc:start -->
The optional transport adds:

```text
internal/infra/grpc
  -> internal/failure, internal/reqctx
  -> generated handlers registered only by bootstrap

internal/infra/grpcclient
  -> one shared connection per dependency
  -> generated clients owned by that dependency adapter
```

Feature-owned error identities are classified through `internal/failure`.
HTTP adds its status/title/type catalog in `internal/problem`; gRPC maps the
same neutral code directly to a status and structured details.
<!-- profile:grpc:end -->

Stable direction rules:
- `internal/<feature>` must not depend on `internal/infra/http` or other concrete transport packages.
- Concrete integration packages belong under `internal/infra/*` and may depend on external libraries.
- Shared contracts start beside the consuming `internal/<feature>` package and should move only when real reuse exists.
- `cmd/service/internal/bootstrap` is allowed to know concrete adapters because it is the composition root.
- `internal/observability/otelconfig` is a vocabulary package only; it must not import config, infra adapters, or OpenTelemetry SDK packages.
<!-- profile:authn-oidc-jwt:start -->
- `internal/authntrust` is a rule leaf only; it must not import config, infra adapters, or transport libraries. A rule that only one owner applies stays with that owner rather than moving here.
<!-- profile:authn-oidc-jwt:end -->

## Primary Runtime Flows

### Request/Response Path

1. `cmd/service/internal/bootstrap.Run` builds the config snapshot, lifecycle logging, telemetry, dependency probes, feature services, router, and HTTP server.
2. `internal/infra/http.NewRouter` validates or creates `X-Request-ID` at the HTTP boundary, extracts only W3C Trace Context, and wraps the root router with security headers, framing/body guards, panic recovery, access logging, route labeling, and OpenTelemetry HTTP instrumentation.
3. The application router contains only the generated client API. `/metrics`
   is served by a separate bootstrap-owned diagnostics listener and is never
   mounted on the application listener.
4. `internal/infra/http` maps the request into the generated OpenAPI handler interface and calls the feature package (`internal/<feature>`).
5. The feature package returns domain/use-case results; the HTTP adapter turns them into contract-shaped responses or RFC 9457 problem responses whose stable `code`, type, title, and status come from one closed transport catalog.
6. Transport observability is emitted at the edge: request logs, OpenTelemetry HTTP metrics exported through Prometheus, and OpenTelemetry spans use bounded route templates from the HTTP layer. HTTP metric server identity comes from configured service identity, never the caller-controlled `Host`; the OTel SDK cardinality cap remains explicit; native startup/config metrics share the same private Prometheus registry.

<!-- profile:grpc:start -->
The optional native gRPC listener follows the same feature and lifecycle
direction. Generated handlers call `internal/<feature>` behavior; bootstrap
registers those handlers with `internal/infra/grpc`. Standard gRPC health and
HTTP readiness publish one startup decision, both transports enter drain
together, and diagnostics stop only after both application transports have
drained. See [Native gRPC](grpc.md).
<!-- profile:grpc:end -->

Current runtime note: the shipped client API is intentionally health-only.
New business endpoints must make a security decision before implementation:
public by design, protected by real OpenAPI security plus auth middleware and
401/403 Problem responses, or blocked pending a security spec. Browser CORS
remains fail-closed by default.

Operational exposure note: `/metrics` defaults to the loopback diagnostics
address `127.0.0.1:9090`. A non-loopback diagnostics bind requires a private
scrape network or a separate authenticated design.

Public ingress note: the service binds `http.addr` and does not gate its own
reachability. The deployment platform — firewall, security group, network
policy, or service mesh — owns ingress admission, because it is the only layer
that observes every connection attempt. The startup summary records `app.env`
and `http.addr` so the effective exposure is visible on every boot.

### Startup/Shutdown Path

1. `cmd/service/main.go` delegates to `bootstrap.Run`.
2. Bootstrap parses config flags, creates the signal-aware root context, initializes baseline metrics, and loads the immutable config snapshot through `internal/config`.
3. Bootstrap reconfigures structured logging from the validated config, initializes local OTel-to-Prometheus metrics and optional tracing with one service resource in fail-open mode, validates explicit public-ingress admission, and probes enabled dependencies.
4. The HTTP runtime may begin serving while startup admission is still running, but external `/health/ready` stays not ready until startup admission marks the process ready.
5. Readiness is guarded by startup admission and `internal/health.Service`, which runs enabled dependency probes sequentially under one readiness timeout.
6. `/health/live` remains process-only; external dependency checks, startup admission, and drain belong in readiness.
7. On shutdown, bootstrap marks the service as draining, flips readiness off, waits the configured propagation delay, gracefully shuts down the HTTP server, and flushes telemetry inside the process-grace budget.

The lifecycle baseline is: config and dependency validation happen before accepting traffic, and shutdown is coordinated from the bootstrap layer rather than from handlers or feature services.

### Background / Async Extension Path

<!-- profile:outbox-postgres:start -->
The optional PostgreSQL outbox keeps the request path broker-independent: the
PostgreSQL repository adapter appends a feature-selected event through the same
`pgx.Tx` as its mutation, and `cmd/outbox-relay` later claims and publishes that
durable intent.
The relay owns only a minimal Publisher contract; an initialized service must
register a real adapter, and the process fails closed if none is registered.
When both outbox and NATS are selected, `internal/infra/natsjs` owns that
adapter and `cmd/outbox-relay` supervises the NATS client beside the relay.
Outbox without messaging retains the nil registration and fails before claims.
See [PostgreSQL transactional outbox](postgres-transactional-outbox.md).
<!-- profile:outbox-postgres:end -->

<!-- profile:inbox-postgres:start -->
The optional PostgreSQL inbox is a sibling persistence capability. A concrete
consumer adapter derives a stable consumer identity and logical message ID,
then calls `postgresinbox.Claim` in the same caller-owned `pgx.Tx` as one
feature effect. A committed duplicate returns success without reapplying the
effect; a rollback or database error remains a handler error. Claims do not
expire, do not serialize generic consumer work, and do not cover external
effects. See [PostgreSQL idempotent inbox](postgres-idempotent-inbox.md).
<!-- profile:inbox-postgres:end -->

<!-- profile:messaging-nats-jetstream:start -->
The optional NATS JetStream profile ships a separate `cmd/worker` composition
root and concrete `internal/infra/natsjs` producer/consumer owner. The service
process remains producer-only; the worker fails before connecting until a
binary-local handler adapter is registered to invoke duplicate-safe feature
behavior. The same package supplies the selected outbox adapter when the outbox
profile is also retained; it forwards the stored W3C creation context without
adding generic consumer ordering. See [Durable messaging](./durable-messaging.md).
<!-- profile:messaging-nats-jetstream:end -->

When a task introduces async work, keep the extension path stable:
1. Put business behavior in `internal/<feature>`.
2. Put queue, scheduler, database, or external-system mechanics in `internal/infra/<integration>`.
3. Own lifecycle, config loading, telemetry, and graceful shutdown in a composition root under `cmd/<binary>/` or another explicit bootstrap entrypoint.

Preferred rule: if the workload has a distinct lifecycle or scaling model, add a new binary instead of hiding durable background loops inside HTTP handlers.

## System Neighbors

The runtime flows above stop at this service's edges. This section records what is on the other side of those edges, and it is the inventory that boundary-crossing diagnosis and design consume before reading any code. This record is repository-owned: [Template Sync](./template-sync.md) never mirrors it, so each derived service maintains its own.

The template itself ships no neighbor. A derived service replaces the placeholder row with one row per system it exchanges work with — inbound callers and clients, outbound providers, brokers, jobs, and managed dependencies.

| Neighbor | Role on the failing path | Canonical contract source | Local checkout or clone | Runtime evidence | Owner |
| --- | --- | --- | --- | --- | --- |
| _(none in the template)_ | inbound caller, outbound provider, broker, job, or managed dependency | its repository, generated contract, published spec, or live contract endpoint | path or clone URL | the log/trace query, dashboard, or command that reads it | accountable team or person |

Rules that stay stable:

- A neighbor belongs here as soon as this service calls it, is called by it, or shares durable state with it — not only when a task changes it.
- Record where its current contract actually lives, not a local copy or generated client standing in for it.
- Record the concrete way to read its runtime evidence, plus the correlation field that joins it to this service's `X-Request-ID` and W3C Trace Context. Without that join, cross-service evidence cannot be gathered for one unit of work.
- A neighbor discovered during diagnosis is added here as part of closing that diagnosis.
- Record access paths and query shapes only. Credentials, tokens, and customer data never belong in this file.

## Extension Seams

Use these seams when extending the repository:

- New HTTP capability: first consume the approved `spec.md` behavior/contract delta plus any needed system/integration contract decisions; then update `api/openapi/service.yaml`, regenerate `internal/openapi`, add use-case logic in `internal/<feature>`, and wire handlers/routes in `internal/infra/http`. Do not use OpenAPI edits, generated code, handlers, or tests to invent resource, status, error, retry, async, freshness, or compatibility semantics.
<!-- profile:grpc:start -->
- New gRPC capability: define the accepted RPC and compatibility behavior first; add an Edition 2023 Opaque schema under `api/proto`, regenerate `internal/gen/proto`, implement a thin feature-facing adapter, and register it in `cmd/service/internal/bootstrap/startup_grpc.go`. Generated handlers, raw statuses, streaming mechanics, and tests do not own domain semantics, deadlines, retry safety, authentication, or stream-duration policy.
- New outbound gRPC dependency: construct one bootstrap-owned `grpcclient` connection, explicitly select `PropagationNone`, `PropagationTraceContext`, or `PropagationTrustedService` at that neighbor's trust boundary, pass it to the provider-generated client, and close it during shutdown and partial-startup cleanup. The zero policy emits no remote correlation; trusted service adds only a valid context request ID to W3C Trace Context; baggage is always omitted. Generated standard health and provider clients consume the same `grpc.ClientConnInterface` seam. Also select the address-selection policy for that neighbor: round robin is the zero value, reaches every resolved address, and follows standard health for the empty service name; first-address selection is the alternative when one subchannel per backend is not wanted. Disable `HealthCheck` only when the named dependency does not publish that whole-process status. When it protects `Health/Watch`, pass a provider-owned connection credential through `Options.PerRPCCredentials`; per-call credentials do not authenticate the health stream. Idle keepalive stays off unless the dependency has a concrete intermediary timeout, in which case set both positive keepalive fields to peer-compatible values. Environment proxies and resolver-provided service configs are deliberately disabled — the client still supplies its own default service config, which carries address selection and health without adding application retries — and a dependency that requires a proxy or resolver-provided config needs its own design instead of weakening the shared connection.
<!-- profile:grpc:end -->
<!-- profile:authn-oidc-jwt:start -->
- New rule about who may call this service — `internal/infra/oidcjwt` has no registration point, so an extra claim requirement, a different audience rule, or propagating more than the verified issuer, opaque subject, and OAuth client ID is an edit inside that package — `parseAccessTokenClaims` for claim rules, and the `reqctx.Principal` construction in `parseToken` for what reaches handlers. A new configured value is the one that leaves the package: it needs a field on `config.AuthnConfig` with its koanf key and validation in `internal/config` as well as the value in `Policy`/`NewPolicy`, because both owners must refuse a bad value — the loader so the process stops, the verifier so a policy built any other way still fails closed. `internal/config` cannot import runtime adapters, so a rule the two share belongs in `internal/authntrust` and is called from each side; a rule only the verifier applies stays in the verifier. Its package documentation names each site. Treat such an edit as a deliberate fork of the template's copy: a later template sync reports it as a conflict, which is what keeps a change to who may call this service visible in review. Anything past identity — roles, tenant policy, per-operation permission — is feature-owned and does not belong in this package.
<!-- profile:authn-oidc-jwt:end -->
- New persistence flow: add one canonical transactional Goose file under `migrations`, add SQLC query sources under `internal/infra/postgres/queries`, regenerate `internal/infra/postgres/sqlcgen`, add a hand-written Postgres repository that maps generated rows into feature-facing types, add a feature-owned port only if needed, then wire the concrete adapter in `cmd/service/internal/bootstrap`.
<!-- profile:inbox-postgres:start -->
- New idempotent consumer effect: keep `postgresinbox.Claim` and the tx-bound feature repository in the concrete consumer adapter; use the logical message ID plus a stable consumer identity, and return success on a committed duplicate. External effects require their own downstream idempotency or transactional outbox.
<!-- profile:inbox-postgres:end -->
- New integration adapter: add it under `internal/infra/<integration>`; add a feature-owned contract only if `internal/<feature>` needs inversion over the concrete adapter; wire concrete dependencies in `cmd/service/internal/bootstrap`. For ordinary provider-specific clients, start with `net/http`. When the repository was initialized with `OUTBOUND_HTTP=bounded`, reuse `internal/infra/httpclient` for fixed-authority transport safety and explicitly select `PropagationNone`, `PropagationTraceContext`, or `PropagationTrustedService` per dependency; zero emits no remote correlation. Keep authentication, operation budgets, retry eligibility, provider errors, and generated clients in the provider adapter. Credentials belong in headers; query-string authentication requires a separate telemetry-disclosure design. When the adapter calls another microservice, first verify the provider's current contract from its repository, generated contract, published spec, or live contract endpoint, then record the source used in the owning spec/design/tasks proof. Before enabling a runtime dependency, define config keys and secret-source policy, platform egress policy, criticality, retry and timeout budget, readiness participation, cleanup on partial initialization, low-cardinality metrics labels, and bootstrap tests.
<!-- profile:messaging-nats-jetstream:start -->
- New durable event flow: keep payload/schema semantics in the feature, compose the concrete `natsjs.Producer` or one duplicate-safe worker handler at bootstrap, and use the existing message identity and ACK boundary. Add outbox or inbox persistence only through its separate accepted workflow; do not hide it inside the transport package.
<!-- profile:messaging-nats-jetstream:end -->
- New outbound target: fixed targets must declare source, timeout, redirect policy, and DNS/IP-class behavior before bootstrap wiring; the deployment owns network-level egress enforcement. Dynamic or user-controlled URLs require a separate security design.
- New durable schema behavior: evolve `migrations/` first, then keep adapter or generated access code derived from that schema.
- New executable surface: add `cmd/<binary>/main.go` with its own bootstrap path and reuse feature/infra packages instead of duplicating logic.
<!-- profile:grpc:start -->
- New non-HTTP contract surface: `api/proto/` is the source-of-truth location for protobuf contracts; `make proto-check` owns format, documentation, lint, and drift, and `BASE_REF=<ref> make proto-breaking` owns compatibility.
<!-- profile:grpc:end -->

## Related Repository Docs

Use these docs instead of duplicating their detail here:

- Structure and placement rules: [Project Structure & Module Organization](./project-structure-and-module-organization.md)
- Config sources, precedence, and secret policy: [Configuration Source Policy](./configuration-source-policy.md)
- Local commands, validation commands, and generation flows: [Build, Test, and Development Commands](./build-test-and-development-commands.md)
- CI gates and production-readiness expectations: [CI/CD Production-Ready Checklist](./ci-cd-production-ready.md)
- Task-local workflow and artifact sequencing: [Spec-First Workflow](./spec-first-workflow.md)
