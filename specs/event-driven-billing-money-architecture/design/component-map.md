# Component Map

Status: repaired review-ready technical design for billing-issued spending leases
Consumes: `overview.md`, `../spec.md`, `docs/repo-architecture.md`

## Repository Baseline

The existing repository has a small HTTP service shape:

- `api/openapi/service.yaml` is the REST contract source of truth.
- `internal/api/` is generated from OpenAPI and is derived code.
- `internal/infra/http/` owns routing, middleware, request/response mapping,
  Problem responses, and edge telemetry.
- `internal/app/` owns transport-agnostic use-case behavior.
- `internal/infra/postgres/` owns Postgres pools, migrations, SQLC query
  sources, generated SQLC code, and repository adapters.
- `cmd/service/internal/bootstrap/` owns composition, config, dependency
  probes, readiness, startup, and graceful shutdown.
- The repository currently has no always-on background worker runtime.

## Target Components

| Component | Target responsibility | Change type |
| --- | --- | --- |
| `api/openapi/service.yaml` | Protected internal HTTP contract for lease issue/replenish/readback/close/cancel, account balance readback, operation/debit readback, and reconciliation redrive. Defines real security, scopes, idempotency, request/response schemas, and Problem responses. | Add business money routes and security. |
| `internal/api/` | Generated server/client types from OpenAPI. | Regenerate after contract changes only. |
| `api/proto/events/v1/*.proto` | Runtime Redpanda event schema source of truth for usage terminal, lease checkpoint/close, billing lease/debit facts, reconciliation, and rejection facts. | New protobuf contract inputs. |
| `internal/api/events/v1` | Generated Go event DTOs derived from `api/proto/`; used by Redpanda adapters only. | New generated surface after proto generation is introduced. |
| `internal/infra/http` | Service-principal auth, route-scope checks, request parsing, response/Problem mapping, route-template telemetry, and raw-payload suppression for protected money routes. | Extend transport edge for protected money routes. |
| `internal/app/money` | App-owned command orchestration for lease issuance/replenishment/readback/close, terminal settlement, reconciliation, admission-control evaluation, and outbox creation. Exposes ports for storage, clock/ID generation, and event publishing only where inversion is needed. | New business package. |
| `internal/domain/money` | Small stable money primitives such as USD atom parsing/formatting, amount caps, and value objects shared by app and storage. | Keep small; do not create a generic domain bucket. |
| `internal/infra/postgres` | Repositories for spending leases, lease checkpoints, child debit settlement lineage, event inbox/outbox, admission controls, reconciliation claims, and support/readback queries. Generated SQLC remains derived from migrations/query inputs. | Extend migration/query/repository surfaces. |
| `internal/infra/redpanda` | Kafka-compatible consumer, producer, offset handling, partitioning, ACL/config validation, retry/backoff adapter, and safe envelope codec mechanics. | New integration adapter. |
| `cmd/service` | HTTP lease issue/replenish/readback/close/balance server. Participates in readiness for Postgres and required auth/config. Does not run durable event loops. | Extend bootstrap with protected route dependencies. |
| `cmd/billing-worker` | Separate worker binary for terminal event consumption, lease checkpoint/close consumption, inbox retry, outbox relay, stale lease/debit reconciliation, and admission-control renewal. Owns worker config, readiness, telemetry, shutdown, and Redpanda/Postgres probes. | New executable surface. |
| `gonka-proxy` | Authenticates public callers, obtains API-key/identity and pricing evidence, stores billing-issued lease grants, allocates durable child debits through a single-writer allocator, creates terminal submission obligations before execution, publishes terminal and checkpoint/close facts, and stops writing authoritative balances for migrated cohorts. | Cross-repo implementation constraint, not owned by this repository. |
| `pricing-service` | Pricing source of truth and immutable USD-compatible pricing snapshot evidence. | Consumed contract; billing must not call pricing inside money DB transactions. |
| `api-key-service` | API-key lifecycle, scope, and policy configuration. `spend_limit_check_required` tells proxy to obtain final money checks from billing. | Consumed contract; billing does not own policy lifecycle. |
| `identity-service` | Identity subject authority. | Consumed subject/account evidence; billing does not hot-call identity in lease transaction. |
| Redpanda | Transport and replay for terminal usage facts, lease checkpoint/close facts, billing facts, rejection facts, and authorized repair commands. | New infra dependency; not money truth. |
| PostgreSQL | Customer-money correctness boundary. | Existing authority; extended for spending leases, child debit lineage, inbox/outbox, and admission controls. |

## Internal Package Shape

