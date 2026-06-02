# Component Map

Status: review-ready
Consumes: `overview.md` and `../spec.md`

## Billing-Service Components

| Surface | Design responsibility | Stable boundary |
| --- | --- | --- |
| `api/openapi/service.yaml` | Add protected internal microlease routes during implementation planning: issue/replenish, readback, close/cancel, and operation readback. | Runtime source of truth for HTTP. This design file is not the contract authority. |
| `internal/api/` | Generated OpenAPI bindings after contract update. | Derived code only; do not hand-edit generated bindings. |
| `api/proto/events/v1/*.proto` | Future source inputs for microlease terminal, checkpoint, close, and billing fact events. | Runtime event authority once introduced; design contracts remain provisional. |
| `internal/api/events/v1` | Generated event DTOs after proto generation. | Adapter-owned DTOs; app logic receives app-owned types after mapping. |
| `internal/app/microlease` | Transport-agnostic use cases for issue/replenish, readback, close verification, terminal settlement command handling, checkpoint application, exposure gates, and reconciliation decisions. | No HTTP, Redpanda, Redis, SQL driver, or process lifecycle imports. |
| `internal/app/reconciliation` | Stale microlease, stale child debit, invalid fence, missing terminal, over-debit, and close-proof repair decisions. | Uses app-owned ports; does not own transport or storage mechanics. |
| `internal/infra/postgres` | Microlease repositories, account balance row locking, idempotency, ledger effects, inbox/outbox, SQLC queries, and migration-backed constraints. | Persistence mechanics only. Business decisions stay in `internal/app`. |
| `internal/infra/http` | Protected route auth mapping, request/response validation, Problem mapping, body identifier placement, route labels, metrics/tracing edge. | No money rules beyond mapping app results. |
| `internal/infra/redpanda` | Producer/consumer adapters for terminal/checkpoint/close/billing facts, offset discipline, producer authenticity, bounded retry, and app type mapping. | Transport/replay mechanics, not money authority. |
| `internal/infra/telemetry` | Low-cardinality metrics and traces for microlease outcomes, lag, strict/fail-closed reasons, and reconciliation. | No raw payloads or high-cardinality account/request labels. |
| `cmd/service/internal/bootstrap` | HTTP service wiring, config validation, dependency probes, protected auth middleware, Postgres repository wiring, and readiness policy. | Composition root only. |
| `cmd/billing-worker` | New explicit worker binary for terminal event consumption, checkpoint/close consumption, inbox retry, outbox relay, stale reconciliation, and admission-control renewal/closure. | Distinct lifecycle and scaling model from HTTP. |
| `internal/config` | Immutable config for microlease caps, TTL/cutoff, lag gates, worker budgets, dependency targets, and fail-closed defaults. | Config validation, not feature behavior. |

## Proxy Components

These are cross-repo obligations for `gonka-proxy`, not files this repository
edits in the technical-design phase.

| Proxy surface | Required role |
| --- | --- |
| Durable microlease grant store | Persist billing-issued grants, account scope, owner, generation/fence, cap, remaining, cutoff, expiry, policy versions, pricing basis, and stored billing outcome. |
| Durable child debit allocator | Single-writer row-lock or compare-and-swap allocation per owner/fence. It atomically reduces local remaining authority, inserts one child debit, and creates a terminal obligation before execution. |
| Memory precheck | Optional cache over durable remaining capacity. It may deny or avoid durable attempts; it cannot authorize execution. |
| Terminal outbox | Durable local terminal facts written before or during request teardown and retried until billing accepts or reconciliation takes over. |
| Checkpoint/close publisher | Emits high-water, cap-sum, terminal coverage, unresolved summary, owner/fence, and fingerprint evidence. |
| Direct reserve fallback controls | Migrated paid cohorts must not call legacy direct reserve/finalize/write-off as an alternate paid path. |

## Redis Decision

Redis is not introduced as a first-target runtime dependency for microlease
admission.

Reason:

- durable child debit remains required before execution, so Redis cannot remove
  the correctness write;
- adding Redis would introduce split-brain, failover, timeout, and rebuild
  proof without creating spend authority;
- billing-owned Postgres admission controls and proxy durable allocator health
  are sufficient for the first production-ready target.

If Redis is added by a later approved design, it must live under
`internal/infra/redis` or the proxy equivalent as limiter/cache/backpressure
only, be rebuildable from durable state, and fail closed or strict when
uncertain. No planning task may treat Redis as customer-money authority from
this packet.

## Stable Non-Touches

The microlease design does not change:

- public OpenAI-compatible `/v1*` routes;
- payment-provider sessions, webhooks, or PSP secrets;
- top-up evidence ingestion and payment reversal flows;
- pricing publication ownership;
- API-key lifecycle or policy configuration ownership;
- identity/session ownership;
- GNK treasury inventory ownership;
- `/metrics`, `/health/live`, `/health/ready`, and `/api/v1/ping` public system
  posture except for additional private operational metrics.

## Generated And Derived Surfaces

Planning must preserve generation authority:

- edit `api/openapi/service.yaml`, then regenerate `internal/api`;
- edit future `api/proto/events/v1/*.proto`, then regenerate event DTOs;
- edit `env/migrations/*.sql` and `internal/infra/postgres/queries/*.sql`, then
  regenerate `internal/infra/postgres/sqlcgen`;
- keep generated artifacts out of `internal/app` authority.

## Boundary Notes

- Billing-service may read pricing snapshot evidence, but `pricing-service`
  owns pricing truth. Current pricing evidence confirms snapshot identity,
  fingerprint, policy version, authoritative decision time, selector/use-class
  context, and contract metadata must be persisted before minting money lineage.
- API-key-service may return `spend_limit_check_required`; billing/proxy must
  perform final spend/account/usage checks without moving API-key policy
  lifecycle into billing.
- Payments-service currently exposes template-level system routes in this
  checkout and remains out of this microlease runtime scope.
