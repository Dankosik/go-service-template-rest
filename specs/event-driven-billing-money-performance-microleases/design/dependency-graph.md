# Dependency Graph

Status: review-ready
Trigger: new app/infra packages, generated-code flow, worker lifecycle,
cross-repo proxy allocator dependency, and avoiding accidental Redis/money
coupling.

## Runtime Graph

```text
gonka-proxy
  -> api-key-service              (credential attribution; no final spend authority)
  -> pricing-service              (snapshot evidence before lineage)
  -> billing-service protected HTTP
       -> billing PostgreSQL      (microlease issue/replenish/readback/close)
  -> proxy durable allocator DB   (child debit before execution)
  -> external paid execution
  -> proxy terminal outbox
       -> Redpanda
          -> cmd/billing-worker
             -> billing PostgreSQL (inbox, ledger, microlease settlement)
             -> billing outbox

cmd/service
  -> internal/infra/http
     -> internal/api
     -> internal/app/microlease
        -> app-owned ports
           -> internal/infra/postgres

cmd/billing-worker
  -> internal/infra/redpanda
  -> internal/app/microlease
  -> internal/app/reconciliation
  -> internal/infra/postgres
  -> internal/infra/telemetry
```

Redis is intentionally absent from the graph. If a later design adds it, it
must hang from the proxy or billing infra layer as a limiter/cache dependency
that cannot be required to prove money correctness.

## Package Direction

Allowed:

- transport adapters depend on generated contract packages and app services;
- app services depend on app-owned ports and small domain types;
- Postgres and Redpanda adapters implement app-owned ports;
- bootstrap packages wire concrete adapters;
- generated code depends only on generation inputs and runtime libraries.

Not allowed:

- `internal/app` importing HTTP, Redpanda, Redis, SQLC generated rows, or
  bootstrap packages;
- `internal/infra/postgres` calling pricing-service, proxy, Redpanda, or HTTP;
- billing-service calling `gonka-proxy` while holding a DB transaction;
- Redpanda consumer committing offsets before inbox/outcome persistence;
- Redis or memory dependencies in app-level money authority decisions.

## Cross-Service Contract Edges

| Edge | Contract source checked | Design use |
| --- | --- | --- |
| proxy to billing historical internal money | `gonka-proxy/src/contracts/internal-money/billing/v1/index.ts` | Confirms current reserve/finalize/write-off TypeBox route IDs and pricing snapshot basis are compatibility inputs only. New billing OpenAPI owns target microlease contracts. |
| shared money headers/pricing basis | `gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts` | Confirms version/deadline headers, caller context, USD decimal strings, and pricing snapshot fields to preserve in new protected HTTP. |
| billing/pricing | `pricing-service/README.md` and pricing HTTP handlers | Confirms pricing-service owns pricing truth and lineage must persist `pricingSnapshotId`, `snapshotFingerprint`, policy version, decision time, selector/use-class context, and contract metadata. |
| proxy/API-key | `api-key-service/api/openapi/service.yaml` | Confirms `spend_limit_check_required` means attribution can succeed while final spend/account/usage checks remain caller/billing responsibilities. |
| payments | `payments-service/api/openapi/service.yaml` | Current checked surface is template/system only; payments/top-up runtime flows remain out of this microlease design. |

## Worker Lifecycle

`cmd/billing-worker` is triggered because terminal settlement, checkpoint/close
application, inbox retry, outbox relay, stale reconciliation, and admission
control renewal have a different lifecycle and scaling model from HTTP.

Worker requirements:

- bounded concurrency per account/microlease where row locks matter;
- signal-aware shutdown;
- readiness and dependency probes for Postgres and Redpanda;
- no HTTP request handler hidden background loops for durable settlement;
- low-cardinality metrics for lag, retry, and reconciliation.

## Coupling Controls

- Billing protected HTTP issue/replenish must not depend on Redpanda health for
  every request; it reads durable admission-control state instead.
- Admission-control renewal/closure is worker-owned and written to Postgres.
- Proxy active admission must not synchronously call billing or pricing after a
  valid microlease and child cap basis are in durable local state.
- Terminal settlement must tolerate out-of-order and duplicate events through
  billing inbox/idempotency.
- Checkpoint/close may arrive before all terminal facts; billing releases only
  proven unallocated capacity and keeps unresolved child cap reserved.

## Review Falsification Checks

Planning should be able to task implementation without inventing:

- a second money authority;
- a Redis runtime dependency;
- a direct per-request reserve fallback;
- a hidden worker inside HTTP handlers;
- a package cycle between app and infra;
- a runtime contract source outside OpenAPI/proto/migrations/query files.
