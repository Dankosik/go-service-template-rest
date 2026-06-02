# Durable Microlease Technical Design

Status: review-ready technical design for durable billing-issued microleases
Date: 2026-06-01
Owner: orchestrator
Consumes: `../spec.md`

## Approach

The microlease design is a tighter, shorter-lived refinement of the approved
billing-issued spending lease packet. It does not introduce a new money
authority.

The selected target is:

- billing-service issues small account-scoped microleases through protected HTTP
  and reserves the full USD exposure in billing PostgreSQL before any proxy can
  spend it;
- `gonka-proxy` admits paid external execution only after a local durable child
  debit authorization and durable terminal obligation commit under a current
  billing-issued owner/fence;
- process memory may cache grant state or run a deny-only precheck, but external
  execution still waits for durable child debit commit;
- Redis is not part of the first target runtime path. If a later phase adds it,
  it may only be a rebuildable limiter/cache/backpressure surface over durable
  authority, never a source of spend capacity;
- terminal facts, checkpoint facts, and close facts move asynchronously through
  durable outbox/inbox/event mechanics and settle or release the billing-held
  exposure in PostgreSQL;
- strict mode means cache bypass, smaller/shorter microlease issuance, durable
  child-debit-only admission, or fail closed. It does not mean direct
  per-request reserve fallback for migrated paid cohorts.

No specification reopen is needed. The design preserves zero unbacked spend
exposure because every paid execution has billing-reserved parent capacity and a
durable child debit before the external effect starts.

## Relationship To Existing Lease Packet

This packet refines `specs/event-driven-billing-money-architecture` without
editing it.

Preserved from the approved lease packet:

- billing PostgreSQL remains the customer-money correctness boundary;
- protected HTTP remains the issuance/readback/close command transport;
- proxy durable rows remain proof and recovery state, not visible balance truth;
- durable proxy child debit and terminal obligation are required before
  execution;
- Redpanda remains terminal/checkpoint/close transport, not money authority;
- direct per-request reserve fallback, pure cached balance authority, and
  branchy paid admission remain rejected for migrated paid cohorts;
- `test-plan.md` and `rollout.md` remain triggered before planning because proof
  and rollout are too broad for a task ledger alone.

Refined for microlease scale:

- lease caps are smaller and formula-bounded by child cap, active exposure,
  safety floor, and health gates;
- TTL and debit cutoff are short by default: `ttl=30s`, `debit_cutoff=25s`, and
  no new child debit may start after cutoff;
- refill begins only when remaining capacity drops below the configured refill
  threshold and terminal/reconciliation health is green;
- active microlease exposure is treated as visible reserved balance immediately;
- checkpoint/close evidence is mandatory before releasing unallocated capacity;
- memory is a cache/precheck over durable proxy state, not a token authority;
- Redis is explicitly absent from the first target rather than a hidden global
  counter.

This microlease packet should supersede the older lease packet for future
planning of this performance track only after its own technical design review
passes. Until then, the older packet remains a separate approved planning-ready
artifact for its own workflow.

## Initial Microlease Budgets

The first target uses configuration-backed limits with fail-closed defaults:

| Budget | Initial design value | Enforcement |
| --- | --- | --- |
| Per-microlease cap | `min(config.max_microlease_usd_atoms, 16 * max_child_cap_usd_atoms, account_active_exposure_remaining)` with `config.max_microlease_usd_atoms <= 100000000` (1.00 USD) for first rollout. | Billing transaction. |
| Account active exposure cap | `min(config.account_microlease_exposure_cap_usd_atoms, settled - reserved - safety_floor)` with first rollout cap `<= 200000000` (2.00 USD) unless a later approved rollout lowers it. | Billing transaction and balance constraints. |
| Safety floor | `max(2 * max_child_cap_usd_atoms, config.min_safety_floor_usd_atoms)`, with first rollout floor `>= 5000000` (0.05 USD). | Billing issue/replenish gate. |
| TTL | 30 seconds from billing commit. | Billing grant, proxy durable grant, and child debit cutoff. |
| Debit cutoff | 5 seconds before expiry, so new child debits stop at 25 seconds. | Proxy allocator and billing terminal validation. |
| Refill threshold | Refill may start at `remaining <= max(4 * max_child_cap_usd_atoms, 25% of cap)` only when health gates pass. | Proxy request plus billing issue/replenish gate. |
| Terminal deadline | 120 seconds after child debit commit unless the route-specific max is shorter. | Proxy terminal obligation and reconciliation. |
| Stale child/debit warning | Oldest unresolved child age >= 60 seconds. | Admission controls and metrics. |
| Stale child/debit critical | Oldest unresolved child age >= 180 seconds or terminal deadline breach. | Refill denied; strict/fail-closed. |
| Reconciliation SLA | Open or update a reconciliation case within 5 minutes of microlease expiry, terminal deadline breach, or critical lag breach. | Billing worker. |

