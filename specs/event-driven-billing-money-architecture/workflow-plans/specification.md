# Specification

Phase: specification
Status: complete after reopen
Owner: orchestrator
Master plan: `../workflow-plan.md`
Spec: `../spec.md`

## Session Outcome

Reopened the specification phase for the event-driven billing money
architecture workflow after the user changed the performance requirement before
planning.

Outcome:

- replaced the prior protected per-request reserve target with a
  billing-issued account-scoped spending lease architecture;
- approved `../spec.md` as the canonical reopened decision record;
- reran and reconciled the required formal spec clarification challenge across
  the broad protected-domain lens set;
- updated master workflow state;
- marked the prior design packet, `test-plan.md`, `rollout.md`, and follow-up
  technical design review PASS as superseded historical context for planning;
- stopped before technical design repair.

No design files, `tasks.md`, migrations, generated SQL, runtime schemas,
runtime adapters, tests, generated artifacts, or implementation code were
written in this reopen session.

## Inputs Read

Workflow authority:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/specification-session/SKILL.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/spec-clarification-challenge/SKILL.md`
- `.agents/skills/go-distributed-architect-spec/SKILL.md`
- `.agents/skills/go-data-architect-spec/SKILL.md`
- `.agents/skills/go-domain-invariant-spec/SKILL.md`
- `.agents/skills/go-performance-spec/SKILL.md`

Active workflow and research:

- `specs/event-driven-billing-money-architecture/workflow-plan.md`
- `specs/event-driven-billing-money-architecture/workflow-plans/specification.md`
- `specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md`
- `specs/event-driven-billing-money-architecture/research/current-provider-consumer-evidence.md`
- `specs/event-driven-billing-money-architecture/research/fan-in-synthesis.md`

Current canonical and superseded design context:

- `specs/event-driven-billing-money-architecture/spec.md`
- `specs/event-driven-billing-money-architecture/design/overview.md`
- `specs/event-driven-billing-money-architecture/design/component-map.md`
- `specs/event-driven-billing-money-architecture/design/sequence.md`
- `specs/event-driven-billing-money-architecture/design/ownership-map.md`
- `specs/event-driven-billing-money-architecture/design/data-model.md`
- `specs/event-driven-billing-money-architecture/design/dependency-graph.md`
- `specs/event-driven-billing-money-architecture/design/contracts/protected-http.md`
- `specs/event-driven-billing-money-architecture/design/contracts/redpanda-events.md`
- `specs/event-driven-billing-money-architecture/test-plan.md`
- `specs/event-driven-billing-money-architecture/rollout.md`

Product and repository context:

- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Sibling/provider evidence checked read-only as needed:

- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/pricing-service/README.md`
- `env/migrations/000003_billing_money_core.up.sql`
- `internal/infra/postgres/queries/billing_money_core.sql`

## Readiness Check

Result: specification-ready after reopen.

Reasoning:

- The user explicitly changed the architecture requirement before `tasks.md`
  was written and directed the workflow to reopen specification.
- The existing workflow already recorded spend tokens, allowance windows, and
  account leases as specification reopen triggers.
- Prior research identified billing-issued spend rights as conditionally viable
  if billing reserves the full exposure in PostgreSQL before proxy uses them.
- The critical decision frontier was spec-level: whether the target path should
  remain per-request reserve or become a billing-owned pre-authorized budget
  model.
- Existing local and sibling evidence was sufficient to decide the target
  architecture and leave concrete route names, schemas, table shapes, allocator
  mechanics, and rollout sequence to technical design.

No upstream research phase was reopened in this session.

## Clarification Challenge

Status: complete after rerun.

Formal challenge was required because the work is full-orchestrated,
high-impact, protected customer-money, API, data, reliability, delivery, and
privacy scope. The architecture decision changed materially after an approved
specification and a PASS follow-up technical design review.

Lane set:

| Lane | Role | Skill | Lens | Concrete approval-critical question | Status |
| --- | --- | --- | --- | --- | --- |
| C1 | `challenger-agent` | `spec-clarification-challenge` | scope and spec coherence | Does the lease candidate close the specification boundary without reintroducing phased architecture, branchy request paths, hidden payment/pricing/API-key scope moves, or design/task detail that belongs later? | complete |
| C2 | `challenger-agent` | `spec-clarification-challenge` | domain invariants and edge cases | Can the lease candidate guarantee no customer overspend and exact ledger math across concurrent requests, multiple proxy instances, crashes, replay, lease expiry, delayed terminal events, and proxy over-debit bugs? | complete |
| C3 | `challenger-agent` | `spec-clarification-challenge` | architecture ownership and dependency boundaries | Are ownership and dependency directions clear across billing, proxy, pricing, API-key, identity, Redpanda, and Postgres for lease issuance, local debits, replenishment, settlement, and reconciliation? | complete |
| C4 | `challenger-agent` | `spec-clarification-challenge` | API, data, compatibility, and source-of-truth consequences | Does the candidate settle the spec-level contract/data authority for lease issuance, replenishment, checkpoint/close, terminal event identity, idempotency, replay, retention, and compatibility? | complete |
| C5 | `challenger-agent` | `spec-clarification-challenge` | security, reliability, delivery, and validation proof | Are outage, revocation/expiry, broker failure, proxy crash, producer authenticity, privacy, rollout, and proof obligations concrete enough without weakening the prior protected-domain guarantees? | complete |

