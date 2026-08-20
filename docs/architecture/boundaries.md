# Component Boundaries

Load when package, dependency, source-of-truth, or composition ownership can
change. Current code and package documentation remain the final factual
authority.

| Area | Owns | Does not own |
| --- | --- | --- |
| `cmd/service/main.go` | Thin process entrypoint that delegates to bootstrap. | Business logic, request handling, dependency details. |
| `cmd/service/internal/bootstrap/` | Composition, startup/shutdown, dependency admission, and runtime policy. | Use-case semantics, transport contracts, persistence logic. |
| `internal/config/` | One validated immutable config snapshot. | Feature behavior, dependency wiring, request handling. |
| `api/openapi/service.yaml` | REST contract source of truth. | Runtime logic. |
| `internal/openapi/` | Generated OpenAPI bindings. | Hand-written business logic. |
| `internal/<feature>/` | Use cases, business types, ports, invariants, and domain errors. | HTTP details, drivers, runtime config, process lifecycle. |
| `internal/failure/` | Transport-neutral client-visible failure codes and mapper ordering. | Feature error identities, HTTP envelopes, gRPC statuses, or I/O. |
| `internal/infra/http/` | HTTP server, middleware, mapping, route policy, and transport-edge observability. | Business rules or config loading. |
| `internal/infra/httpclient/` | Optional bounded outbound target validation, transport bounds, correlation policy, and cleanup. | Provider auth, trust, operation budgets, retries, error mapping, or readiness. |
| `internal/infra/postgres/` | PostgreSQL admission, commit-outcome policy, and repository code over `pgxpool`. | Pool lifecycle, migrations, HTTP behavior, config precedence. |
| `internal/infra/postgresmigrate/` | Migration execution for `cmd/migrate`. | Runtime pool ownership or application startup. |
| `internal/infra/telemetry/` | OpenTelemetry SDK setup and Prometheus export. | Feature semantics, startup logging, request routing. |
| `internal/observability/otelconfig/` | Shared OTel vocabulary, defaults, and pure validation. | Config loading or SDK construction. |
| `internal/observability/correlationpolicy/` | Outbound correlation policy shared by bounded HTTP and gRPC clients. | Carrier stripping, transport construction, wire spelling. |
| `internal/observability/logctx/` | The process logger and context correlation fields. | Feature field choice, sinks, or request-ID meaning. |
| `migrations/` | SQL schema source of truth. | Runtime repositories or generated Go. |

<!-- profile:object-storage:start -->
`internal/objectstorage/` owns the provider-neutral port;
`internal/infra/s3/` owns one fixed-authority Amazon S3 or Cloudflare R2
adapter, credential snapshot, provider authority check, bounded reads,
streaming integrity, multipart cleanup, and lifecycle wiring.
<!-- profile:object-storage:end -->

<!-- profile:grpc:start -->
`api/proto/` owns protobuf contracts, `internal/gen/proto/` their generated
messages and interfaces, `internal/infra/grpc/` native server policy/lifecycle,
and `internal/infra/grpcclient/` bounded shared connections and explicit
outbound correlation/routing policy. None owns feature semantics, storage,
authentication, per-operation deadlines, retry eligibility, dependency
criticality, or trust for a concrete neighbor.
<!-- profile:grpc:end -->

<!-- profile:authn-oidc-jwt:start -->
`internal/authntrust/` owns only the pure provider-URL and trusted-peer
predicates shared by config and the OIDC verifier. It owns no configured value,
credential verification, policy object, or authorization decision.
<!-- profile:authn-oidc-jwt:end -->

<!-- profile:outbox-postgres:start -->
`internal/domainevent/` owns typed event identity/version/time/encoding;
`internal/infra/postgresoutbox/` owns transactional append; `cmd/outbox-relay/`
owns River-to-NATS composition, readiness, drain, and cleanup.
<!-- profile:outbox-postgres:end -->

<!-- profile:jobs-postgres:start -->
`cmd/jobs-worker/` and River own default-off typed PostgreSQL job execution;
business job kinds, effect idempotency, operator exposure, and capacity remain
feature/deployment decisions.
<!-- profile:jobs-postgres:end -->

<!-- profile:webhooks-durable:start -->
`internal/outboundtrust/` and `internal/infra/postgreswebhook/` own the public
address predicate and Standard Webhooks job adapter, not generic job state,
subscriber administration, feature transactions, or deployment policy.
<!-- profile:webhooks-durable:end -->

## Dependency Direction

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

Feature packages never depend on concrete transports. Integration packages stay
under `internal/infra`; bootstrap may know them because it is the composition
root. Shared contracts start beside their real consumer and move only for
observed reuse. Rule and logging leaves do not import config, concrete adapters,
transport libraries, or feature packages beyond the explicitly named inputs
above.
