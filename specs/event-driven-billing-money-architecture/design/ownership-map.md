# Ownership Map

Status: repaired review-ready technical design for billing-issued spending leases
Consumes: `../spec.md`, `component-map.md`, `sequence.md`

## Source-Of-Truth Ownership

| Concern | Owner | Non-owners / Constraints |
| --- | --- | --- |
| Customer USD balances, available/reserved/settled atoms, ledger effects, stored outcomes, and reconciliation state | billing-service Postgres | Redpanda, Redis, proxy-local tables, process memory, and broker offsets are not money truth. |
| Spending lease issuance, replenishment, expiry, revocation, close, and release | billing-service app logic and Postgres transaction | Proxy cannot mint lease capacity. Redpanda reserve command/outcome is out of scope. |
| Lease reserved exposure | `spending_leases`, ledger effects, and `account_balances.reserved_usd_atoms` in billing Postgres | Proxy local remaining authority is derived recovery proof only. |
| Proxy local child debit allocation | `gonka-proxy` durable allocator store for one lease owner/generation | Billing does not run proxy execution lifecycle. Proxy allocator cannot exceed billing-issued capacity. |
| Terminal execution fact durability before/after external execution | `gonka-proxy` durable terminal-submission store | Billing does not own raw execution payloads. Proxy outbox is not money truth. |
| Terminal settlement money mutation | billing-service app logic and Postgres transaction triggered by durable inbox processing | Proxy terminal event, Redpanda delivery, and event IDs do not by themselves mutate money. |
| Lease checkpoint/close evidence | Proxy durable allocator emits evidence; billing validates and decides release in Postgres | Proxy close evidence does not release money until billing commits. |
| Event receipt, replay, quarantine, committed-offset retry | `billing_event_inbox` plus billing worker | Redpanda redelivery is not the recovery owner after a committed retry outcome. |
| Emitted billing facts | `billing_event_outbox` plus billing worker | Direct DB write plus direct broker publish dual writes are not allowed. |
| Lease issuance/admission backpressure | `billing_admission_controls` in billing-service Postgres; automatic writer is `cmd/billing-worker`; manual writer is protected operator/admin repair authority | Config owns thresholds/default startup only. `cmd/service` reads Postgres controls and does not call Redpanda or the worker during lease issuance. |
| Pricing truth | pricing-service | Billing verifies immutable USD-compatible snapshot evidence; billing must not own pricing catalog or call pricing inside money transactions. |
| Account scope for current billing | billing-service canonical `accountScopeKey`, day-one `user:<identity_user_id>` | API keys are not balance owners. Organization scope requires future authority contract. |
| API-key lifecycle, scopes, policy configuration | api-key-service | Billing owns money-backed spend aggregates and final money checks, not API-key policy lifecycle. |
| Identity user records | identity-service | Billing stores safe subject/account references only. No hot-path identity call inside money transaction. |
| Protected HTTP contract authority | `api/openapi/service.yaml` | Proxy TypeBox contract is source evidence only. Generated `internal/api` is derived. |
| Runtime Redpanda event schema authority | `api/proto/events/v1/*.proto` | `design/contracts/redpanda-events.md` is design context only. Generated DTOs are derived and adapter-owned, not app business authority. |
| Worker lifecycle and readiness | `cmd/billing-worker` bootstrap | HTTP handlers do not hide durable event/retry/reconciliation loops. |
| Observability labels and route mapping | `internal/infra/http`, `internal/infra/redpanda`, `internal/infra/telemetry` | App logic can return safe failure classes, but adapters own low-cardinality label emission and raw payload suppression. |

## Dependency Direction Rules

- `internal/app/money` owns business transitions and depends only on app-owned
  ports and small stable money/domain types.
- `internal/infra/http` maps OpenAPI transport to app commands. It may depend
  on generated `internal/api` and `internal/app/money`.
- `internal/infra/postgres` implements app repository ports. SQLC-generated
  types are derived and must not leak as app-level contract authority unless
  wrapped by app-facing types.
