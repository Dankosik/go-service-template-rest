# Technical Design

Phase: technical design
Status: complete after lease-architecture repair
Owner: orchestrator
Master plan: `../workflow-plan.md`
Spec: `../spec.md`
Date: 2026-06-01

## Session Outcome

Completed the technical design repair for the reopened event-driven billing
money architecture. The repaired packet now targets billing-issued
account-scoped spending leases, proxy durable child debit allocation,
lease/debit fencing, expiry, checkpoint/close, asynchronous terminal
settlement, reconciliation, protected HTTP lease contracts, Redpanda event
contracts, updated test obligations, and rollout choreography.

The prior design packet, `test-plan.md`, `rollout.md`, and prior technical
design review PASS remain historical context only for the superseded
per-request reserve architecture.

This phase did not implement runtime code, migrations, generated SQL, runtime
schemas, adapters, tests, generated artifacts, or `tasks.md`.

## Inputs Read

Workflow authority:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/technical-design-session/SKILL.md`
- `.agents/skills/go-design-spec/SKILL.md`

Active workflow:

- `../workflow-plan.md`
- `../workflow-plans/specification.md`
- `../spec.md`
- stale `../workflow-plans/technical-design.md`
- historical `../workflow-plans/technical-design-review.md`
- stale design bundle, `../test-plan.md`, and `../rollout.md`
- `../research/current-provider-consumer-evidence.md`
- `../research/fan-in-synthesis.md`

Product and repository context:

- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `api/openapi/service.yaml`
- `api/proto/service/v1/service.proto`
- `env/migrations/000003_billing_money_core.up.sql`
- `internal/infra/postgres/queries/billing_money_core.sql`

Provider/consumer contract evidence:

- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go`
- `/Users/daniil/Projects/GonkaGate/pricing-service/README.md`
- `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`

## Entry Readiness

Result: ready for technical design and complete after repair.

Reasoning:

- `../spec.md` is approved after reopen and records all approval-changing
  lease architecture decisions needed for this phase.
- Workflow state had no blockers for starting technical design repair.
- The required reopen triggers were known: direct per-request reserve fallback,
  branchy paid admission, pure cached balance authority, payment/top-up scope,
  changed pricing/account/API-key authority, weaker terminal durability, weaker
  billing Postgres money authority, or weaker outage/privacy policy.
- The design completed without needing any of those changes.

## Artifact Status

| Artifact | Status | Notes |
| --- | --- | --- |
| `../design/overview.md` | repaired review-ready | Entry point, bundle index, selected lease design forks, assumptions, reopen conditions, and review packet. |
| `../design/component-map.md` | repaired review-ready | Billing/proxy package surfaces, generated OpenAPI/protobuf flow, proxy durable allocator, worker lifecycle, stable non-touches, and bridge placement. |
| `../design/sequence.md` | repaired review-ready | Lease issuance/replenishment, child debit allocation, terminal settlement, checkpoint/close, inbox retry, outbox relay, expiry, reconciliation, and backpressure. |
| `../design/ownership-map.md` | repaired review-ready | Billing money authority, proxy allocator ownership, lease/debit source-of-truth rules, dependency direction, and explicit non-owners. |
| `../design/data-model.md` | repaired review-ready | Spending leases, child debit settlement lineage, checkpoints, inbox/outbox, admission controls, ledger deltas, replay/conflict, migration, retention, and privacy. |
| `../design/dependency-graph.md` | repaired review-ready | Runtime graph, package graph, protected HTTP dependency, Redpanda dependency, admission-control coupling, Postgres risks, and cross-service evidence. |
| `../design/contracts/protected-http.md` | repaired review-ready | Protected lease issue/replenish/readback/close contracts, operation/balance readback, auth, idempotency, and Problem mapping. |
| `../design/contracts/redpanda-events.md` | repaired review-ready | Terminal, lease checkpoint/close, billing lease/debit facts, protobuf authority, producer authenticity, retention, and evolution. |
| `../test-plan.md` | repaired review-ready | Triggered by broad lease/debit, proxy durability, fencing, DB, event, privacy, performance, and rollout proof obligations. |
| `../rollout.md` | repaired review-ready | Triggered by lease-path cutover, proxy durable allocator rollout, old writer/bridge disablement, rollback/failback, and mixed-version behavior. |
| `technical design review` | missing fresh review | Prior PASS is superseded historical context. Fresh review is mandatory before planning. |
| `../tasks.md` | missing, blocked | Planning must wait for fresh technical design review. |

## Selected Design Decisions

