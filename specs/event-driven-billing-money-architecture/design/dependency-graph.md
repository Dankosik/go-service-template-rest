# Dependency Graph

Status: repaired review-ready technical design for billing-issued spending leases
Consumes: `component-map.md`, `ownership-map.md`

## Target Runtime Graph

```text
gonka-proxy
  -> pricing-service          # immutable USD-compatible pricing snapshot evidence
  -> api-key-service          # credential/policy evidence before paid admission
  -> identity-service         # represented subject/account evidence before paid admission
  -> billing-service HTTP     # protected lease issue/replenish/readback/close
  -> local durable allocator  # child debit allocation and terminal submission
  -> Redpanda                 # terminal and lease checkpoint/close facts

billing-service cmd/service
  -> Postgres                 # lease issuance/readback/close money truth
  -> OpenTelemetry/Prometheus

billing-service cmd/billing-worker
  -> Redpanda                 # terminal/checkpoint consumers and billing fact producer
  -> Postgres                 # inbox/outbox/lease/debit/money truth
  -> OpenTelemetry/Prometheus

Downstream observers
  -> Redpanda billing facts   # derived facts only, no billing DB writes
```

There is no billing-service outbound hot-path call to pricing-service,
identity-service, API-key-service, payments-service, or gonka-proxy inside a
money transaction.

## Package Dependency Graph

```text
cmd/service/internal/bootstrap
  -> internal/config
  -> internal/app/money
  -> internal/infra/http
  -> internal/infra/postgres
  -> internal/infra/telemetry

cmd/billing-worker/internal/bootstrap
  -> internal/config
  -> internal/app/money
  -> internal/infra/postgres
  -> internal/infra/redpanda
  -> internal/infra/telemetry

internal/infra/http
  -> internal/api
  -> internal/app/money
  -> internal/infra/telemetry

internal/infra/postgres
  -> internal/infra/postgres/sqlcgen
  -> internal/app/money app-facing contracts

internal/infra/redpanda
  -> internal/api/events/v1 generated event DTOs
  -> internal/app/money event-facing contracts

internal/app/money
  -> internal/domain/money
```

Forbidden dependencies:

- `internal/app/money -> internal/infra/http`
- `internal/app/money -> internal/infra/postgres`
- `internal/app/money -> internal/infra/redpanda`
- `internal/infra/postgres -> internal/infra/http`
- `internal/infra/redpanda -> internal/infra/http`
- generated `internal/api`, generated protobuf DTOs, or SQLC models as direct
  business contract authority

## New External Dependency Shape

### Protected Billing HTTP

Billing HTTP is a critical dependency for new lease capacity, not for every
paid child request after a valid lease exists.

Required config/design inputs:

- service-principal auth and route scopes;
- deadline and idempotency policy;
- account binding and represented subject evidence;
- request/response size limits;
- retry/readback policy for ambiguous lease command outcomes;
- startup/readiness fail-closed behavior for missing protected-route auth.

Proxy behavior:

- if lease issuance/replenishment is unavailable, proxy may spend existing
  active valid lease capacity until cap/cutoff/local health policy allows;
- if no valid capacity exists, paid admission fails closed;
- proxy must not fall back to direct per-request reserve or cached balance.

### Redpanda

Redpanda is a critical dependency for terminal settlement, checkpoint/close
replay, and billing facts after lease issuance. It is not a source of spendable
capacity.

Required config/design inputs:

- brokers, TLS/SASL/mTLS settings, client ID, topic names, consumer groups, and
  producer identity;
- per-topic allowed producer authority;
- per-topic partition key and retention minimums;
- consumer max in-flight per partition;
- offset commit policy after durable inbox outcome;
- outbox relay retry/backoff;
- critical lag thresholds and Postgres admission-control lease settings.

Readiness:

- `cmd/billing-worker` readiness depends on Redpanda and Postgres for event
  loops.
- `cmd/service` lease issuance readiness depends on Postgres and protected
  route auth config, not Redpanda directly.
- New lease issuance/replenishment is backpressured by worker lag through
  Postgres `billing_admission_controls`.
- Config supplies thresholds, lease duration, topic names, and startup
  defaults. It is not the live paid-admission control surface.

### Admission Control

Selected control surface: `billing_admission_controls` in billing-service
Postgres.

Runtime edges:

```text
cmd/billing-worker
  -> Redpanda lag observations
  -> Postgres inbox/outbox/reconciliation/stale-lease/debit observations
  -> Postgres billing_admission_controls writes

cmd/service protected lease issue/replenish
  -> Postgres billing_admission_controls reads
  -> Postgres lease issuance transaction
```

Rules:

- `cmd/service` does not call Redpanda or the worker to decide lease issuance.
- Missing, expired, stale, malformed, `throttle`, or `fail_closed` control
  state rejects new lease capacity before any reserved exposure is created.
- The global `paid_usage_admission` row is required. Account/cohort rows are
  optional and can only make the matching scope stricter than global state.
- Worker-authored `open` is a short lease. A stopped worker, broken lag
  observer, failed checkpoint processor, or failed stale-debit scanner causes
  the control lease to expire and new capacity to fail closed.
- Operator/admin overrides write the same table through protected repair
  authority and must be auditable.
- Operation, lease, and balance readback can remain available while new paid
  capacity is closed.

### Postgres

Postgres remains the shared correctness dependency across HTTP and worker
binaries.

Risk controls:

- bounded connection pools per binary;
- worker pool budget reserved separately from HTTP lease issuance;
- account-row and lease-row lock timeouts classified separately;
- one short transaction per money command or consumed event;
- no external HTTP inside transactions.

## Cross-Service Contract Dependencies

| Dependency | Source checked | Design use |
| --- | --- | --- |
| Proxy internal-money TypeBox contract | `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts` | Source evidence for field intent and migration bridge only. Not target authority. |
| Proxy shared internal-money headers | `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts` | Source evidence for deadline, caller principal, represented user context, and pricing snapshot basis. |
| Pricing-service current handler | `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go` | Evidence of GNK/USDT selector drift; implementation planning must verify USD-compatible pricing snapshot evidence before lease issuance. |
| API-key-service OpenAPI | `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml` | Confirms `spend_limit_check_required` means caller-side final spend/account/usage checks. |
| Payments-service OpenAPI | `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml` | Confirms no current business payment/top-up OpenAPI authority for this scope. |

## Coupling Risk Controls

- Keep proxy durable lease allocator and terminal submission proxy-owned; billing
  consumes facts and reconciles from billing durable state.
- Keep account-scope validation local to billing from supplied evidence; do not
  add hot-path identity/API-key calls.
- Keep pricing-service as evidence provider, not a transaction participant.
- Keep event contracts versioned and additive; semantic changes to amount
  meaning, identity, finality, producer authenticity, or required proof need new
  versioned topics.
- Keep worker lifecycle separate from HTTP lease issuance so broker outages do
  not take down readback. New capacity can still fail closed through explicit
  Postgres admission-control policy.