Scoped-down rationale: not used. The default five-lens formal challenge ran.

## Fan-In Resolution

| Finding area | Strongest classification | Resolution | Owner / status |
| --- | --- | --- | --- |
| Reopen legitimacy | `blocks_spec_approval` | Reopened specification intentionally replaces approved per-request reserve `D1/D2/D7` direction. The prior design and TDR PASS are superseded for planning. | specification closed |
| Uniform path | `blocks_spec_approval` | Reopened spec rejects direct per-request reserve fallback, pure cached balance checks, proxy-local mutable balances, and risk-based branchy paid admission for migrated cohorts. | specification closed |
| Scope boundary | `blocks_spec_approval` | Lease scope is paid usage admission and settlement only. Payment/top-up, pricing catalog, model routing, API-key lifecycle/policy configuration, and identity authority remain out of scope. | specification closed |
| Spec vs design placement | `blocks_spec_approval` | Spec records authority, invariants, rejected paths, outage policy, proof obligations, and handoff. Exact routes, schemas, table shapes, allocator mechanics, and rollout details are deferred to technical design. | specification closed |
| Design/TDR invalidation | `blocks_spec_approval` | `workflow-plan.md` and `spec.md` mark the previous design, `test-plan.md`, `rollout.md`, and TDR PASS as stale historical context only. | specification closed |
| Lease ledger invariant | `blocks_spec_approval` | `spec.md` requires lease issuance/replenishment to reserve full USD exposure in billing PostgreSQL and child debits to be strict children capped by child and lease authority. | specification closed |
| Concurrent proxy fencing | `blocks_spec_approval` | `spec.md` requires lease owner, generation/fence, and durable single-writer allocation semantics. Concrete allocator choice is technical design. | specification closed with design deferral |
| Proxy crash recovery | `blocks_spec_approval` | `spec.md` requires durable proxy lease grants, child debit allocations, terminal obligations, and recovery states before execution. | specification closed |
| Delayed terminal after expiry | `blocks_spec_approval` | `spec.md` requires terminal settlement against original lease/debit authority, even after lease expiry or newer lease issuance. | specification closed |
| Proxy over-debit bug | `blocks_spec_approval` | `spec.md` caps charge by child authorization and aggregate lease budget, with write-off/reconciliation for excess or invalid child evidence. | specification closed |
| Lease lifecycle | `blocks_spec_approval` | `spec.md` defines issuance, replenishment, debit allocation, terminal settlement, checkpoint/close, expiry, cancellation, stale recovery, and release boundaries at decision level. | specification closed |
| Settlement identity | `blocks_spec_approval` | `usageOperationId` remains usage settlement identity; `spendingLeaseId`, fence, and `debitAuthorizationId` are required lineage and cap authority. | specification closed |
| Proxy durable debit proof | `blocks_spec_approval` | Proxy durable debit records are proof inputs and recovery anchors, not balance authority. Billing validates lease/debit lineage before money mutation. | specification closed |
| Redpanda event authority | `blocks_specific_domain` | `spec.md` expands current-scope events to include lease checkpoint/close and billing lease/debit facts. Runtime schema source remains a technical-design contract task under repository-owned event inputs. | specification closed with design deferral |
| Compatibility and no dual writer | `blocks_specific_domain` | Compatibility bridge may only adapt old proxy calls to target lease/debit semantics and must not preserve per-request reserve or proxy-local balance writes as long-lived paths. | specification closed |
| Expiry, revocation, and fail-closed | `blocks_spec_approval` | `spec.md` requires expired/revoked/stale-fenced leases to stop new child debits, allows already minted capacity only within cutoff/cap, and fails closed when no valid capacity exists. | specification closed |
| Broker outage and lag | `blocks_spec_approval` | `spec.md` keeps terminal facts durable in proxy, keeps lease exposure reserved until settlement/reconciliation, and adds stale-lease/debit backpressure obligations. | specification closed |
| Producer authenticity and privacy | `blocks_specific_domain` | `spec.md` keeps service auth, producer authority, safe payload fields, route-template logging, low-cardinality metrics, and no bearer spend-token leakage. | specification closed |
| Validation | `blocks_spec_approval` | `spec.md` adds lease issuance, local debit allocation, fencing, expiry, close, over-debit, crash recovery, lag, rollout, privacy, and performance proof obligations. | specification closed |

