# Ownership Map

Status: review-ready
Consumes: `overview.md` and `../spec.md`

## Source-Of-Truth Ownership

| Thing | Owner | Non-owners |
| --- | --- | --- |
| Customer USD ledger, settled/reserved/available balances, active exposure | billing-service PostgreSQL | proxy, Redis, memory, Redpanda, ClickHouse, pricing-service, API-key-service |
| Microlease issuance/replenishment outcome | billing-service PostgreSQL with durable idempotency and stored outcome | proxy durable rows, Redpanda, Redis, memory |
| Proxy child debit authorization before execution | `gonka-proxy` durable allocator under billing-issued parent authority | billing request path, memory, Redis, Redpanda |
| Terminal settlement and customer charge | billing-service PostgreSQL after durable event inbox validation | proxy, Redpanda, analytics stores |
| Pricing truth and snapshot evidence | pricing-service | billing-service, proxy, API-key-service |
| API-key lifecycle and policy attribution | api-key-service | billing-service, proxy |
| Final spend/account/usage check for paid admission | billing-service and proxy using billing authority plus API-key attribution | api-key-service alone |
| Payment-provider lifecycle and raw PSP evidence | payments-service | billing-service microlease scope |
| Event transport and replay | Redpanda plus producer/consumer adapters | Redpanda as money authority |
| Operational projections and analytics | ClickHouse or future stores if introduced | prepaid admission or ledger authority |

## Authority Classes

- `authoritative_money`: billing PostgreSQL tables, balance locks, ledger,
  idempotency, operation outcomes, inbox/outbox, reconciliation records.
- `durable_non_authoritative_proof`: proxy grants, child debits, terminal
  obligations, local outbox, checkpoint/close facts.
- `transport`: Redpanda topics and producer/consumer offsets.
- `cache_limiter_projection`: process memory and any future Redis use.
- `analytics`: ClickHouse or aggregate stores.

Every future design surface must map to one of these classes. Any surface that
would mint spend authority outside `authoritative_money` reopens specification.

## Dependency Direction

Billing-service keeps the repository baseline:

```text
cmd/service or cmd/billing-worker
  -> cmd/*/internal/bootstrap
     -> internal/config
     -> internal/app/microlease
     -> internal/app/reconciliation
     -> internal/infra/postgres
     -> internal/infra/http or internal/infra/redpanda
     -> internal/infra/telemetry

internal/infra/http
  -> internal/api
  -> internal/app/microlease

internal/infra/redpanda
  -> internal/api/events/v1
  -> internal/app/microlease
  -> internal/app/reconciliation

internal/app/*
  -> app-owned ports and small domain types only
```

`internal/app` must not depend on HTTP, Redpanda, Redis, SQLC generated types,
or bootstrap packages. Concrete adapters are wired only in composition roots.

## Generated-Code Authority

- HTTP runtime authority: `api/openapi/service.yaml`.
- Event runtime authority after introduction: `api/proto/events/v1/*.proto`.
- SQL runtime authority: `env/migrations/*.sql` plus
  `internal/infra/postgres/queries/*.sql`.
- Generated outputs are derived and must not become the place where decisions
  are made.

## Adapter Responsibility

HTTP adapter:

- validates protected service auth and route scopes;
- maps bodies and headers into app commands;
- keeps account, lease, child, and operation identifiers in body fields to avoid
  raw path logging exposure;
- maps app outcomes to bounded Problem responses and result codes;
- emits route-level low-cardinality telemetry.

Postgres adapter:

- owns transaction mechanics, row locks, unique constraints, SQLC calls, inbox
  and outbox persistence, and immutable ledger writes;
- does not call external services while holding a DB transaction;
- maps generated rows to app-owned types.

Redpanda adapter:

- owns producer/consumer sessions, offset discipline, bounded retry/backoff,
  idempotent producer configuration where available, authenticity checks, and
  event DTO mapping;
- never treats broker exactly-once semantics as covering billing DB effects.

Telemetry adapter:

- emits low-cardinality metrics for outcome class, failure class, strict reason,
  lag bucket, stale age bucket, and reconciliation reason;
- excludes account IDs, raw request IDs, raw payloads, tokens, secrets, and
  dynamic proof URLs from metric labels.

## Explicit Non-Owners

- Process memory does not own spend authority, visible balance, child debit
  identity, idempotency, terminal outcome, checkpoint proof, or release proof.
- Redis does not own spend authority, visible balance, durable idempotency,
  child debit identity, terminal outcome, or release proof. Redis is absent from
  the first target path.
- Redpanda does not own reserve authority, money mutation, final ledger truth,
  or idempotency outcome.
- Proxy durable rows do not own visible balance or customer-money ledger truth.
- Billing-service does not own pricing publication, API-key lifecycle, identity
  lifecycle, payment-provider state, public gateway routes, model routing, or
  transfer-agent signing.

## Review-Relevant Trade-Offs

Selected: durable child debit before execution remains mandatory.

Rejected alternative: memory or Redis token spend followed by async checkpoint.

Reason:

- the rejected option is the only way to remove the per-request durable proxy
  allocation write, but it creates unbacked spend exposure without a product
  budget;
- the approved spec sets that budget to `0 USD`;
- durable proxy allocation preserves auditability and lets billing cap charges
  by child and parent authority.

Selected: Redis absent from first target.

Rejected alternative: Redis shared bucket in the request path.

Reason:

- Redis cannot replace durable child debit;
- its failure modes would need extra proof while adding no money authority;
- global backpressure can be handled by billing admission controls and proxy
  local backlog gates for the first target.