| Area | Decision | Reopen condition |
| --- | --- | --- |
| Lease allocator identity | Leases are issued to stable proxy allocator owners such as `proxy:<environment>:<allocatorShardId>` and fenced by generation/token. One durable allocator may spend one lease generation at a time. | Reopen technical design if review proves per-shard allocator ownership cannot satisfy multi-proxy concurrency or operational rollover. |
| Lease issuance/replenishment | Protected billing HTTP issues/replenishes lease capacity and reserves full USD exposure in billing Postgres. | Reopen specification if Redpanda reserve commands or direct per-request reserve become necessary. |
| Paid hot path | Proxy durably allocates one child debit from active lease capacity before external execution. | Reopen specification if the design needs pure cached balance, process-local allowance, or direct reserve fallback. |
| Terminal settlement | Proxy publishes durable terminal facts; billing settles through Redpanda inbox and Postgres. No target HTTP finalize/write-off mutation path. | Reopen specification if terminal durability is weakened or a second terminal truth is introduced. |
| Lease checkpoint/close | Proxy emits checkpoint/close facts and may use protected HTTP close for bounded readback. Billing releases only verified safe unused capacity. | Reopen technical design if review needs a single transport for all close operations. |
| Admission backpressure | `billing_admission_controls` gates new lease issuance/replenishment. Active leases may spend only within cap/fence/cutoff and local proxy health. | Reopen technical design if Postgres controls cannot meet lag/stale exposure budgets. |
| Runtime event schema authority | Current-scope Redpanda schemas live under `api/proto/events/v1/*.proto`; generated DTOs are adapter-owned. | Reopen technical design if protobuf tooling cannot be made repository-owned. |
| Protected HTTP identifiers | Account, lease, debit, and operation identifiers are in request bodies because current access logs include raw paths. | Reopen technical design if route shape changes or path redaction is implemented and reviewed. |

## Blockers And Reopen Conditions

Blockers:

- None for starting fresh technical design review.
- Planning remains blocked until technical design review records `PASS` or
  eligible `CONCERNS` on the repaired lease packet.

Reopen specification if a later phase needs:

- direct per-request reserve fallback, branchy paid admission, pure cached
  balance authority, process-local allowance, or proxy-local mutable balance
  authority for migrated cohorts;
- Redpanda reserve command/outcome in the normal lease issuance path;
- payment/top-up or payment-evidence ingestion scope;
- account scope, pricing authority, API-key policy authority, or spend-limit
  authority changes;
- weaker proxy terminal durability than durable local submission before
  external execution;
- weaker fail-closed, privacy, producer-authenticity, or outage policy.

Reopen technical design if review finds planning cannot task:

- package boundaries and dependency direction;
- protected HTTP or Redpanda contract context;
- lease/debit data-model deltas;
- worker lifecycle and retry ownership;
- proxy durable allocator and terminal submission obligations;
- rollout gates and compatibility exit criteria;
- validation proof obligations.

## Technical Design Review Handoff

Handoff: ready for fresh technical design review.

Review must inspect:

- approved `../spec.md`;
- the full repaired design bundle under `../design/`;
- `../test-plan.md`;
- `../rollout.md`;
- this phase file;
- `../workflow-plan.md`;
- `../workflow-plans/specification.md`;
- prior `../workflow-plans/technical-design-review.md` as historical context
  only;
- `docs/PRD.md`, `docs/critical-billing-context.md`, and
  `docs/repo-architecture.md`;
- current proxy, pricing, and API-key contract evidence as needed.

The review must be read-only and risk-driven. It must verify that the repaired
packet preserves billing-issued spending leases, proxy durable child debit
allocation, fencing, expiry, checkpoint/close, terminal settlement durability,
billing Postgres money authority, outage behavior, auth/privacy, validation,
and rollout. It must record a distinct technical design review verdict with
`PASS`, eligible `CONCERNS`, or `FAIL`. Planning remains blocked until that
verdict exists and has no unresolved planning blockers.

## Completion Marker

Complete because:

- all required split design artifacts were repaired for the approved lease
  architecture;
- all triggered conditional artifacts were repaired with concrete content;
- workflow control is updated for the phase boundary;
- no prohibited downstream artifacts or runtime files were created.

## Stop Rule

Stop after technical design. Do not start technical design review, planning,
`tasks.md`, migrations, generated SQL, runtime schemas, runtime adapters,
tests, generated artifacts, or implementation in this session.

## Next Action

Start fresh technical design review in the next session.

Expected next output:

- read-only technical design review record for the repaired lease packet listed
  above;
- verdict `PASS`, `CONCERNS` with named proof obligations, or `FAIL` with
  reopen target;
- updated workflow state;
- stop before planning.