- `internal/infra/redpanda` implements event consumer/producer mechanics and
  safe envelope decoding/encoding. It must not own lease/debit business rules.
- `cmd/service` and `cmd/billing-worker` are composition roots. They may know
  concrete adapters and config.
- Sibling services are reached only through verified contracts. No
  cross-service database access and no cross-service ACID boundary.

## Operation Ownership

| Operation | Business owner | Transport owner | Persistence owner | Replay identity |
| --- | --- | --- | --- | --- |
| Issue/replenish spending lease | `internal/app/money` | `internal/infra/http` | `internal/infra/postgres` | `(account_id, lease_issue, idempotencyKey)` plus `operationFingerprint`; `spendingLeaseId` for lease lineage. |
| Lease readback | `internal/app/money` | `internal/infra/http` | `internal/infra/postgres` | `spendingLeaseId`, stored outcome identity, or idempotency key plus account/kind. |
| Lease close/cancel | `internal/app/money` | `internal/infra/http` and/or `internal/infra/redpanda` | `internal/infra/postgres` | `spendingLeaseId`, generation/fence, close fingerprint, and idempotency key or event identity. |
| Local child debit allocation | `gonka-proxy` durable allocator | `gonka-proxy` request path | Proxy durable store | `debitAuthorizationId` plus `usageOperationId`, operation fingerprint, and lease generation/fence. |
| Terminal event settlement | `internal/app/money` | `internal/infra/redpanda` | `internal/infra/postgres` | `billing_event_inbox` event identity plus semantic idempotency from `usageOperationId` and `debitAuthorizationId`. |
| Lease checkpoint/close event processing | `internal/app/money` | `internal/infra/redpanda` | `internal/infra/postgres` | `spendingLeaseId`, generation/fence, checkpoint sequence, and event identity. |
| Stale lease/debit reconciliation | `internal/app/money` | `cmd/billing-worker` scheduler/worker | `internal/infra/postgres` | `reconciliation_cases` dedupe by lease/debit/usage lineage and reason. |
| Admission-control update | `internal/app/money` evaluates policy from worker inputs | `cmd/billing-worker` scheduler or protected admin tooling | `billing_admission_controls` | Control key plus scope and generation; expired rows fail closed. |
| Billing fact emission | `internal/app/money` decides source facts | `internal/infra/redpanda` relay | `billing_event_outbox` | `spendingLeaseId`, `debitSettlementId`, `ledgerEntryId`, `settlementEffectId`, `storedOutcomeId`, `reconciliationCaseId`, or `eventInboxId`. |

## Explicit Non-Owners

- Redpanda is not lease issuance authority, not customer-money correctness
  boundary, and not a source of spendable capacity.
- `gonka-proxy` is not the authoritative balance, ledger, lease exposure, or
  account-spend aggregate writer for migrated cohorts.
- Proxy child debit rows are proof and recovery anchors; they are not customer
  balance truth.
- `pricing-service` is not a billing transaction participant; it supplies
  immutable evidence before lease issue/replenishment and child debit
  allocation.
- `api-key-service` does not decide final account spend admission after
  `spend_limit_check_required`.
- `identity-service` does not own billing ledger rows or balance read model.
- `internal/domain` must not become a dumping ground for transport,
  repository, worker, or orchestration logic.
- OpenAPI generated Go code, protobuf generated Go code, and SQLC generated
  code are never hand-edited authorities.

## Compatibility Ownership

If a compatibility bridge is used:

- owner: `gonka-proxy` integration layer plus billing OpenAPI target contract;
- target: billing-service protected lease/debit route semantics;
- forbidden: direct per-request reserve fallback, proxy-local authoritative
  money writes, and pure cached balance admission for migrated cohorts;
- exit criteria: old shared-balance writer disabled, no dual writer, bridge
  disabled/removed by default, and lease-path parity proof recorded in
  `rollout.md`.
