# Event-Driven Billing Money Architecture Technical Design

Status: repaired review-ready technical design for billing-issued spending leases
Date: 2026-06-01
Owner: orchestrator
Consumes: `../spec.md`

## Approach

The design implements the reopened specification's uniform lease path:

- billing-service issues bounded account-scoped spending leases through
  protected HTTP and reserves the full lease USD exposure in billing Postgres
  before proxy can spend it;
- `gonka-proxy` admits paid work only by atomically allocating a durable child
  debit authorization from an active billing-minted lease generation;
- the proxy hot path has no direct per-request reserve fallback, no risk-based
  branch between reserve and lease admission, no pure cached-balance authority,
  and no process-local spending allowance;
- terminal settlement remains asynchronous: proxy durable terminal submission,
  Redpanda transport, billing durable inbox, billing Postgres ledger effects,
  billing outbox facts, and reconciliation;
- billing-service Postgres remains the only customer-money correctness
  boundary. Proxy durable lease/debit rows are recovery proof, not visible
  balance truth.

No specification reopen is needed. The design does not introduce payment/top-up
scope, pricing/account/API-key authority changes, Redpanda reserve commands,
weaker terminal durability, weaker billing Postgres money authority, or weaker
outage/privacy policy.

## Bundle Index

| Artifact | Status | Trigger / Rationale |
| --- | --- | --- |
| `design/overview.md` | repaired review-ready | Entry point, selected design forks, artifact status, assumptions, reopen conditions, and review handoff. |
| `design/component-map.md` | repaired review-ready | Required split artifact for billing/proxy packages, generated surfaces, lease allocator ownership, workers, and stable non-touches. |
| `design/sequence.md` | repaired review-ready | Required split artifact for lease issuance/replenishment, proxy child debit allocation, terminal settlement, checkpoint/close, expiry, reconciliation, and outages. |
| `design/ownership-map.md` | repaired review-ready | Required split artifact for money authority, lease/debit ownership, dependency direction, generated-code authority, and explicit non-owners. |
| `design/data-model.md` | repaired review-ready | Triggered by spending lease state, child debit lineage, checkpoint/close state, inbox/outbox, admission controls, migration shape, replay, and retention. |
| `design/dependency-graph.md` | repaired review-ready | Triggered by new app/infra packages, protected OpenAPI flow, protobuf event flow, proxy durable allocator dependency, and worker lifecycle. |
| `design/contracts/protected-http.md` | repaired review-ready | Triggered by protected lease issuance/replenishment/readback/close contracts, idempotency, auth, and Problem mapping. Design-only; `api/openapi/service.yaml` remains canonical. |
| `design/contracts/redpanda-events.md` | repaired review-ready | Triggered by terminal, lease checkpoint/close, billing lease/debit facts, reconciliation, rejection facts, and runtime protobuf authority. |
| `test-plan.md` | repaired review-ready | Required before planning because proof spans money math, lease/debit data, proxy durability, fencing, events, privacy, performance, and rollout. |
| `rollout.md` | repaired review-ready | Required before planning because cutover needs lease-path migration, no dual writer, bridge exit, rollback/failback, and mixed-version gates. |
| `tasks.md` | missing by design | Planning is blocked until a fresh technical design review records `PASS` or eligible `CONCERNS`. |

## Selected Design Forks

### Concrete Lease Allocator Identity

Selected: leases are issued to a stable proxy allocator owner of the form
`proxy:<environment>:<allocatorShardId>` with a billing-issued
`spendingLeaseGeneration` / `leaseFence`. The proxy may run multiple processes
for one allocator shard only through one durable proxy allocator row-lock or
compare-and-swap boundary. Process memory may cache lease state, but it cannot
allocate child debits without durable allocator state.

Reason: per-process leases reduce coordination but create excessive lease churn
and stale exposure during restarts. A shared cache or Redis allowance would move
correctness outside billing and proxy durable storage. Per-shard durable
allocation keeps one writer per lease generation while still allowing multiple
billing-reserved leases for the same account when capacity is genuinely needed.

### Lease Issuance And Replenishment Transport

Selected: protected billing-service HTTP is the target transport for
spending-lease issue/replenish/readback/close commands. Redpanda does not carry
lease reserve commands in this scope.

Reason: proxy must receive a durable billing outcome before using new lease
capacity. Protected HTTP plus same-identity retry/readback is simpler than an
async reserve command/outcome pair and keeps billing Postgres as the issuance
transaction boundary.

### Local Child Debit Allocation

Selected: proxy local child debit allocation is a durable proxy transaction that
atomically reduces local remaining lease authority, creates one
`debitAuthorizationId`, stores the `usageOperationId`, child cap, fingerprint,
pricing snapshot identity, terminal deadline, and creates the terminal
submission obligation before external execution.

Reason: this is the only paid hot path after a lease is active. The proxy does
not call billing per request and does not rely on memory-only allowance.

### Terminal Mutation Ingress

Selected: terminal money mutation is Redpanda-only for the target lease path.
Billing protected HTTP provides lease commands and readback, not a parallel
proxy finalize/write-off mutation path.