No lane finding remains an approval blocker.

Rerun trigger:

- rerun formal clarification if technical design changes the uniform lease path,
  introduces direct per-request reserve fallback, expands payment/top-up scope,
  changes pricing/account/API-key authority, moves terminal durability away from
  durable proxy submission plus billing inbox, weakens billing PostgreSQL money
  authority, or weakens outage/auth/privacy policy.

## Spec Status

`../spec.md`: approved after reopen.

Approval rationale:

- The spec chooses the production-ready target architecture rather than an
  MVP/future-hardening split.
- It selects a billing-owned pre-authorized budget model because it preserves
  billing-authorized funds while avoiding a billing reserve call for every tiny
  paid request.
- It rejects pure fire-and-forget usage for normal paid usage unless backed by
  a billing-issued lease.
- It rejects direct per-request reserve fallback as target behavior for migrated
  paid cohorts.
- It assigns customer-money authority to billing-service PostgreSQL.
- It defines proxy local durable accounting as non-authoritative recovery proof.
- It records lease/debit identity, fencing, expiry, outage, privacy, rollout,
  and validation obligations at specification level.
- Remaining unknowns are explicit technical-design deferrals or reopen
  conditions, not hidden implementation decisions.

## Artifact Status

| Artifact | Status | Notes |
| --- | --- | --- |
| `../spec.md` | approved after reopen | Canonical decision record now selects billing-issued spending leases. |
| `../workflow-plan.md` | updated | Master state routes next to technical design repair. |
| `specification.md` | complete after reopen | This phase-local record. |
| `../design/` | stale, expected next repair | Existing design is historical context for old reserve architecture. |
| `../design/contracts/` | stale, expected next repair | Existing contract design must be repaired for lease APIs/events. |
| `../test-plan.md` | stale, expected repair before planning | Must include lease/debit proof obligations. |
| `../rollout.md` | stale, expected repair before planning | Must include lease path cutover and direct-reserve fallback disablement. |
| `technical-design-review.md` | superseded historical PASS | Fresh technical design review required after repair. |
| `../tasks.md` | missing, blocked | Planning must wait for repaired design and fresh technical design review. |

## Blockers And Reopen Conditions

Blockers:

- None for starting technical design repair.
- Planning is blocked until repaired technical design and fresh technical design
  review pass.

Reopen specification if:

- pricing-service cannot provide or attest USD-compatible immutable pricing
  snapshot evidence;
- a current web-search paid path cannot map into the shared
  lease/debit/finalize/write-off/reversal model;
- the target performance envelope cannot be met with billing-issued leases and
  durable local debit allocation without moving balance authority to proxy or
  cache;
- design introduces direct per-request reserve fallback, branchy request paths,
  or pure cached balance admission;
- design expands payment/top-up or payment-evidence ingestion scope;
- design changes account-scope, spend-limit, pricing, API-key, terminal
  durability, billing Postgres authority, privacy, or outage policy.

## Completion Marker

Complete because:

- `../spec.md` is reopened, updated, and approved;
- required formal clarification challenge reran and was reconciled;
- `../workflow-plan.md` is updated for the phase boundary;
- the prior design/TDR/test/rollout packet is explicitly marked stale for
  planning;
- no prohibited downstream artifacts or runtime files were created or modified.

## Stop Rule

Stop after specification reopen. Do not start technical design repair,
technical design review, planning, tasks, migrations, generated SQL, runtime
schemas, adapters, tests, generated artifacts, or implementation in this
session.

## Next Action

Start the technical design phase in the next session.

Expected next output:

- repair the task-local technical design bundle from approved `../spec.md`;
- update protected HTTP and Redpanda contract design context for lease issuance,
  replenishment, readback, checkpoint/close, terminal events, and billing facts;
- update data-model delta for spending leases, child debit authorization
  lineage, event inbox/outbox, lease/debit expiry and reconciliation, and any
  required repair to existing money primitives;
- update sequence/failure design for lease issuance, local durable debit
  allocation, terminal event settlement, lease checkpoint/close,
  reconciliation, proxy durable submission, billing inbox retry, and outbox
  relay;
- update ownership and dependency maps for billing-service, gonka-proxy,
  pricing-service, api-key-service, identity-service, Redpanda, and PostgreSQL;
- update `test-plan.md` and `rollout.md` or route them explicitly if split from
  design;
- update workflow state and stop before technical design review unless the next
  phase file records a valid technical-design-only boundary.
