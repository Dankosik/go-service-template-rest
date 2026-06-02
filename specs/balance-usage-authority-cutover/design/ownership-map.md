# Ownership Map

Status: review-ready
Date: 2026-06-02

## Source Of Truth

| Concern | Owner | Non-owners |
| --- | --- | --- |
| Customer USD ledger, balance, reserved exposure, operation outcomes, idempotency, reconciliation | `billing-service` Postgres | `gonka-proxy`, Redis, memory, Redpanda, ClickHouse |
| Parent microlease reserve and active exposure | `billing-service` Postgres and app logic | Proxy local balance rows, Redis, memory |
| Per-request migrated paid authorization proof | `gonka-proxy` durable child debit and terminal obligation against billing-issued microlease | Proxy memory cache, Redis token, request ID |
| Public OpenAI-compatible `/v1*` behavior | `gonka-proxy` | `billing-service` |
| Pricing snapshot truth | `pricing-service` | `billing-service`, `gonka-proxy` |
| Pricing lineage stored with money operations | `billing-service` stores immutable evidence supplied by caller/pricing | Pricing-service as money ledger, proxy as pricing authority |
| API-key lifecycle and policy object ownership | `api-key-service` | `billing-service` |
| Final spend/account/usage authority for migrated paid requests | Billing microlease plus proxy durable child debit lineage | API-key-service `spend_limit_check_required`, proxy local balance, Redis, memory |
| Payment/top-up/provider evidence | Out of scope; later payment/top-up specs | This cutover |
| Worker lifecycle and durable event recovery | `cmd/billing-worker` plus `internal/app/microleaseworker` | HTTP handlers |
| Redpanda event transport | `internal/infra/redpanda` adapters and billing worker | App packages as Kafka clients, Postgres repositories as producers |

## Dependency Direction

Target direction:

```text
api/openapi/service.yaml
  -> internal/api generated bindings
     -> internal/infra/http strict adapter
        -> internal/app/billingauthority
           -> app-owned ports
              <- internal/infra/postgres concrete repositories

cmd/service/internal/bootstrap
  -> config snapshot
  -> postgres pool and probes
  -> app services
  -> http router

api/proto/events/v1
  -> internal/api/events/v1 generated DTOs
     -> internal/infra/redpanda adapters
        -> internal/app/billingauthority or app-owned event ports
           -> internal/infra/postgres concrete repositories

cmd/billing-worker/internal/bootstrap
  -> config snapshot
  -> postgres and Redpanda clients/probes
  -> microleaseworker tasks
```

Rules:

- `internal/app/*` must not import `internal/infra/http`,
  `internal/infra/postgres`, `internal/infra/redpanda`, or config packages.
- `internal/infra/http` may import generated `internal/api` and app packages.
- `internal/infra/postgres` may import generated SQLC and app/domain types.
- `internal/infra/redpanda` may import generated event DTOs and app-owned event
  command types.
- `cmd/service` and `cmd/billing-worker` composition roots may know concrete
  adapters.
- Generated code is derived code; OpenAPI/proto/SQL query sources are edited
  first.

## Generated Contract Authority

- REST contract authority: `api/openapi/service.yaml`.
- REST generated server/client types: `internal/api/openapi.gen.go`.
- Event contract authority: `api/proto/events/v1/*.proto`.
- Event generated DTOs: `internal/api/events/v1`.
- SQL query source: `internal/infra/postgres/queries/*.sql`.
- SQL generated code: `internal/infra/postgres/sqlcgen/*`.

Design-only contract notes under `design/contracts/` do not override runtime
source files. During implementation, generated drift must be resolved through
repository-owned generation targets.

## Auth Ownership

Billing-service owns inbound service authentication and route-scope
authorization:

- JWT/JWKS verifier and principal extraction: `internal/infra/http`.
- Auth config validation: `internal/config`.
- Route scope mapping: OpenAPI `x-route-scopes` plus HTTP middleware constants.
- Account binding and represented-user consistency: app layer.

Gonka-proxy owns minting or obtaining scoped service JWTs for its internal calls
to billing-service. `BILLING_SERVICE_AUTH_KEY` bearer-key production auth is
not accepted for migrated money authority.

## Runtime Ownership

HTTP service owns:

- request validation;
- synchronous account resolve, balance read, usage command, operation readback,
  reconciliation/admin readback responses;
- fail-closed `503` when runtime is disabled or not ready;
- no durable retry loops.

Billing worker owns:

- terminal event consumption;
- checkpoint and close event consumption;
- inbox retry and quarantine/redrive;
- outbox relay;
- stale usage/microlease/child-debit reconciliation;
- admission-control renewal and closure;
- readiness for worker-dependent migrated cohorts.

Postgres owns:

- idempotency conflict detection;
- row locks and single-writer balance invariants;
- immutable ledger effects;
- stored outcomes;
- durable inbox/outbox state.

Redpanda owns transport only. It does not authorize money or replace inbox,
idempotency, or ledger state.

## Proxy Ownership During Cutover

Gonka-proxy remains the public facade and execution coordinator. For migrated
cohorts it must own:

- public auth and API-key/identity context collection;
- pricing evidence collection from pricing authority;
- billing account resolve/balance/readback client calls;
- durable microlease grant persistence;
- durable child debit and terminal obligation before external execution;
- terminal/checkpoint/close event publication;
- OpenAI-compatible error mapping from billing fail-closed outcomes.

Gonka-proxy must not own migrated customer-money balance mutation. Local
`User.balanceNgonka`, local in-memory reservations, local
`BalanceTransaction` writes, and local usage logs are legacy or analytics
surfaces only after account migration.

## Explicit Non-Owners

- Redis is not customer-money authority, idempotency authority, or visible
  balance authority.
- Process memory is not spend authority.
- Redpanda is not money mutation authority.
- Request IDs are not settlement keys.
- Billing-service does not own pricing catalogs, model routing, devshard
  execution, transfer-agent signing, API-key lifecycle, public user/session
  authentication, payment-provider sessions, top-ups, payment evidence, or
  refunds in this cutover.