Reason: a second terminal mutation ingress would widen ordering and replay
surface. Terminal facts are already durable in the proxy before publish, and
billing consumes them through an inbox before committing broker offsets.

### Lease Checkpoint And Close

Selected: proxy sends lease checkpoint and close facts through Redpanda, and
may also call protected HTTP close/cancel when it needs bounded synchronous
readback. Billing releases unused capacity only after it verifies the lease
owner, generation/fence, checkpoint fingerprint, allocated child high-water
mark, terminal submission coverage, and absence of unresolved open children.

Reason: checkpoint/close is not a per-request hot path. Redpanda gives durable
replay for close evidence, while protected HTTP close supports controlled
rollout, operator repair, and ambiguous readback without making Redpanda money
truth.

### Admission Backpressure Surface

Selected: retain a billing-owned Postgres `billing_admission_controls` surface,
but apply it to new lease issuance/replenishment and operator-controlled paid
admission gates, not to every child debit. `cmd/billing-worker` renews healthy
`open` control leases or writes `throttle` / `fail_closed` when terminal lag,
stale child debits, stale leases, reconciliation backlog, or worker health
breaches budgets. Protected operator/admin tooling may override through the
same audited table.

`cmd/service` reads the global and account rows inside lease issuance or
replenishment transactions before reserving more lease capacity. Missing,
expired, stale, malformed, `throttle`, or `fail_closed` controls reject new
lease capacity without money mutation. Active already-minted lease capacity may
continue only until its cap, owner fence, debit cutoff, and local backlog policy
allow it.

Reason: the control state remains in the billing Postgres correctness boundary
and avoids coupling the HTTP server directly to Redpanda readiness. It also
keeps the approved outage policy: no new unbacked capacity when settlement
health is unsafe.

### Runtime Redpanda Schema Authority

Selected: current-scope Redpanda event schemas are authored as protobuf
contracts under `api/proto/events/v1/*.proto`.

Derived Go DTOs are generated into a repository-owned internal package such as
`internal/api/events/v1` and used only by Redpanda adapters. App logic receives
app-owned command/fact types after adapter mapping. Planning must add
repository-owned proto lint/generate/drift and compatibility checks before
runtime producers or consumers are implemented.

Reason: `docs/repo-architecture.md` reserves `api/proto/` for non-HTTP
contract surfaces, and money-critical event identity, amount meaning, finality,
producer authenticity, and replay semantics need versioned compatibility rules.

### Protected HTTP Identifier Placement

Selected: new protected money routes put account, lease, debit, and operation
identifiers in request bodies rather than path variables because current access
logging records raw paths.

Reopen technical design if route shape changes or path redaction is
implemented and reviewed.

## Accepted Assumptions

- Current evidence is static local repository and sibling repository
  source/contract evidence. No live deployment, production database, or traffic
  distributions were used.
- `pricing-service` can provide or attest USD-compatible immutable pricing
  snapshot evidence before implementation planning approves lease issuance and
  child debit allocation. If not, reopen specification.
- Current web-search-like paid operations can map into the shared
  lease/debit/finalize/write-off/reversal model. If not, reopen specification.
- The old proxy shared-balance bridge and TypeBox internal-money routes are
  source evidence and compatibility inputs only. Billing-service OpenAPI is the
  target HTTP contract authority.
- Payment/top-up and payment-evidence ingestion remain out of scope.

## Review Readiness

Fresh technical design review can start next. The review packet is:

1. `../spec.md`
2. `design/overview.md`
3. `design/component-map.md`
4. `design/sequence.md`
5. `design/ownership-map.md`
6. `design/data-model.md`
7. `design/dependency-graph.md`
8. `design/contracts/protected-http.md`
9. `design/contracts/redpanda-events.md`
10. `test-plan.md`
11. `rollout.md`
12. `../workflow-plan.md`
13. `../workflow-plans/technical-design.md`
14. `../workflow-plans/specification.md`
15. `../workflow-plans/technical-design-review.md` as historical context only
16. `docs/PRD.md`, `docs/critical-billing-context.md`, and `docs/repo-architecture.md`

The review must be read-only and must not write `tasks.md`, migrations,
generated SQL, runtime schemas, runtime adapters, tests, implementation code,
or review-driven repairs in the review lane.

## Reopen Conditions

Reopen specification if a later phase needs:

- direct per-request reserve fallback for migrated paid cohorts;
- branchy paid admission where some paid requests use leases and others use
  direct reserve or pure cached balance;
- Redpanda reserve command/outcome as the normal lease issuance gate;
- payment/top-up or payment-evidence ingestion scope;
- account scope, pricing authority, API-key policy authority, or spend-limit
  authority changes;
- weaker proxy terminal durability than durable local terminal submission
  before external execution;
- weaker billing Postgres money authority, fail-closed outage behavior,
  producer authenticity, or privacy policy.

Reopen technical design if review finds that planning cannot task package
boundaries, lease/debit data deltas, protected HTTP contracts, event contracts,
worker lifecycle, proxy durable allocator obligations, rollout gates, or
validation proof without choosing a new design.