```text
cmd/service
  -> cmd/service/internal/bootstrap
     -> internal/config
     -> internal/app/money
     -> internal/infra/http
     -> internal/infra/postgres
     -> internal/infra/telemetry

cmd/billing-worker
  -> cmd/billing-worker/internal/bootstrap
     -> internal/config
     -> internal/app/money
     -> internal/infra/postgres
     -> internal/infra/redpanda
     -> internal/infra/telemetry

internal/infra/http
  -> internal/api
  -> internal/app/money

internal/app/money
  -> internal/domain/money
  -> app-owned ports for lease repositories, inbox/outbox repositories,
     admission-control repository, clock, ID generation, and event publication

internal/infra/postgres
  -> internal/infra/postgres/sqlcgen
  -> internal/app/money app-facing repository contracts

internal/infra/redpanda
  -> external Kafka-compatible client
  -> internal/api/events/v1 generated event DTOs
  -> internal/app/money event consumer/producer contracts
```

`internal/app/money` must not import `internal/infra/http`,
`internal/infra/postgres`, or `internal/infra/redpanda`. Bootstrap wires
concrete adapters.

## Generated And Canonical Surfaces

- Runtime HTTP contract authority: `api/openapi/service.yaml`.
- Generated HTTP Go types: `internal/api/openapi.gen.go`, regenerated only from
  OpenAPI.
- Runtime database authority: deterministic migrations under
  `env/migrations/`.
- SQLC query authority: `internal/infra/postgres/queries/*.sql`.
- SQLC generated code: `internal/infra/postgres/sqlcgen/`, regenerated only
  from query/schema inputs.
- Runtime Redpanda event schema authority: protobuf contract inputs under
  `api/proto/events/v1/*.proto`.
- Generated Redpanda event Go DTOs: `internal/api/events/v1`, regenerated only
  from protobuf inputs and used by `internal/infra/redpanda`.
- Event contract design context: `design/contracts/redpanda-events.md` maps
  approved event semantics to protobuf authority, but is not runtime authority.

Planning must add repository-owned proto validation/generation before
implementing event producers or consumers. That flow must lint `.proto` files,
regenerate derived Go DTOs, detect generated drift, and run compatibility checks
for additive evolution. Changes to event identity, amount meaning, finality,
required proof, producer authenticity, or replay semantics require a new
versioned proto package/topic instead of an in-place breaking change.

## Proxy Durable Lease Allocator

`gonka-proxy` owns the hot-path lease allocator store. The target proxy
component must:

- persist each lease grant with `spendingLeaseId`, generation/fence,
  `proxyLeaseOwnerId`, account scope, issued amount, local remaining authority,
  expiry/cutoff, billing stored outcome reference, and pricing/policy
  constraints;
- allocate one child debit per paid request with `debitAuthorizationId`,
  `usageOperationId`, child cap, operation fingerprint, pricing snapshot
  identity/fingerprint, sequence or allocation position, terminal deadline, and
  safe caller lineage;
- atomically decrement local remaining lease authority and create the terminal
  submission obligation before external execution;
- stop allocating from expired, revoked, stale-fenced, exhausted, or locally
  unhealthy leases;
- publish terminal facts and lease checkpoint/close facts from durable rows and
  retry until broker acknowledgement;
- never use this store as customer balance truth or as a way to mint capacity
  beyond billing-issued leases.

If the proxy durable allocator or terminal-submission store is unavailable, paid
admission fails closed before external execution.

## Stable Non-Touches

The design intentionally does not change:

- public OpenAI-compatible `/v1*` route behavior in `gonka-proxy`;
- pricing catalog ownership or model routing;
- API-key lifecycle or policy configuration ownership;
- identity user records or organization authority;
- payment-provider sessions, payment webhooks, customer top-up runtime flows,
  or payment evidence ingestion;
- GNK treasury inventory as customer balance truth;
- browser CORS defaults or public business ingress posture;
- existing `/metrics` direct root-router exception, except rollout must keep it
  private or explicitly protected as already required by repo architecture.

## Compatibility Bridge Placement

A compatibility bridge is not the target. If rollout proves one-step cutover
unsafe, the bridge belongs at the proxy-to-billing integration boundary:

- target billing OpenAPI remains the contract owner;
- bridge code adapts old proxy call sites to target lease/debit semantics;
- the bridge may translate old reserve/finalize/write-off call sites into
  lease issue/replenish, local child debit, terminal fact, and readback
  operations;
- no bridge may keep direct per-request billing reserve or proxy-local balance
  writes as a target path for migrated cohorts;
- bridge exit criteria and removal proof live in `rollout.md`.
