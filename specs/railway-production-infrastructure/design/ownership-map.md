# Railway Full Production Infrastructure Ownership Map

Status: review-ready
Date: 2026-06-02

## Source-Of-Truth Owners

| Surface | Owner | Consumed By |
| --- | --- | --- |
| Accepted scope, non-goals, invariants, and proof obligations | `spec.md` | All design, review, planning, and implementation artifacts. |
| Workflow phase routing and next-session handoff | `workflow-plan.md` and `workflow-plans/technical-design.md` | Technical design review and later planning. |
| App deployment policy | `railway.toml` | Railway app service and deployment proof. |
| Canonical image contents | `build/docker/Dockerfile` | App service, migrator, and worker service. |
| Runtime config schema/validation | `internal/config` | App bootstrap, worker bootstrap, tests, and variable proof. |
| Non-secret default config | `env/config/default.yaml` | Runtime defaults and default-closed posture. |
| Variable key examples | `env/.env.example` | Future Railway key-name proof; not value authority. |
| Network ingress/egress operator policy | `NETWORK_*` process env channel | App/worker bootstrap network policy. |
| REST contract and route scopes | `api/openapi/service.yaml` | Generated `internal/api`, HTTP middleware, proxy scope handoff. |
| Generated HTTP bindings | `internal/api` | Runtime adapters; derived from OpenAPI, not authority. |
| HTTP route/auth/runtime edge | `internal/infra/http` | App service and protected route handling. |
| Billing money and microlease business behavior | `internal/app/*` and migration-backed Postgres repositories | HTTP handlers and worker event store. |
| Database schema | `env/migrations` | Postgres runtime, SQLC, migration proof, restore proof. |
| SQLC generated query code | `internal/infra/postgres/sqlcgen` | Postgres repositories; derived from migrations/query sources. |
| Postgres service identity and backups/PITR state | Railway production environment | App/worker variables and data validation. |
| Broker service identity/topic policy/lag | Railway production environment and broker admin/read-back tooling | Worker consumers/producers and validation. |
| Worker lifecycle | `cmd/billing-worker/internal/bootstrap` and `internal/app/microleaseworker` | Railway `billing-worker` service. |
| JWT verification | billing-service `internal/infra/http/service_auth.go` | Protected internal routes. |
| JWT signing, JWKS publication, and key rotation | `gonka-proxy` provider contract | Billing-service verifier and paid-readiness gate. |
| Child-debit lineage before external execution | `gonka-proxy` provider contract and durable proxy storage | Billing-service event consumers and reconciliation. |
| Final implementation tasking | Future reviewed `tasks.md` | Implementation only after technical design review and planning. |

## Dependency Direction

Repository dependency direction remains aligned with `docs/repo-architecture.md`:

```text
cmd/service
  -> cmd/service/internal/bootstrap
     -> internal/config
     -> internal/infra/http
     -> internal/infra/postgres
     -> internal/infra/redpanda
     -> internal/app/*

cmd/billing-worker
  -> cmd/billing-worker/internal/bootstrap
     -> internal/config
     -> internal/infra/postgres
     -> internal/infra/redpanda
     -> internal/app/billingauthority
     -> internal/app/microleaseworker

internal/infra/http
  -> internal/api
  -> internal/app/*

internal/app/*
  -> app-owned contracts and stable domain types
  -> no HTTP, Railway, or concrete broker/service-auth dependency
```

Concrete Railway, Postgres, Kafka, and service-auth wiring belongs in process
composition roots or infra adapters, not inside business use cases.

## Cross-Service Ownership

Billing-service owns:

- customer-money authority for migrated scopes through its dedicated Postgres;
- protected route verification and scope enforcement;
- accepted topic names, consumer group, producer identity validation, quarantine,
  inbox/outbox handling, and reconciliation once events arrive;
- fail-closed admission and authority mode behavior;
- secret-free evidence for its own Railway services.

`gonka-proxy` owns:

- signing private-key custody;
- JWKS publication and rotation;
- service JWT issuance with subject `svc:gonka-proxy`;
- private URL configuration for billing-service calls;
- route-scope selection for each billing call;
- durable child-debit allocation before external execution;
- terminal/checkpoint/close event production with producer identity
  `gonka-proxy`;
- proof that legacy shared-key fallback and proxy-local money writes are absent
  for migrated authority.

Railway owns live infrastructure state:

- project/environment/service IDs;
- source repo/branch/root/config path read-back;
- service domains and public/private network posture;
- variable values;
- deployment status and logs;
- backup/PITR/restore resources;
- volume/storage and broker service runtime state.

Artifacts may record only secret-free IDs, statuses, key names, modes, and
sanitized summaries from Railway.

## Non-Owners And Rejected Authority

These surfaces must not become owners of migrated money authority:

- Redis;
- in-memory proxy reserve state;
- process-local billing-service state;
- proxy-local balance rows;
- legacy `BILLING_SERVICE_AUTH_KEY`;
- direct per-request reserve fallback;
- public `/metrics` proof;
- app-only `tasks.md`;
- dirty or unreviewed sibling `gonka-proxy` changes.

## Generated-Code And Contract Authority

`api/openapi/service.yaml` remains the route/scope authority. Generated Go code
under `internal/api` is derived and must not be hand-edited as the authority.

Broker/event contract design in `design/contracts/service-auth-and-broker.md`
is task-local design context. It does not replace runtime source files, future
event schema authorities, or broker admin read-back.

SQLC output is derived from `env/migrations` and query sources. Production
schema claims require migration proof against the target database, not only
generated-code drift checks.

## Evidence Boundary

Allowed evidence:

- service IDs, deployment IDs, environment/project IDs;
- source repo/branch/root/config-path read-back;
- variable key names and non-secret mode booleans;
- migration version and dirty-state booleans;
- backup/PITR enabled state and restore drill status;
- topic names, retention/partition summaries, consumer group, and lag buckets;
- worker role names and bounded result/reason labels;
- health/readiness status;
- sanitized log summaries.

Forbidden evidence:

- raw secrets, DSNs, bearer tokens, private keys, JWKS documents, JWTs, event
  payloads, request bodies, prompts, completions, SSE chunks, provider payloads,
  OTLP headers, dynamic proof URLs, Railway variable values, or raw customer
  identifiers beyond approved support-safe IDs.

## Planning Boundary

Planning consumes this ownership map after technical design review. It must not
invent new owners for source topology, database authority, topic administration,
worker readiness, JWKS publication, proxy lineage, public ingress, or rollback.