Runtime defaults must be fail-closed: if a cap, TTL, freshness, or health budget
is absent or malformed, billing denies new microlease capacity and proxy cannot
fall back to direct reserve or cached balance.

The numeric values above are planning inputs and release gates. They may be
lowered by configuration during rollout. Raising them above the first-rollout
caps requires technical design review or a recorded rollout risk acceptance.

## Artifact Index

| Artifact | Status | Trigger / Rationale |
| --- | --- | --- |
| `design/overview.md` | review-ready | Entry point, selected approach, lease-packet reconciliation, budgets, artifact status, and review handoff. |
| `design/component-map.md` | review-ready | Required split artifact for billing packages, generated surfaces, proxy allocator obligations, workers, and stable non-touches. |
| `design/sequence.md` | review-ready | Required split artifact for issue/replenish, precheck, child debit, terminal settlement, checkpoint/close, expiry, reconciliation, and outage behavior. |
| `design/ownership-map.md` | review-ready | Required split artifact for source-of-truth ownership, dependency direction, generated code, adapter responsibility, and explicit non-owners. |
| `design/data-model.md` | triggered, review-ready | Persisted microlease state, child terminal projection, checkpoint/close evidence, inbox/outbox, cache contract, replay, migration, and retention change. |
| `design/dependency-graph.md` | triggered, review-ready | New app/infra packages, protected HTTP flow, event contract flow, worker lifecycle, cross-repo proxy allocator, and no-Redis runtime dependency. |
| `design/contracts/protected-http.md` | triggered, review-ready | Protected microlease issue/replenish/readback/close contract design. Design-only; `api/openapi/service.yaml` remains runtime authority. |
| `design/contracts/redpanda-events.md` | triggered, review-ready | Terminal, checkpoint, close, rejection, and billing fact event design. Design-only; future `api/proto/events/v1/*.proto` inputs remain runtime authority. |
| `test-plan.md` | triggered, review-ready | Validation spans money math, persistence, proxy durability, events, privacy, performance, Redis absence/degrade policy, and rollout proof. |
| `rollout.md` | triggered, review-ready | Mixed-version and cutover choreography affects money correctness, no dual writer, no direct reserve fallback, rollback/failback, and operator gates. |
| `tasks.md` | missing by design | Planning is blocked until mandatory technical design review records `PASS` or eligible `CONCERNS`. |

## Review Readiness

Technical design review can start next. The review packet is:

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
13. `../workflow-plans/specification.md`
14. `../workflow-plans/technical-design.md`
15. read-only context from `specs/event-driven-billing-money-architecture`
16. `docs/PRD.md`, `docs/critical-billing-context.md`, and
    `docs/repo-architecture.md`

The review must be read-only. It must not write `tasks.md`, migrations, runtime
schemas, generated artifacts, adapters, tests, implementation code, or edits to
the existing `event-driven-billing-money-architecture` packet.

## Reopen Conditions

Reopen specification if a later phase needs any of the following:

- memory-only or Redis-only spend before durable child debit;
- direct per-request reserve fallback for migrated paid cohorts;
- a nonzero platform write-off budget for unrecorded local/Redis spend;
- weaker billing PostgreSQL authority or visible exposure subtraction;
- weaker proxy durable child debit or terminal obligation before execution;
- broader payment/top-up/account/pricing/API-key authority;
- pricing-service cannot provide or attest USD-compatible immutable pricing
  snapshot evidence;
- a web-search-like paid path cannot map into microlease, child debit, terminal
  settlement, write-off, reversal, and reconciliation semantics;
- performance cannot be met with billing-issued reserved microleases plus
  durable proxy child debit allocation.

Reopen technical design if review or planning cannot task package boundaries,
contracts, persisted state, worker lifecycle, proxy allocator obligations,
failure semantics, rollout gates, or validation proof without choosing a new
design.
