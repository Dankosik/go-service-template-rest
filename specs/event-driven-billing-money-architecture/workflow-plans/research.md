# Research

Phase: research
Status: complete
Owner: orchestrator
Master plan: `../workflow-plan.md`

## Session Outcome

Ran the R1-R9 read-only research fan-out for the event-driven billing money architecture workflow, reconciled the lane outputs, preserved compact findings under `../research/`, updated the master workflow state, and stopped before specification.

Success criteria:
- R1-R9 complete with facts, inferences, assumptions, risks, open points, and handoff implications reconciled by the orchestrator.
- Current provider/consumer evidence checked from the active repositories, especially `gonka-proxy` money paths and `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`.
- Compact findings preserved in `../research/current-provider-consumer-evidence.md` and `../research/fan-in-synthesis.md`.
- No `spec.md`, design files, `tasks.md`, `test-plan.md`, `rollout.md`, migrations, generated SQL, runtime adapters, tests, or implementation code written.

## Evidence Read In This Phase

Workflow authority and product context:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/event-driven-billing-money-architecture/workflow-plan.md`
- `specs/event-driven-billing-money-architecture/workflow-plans/workflow-planning.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Historical context only:
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`

Current provider/consumer evidence sampled for research synthesis:
- `api/openapi/service.yaml`
- `api/proto/service/v1/service.proto`
- `env/migrations/000003_billing_money_core.up.sql`
- `internal/infra/postgres/queries/billing_money_core.sql`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/shared-balance-live.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/balance.service.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/balance/balance-reservations.service.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/completions/shared/billing.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/completions/shared/finalize.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/web-search/billing-guards.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/web-search/operation-finalization.service.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/pricing/constants.ts`
- `/Users/daniil/Projects/GonkaGate/pricing-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go`
- `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/identity-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml`

## Lane Execution

| Lane | Role | Status | Reconciled Output |
| --- | --- | --- | --- |
| R1 | `architecture-agent` | complete | Current `gonka-proxy` still owns local balance reservation/deduction, has a target TypeBox internal-money billing contract, and also has an older live bridge whose routes do not match the billing-service OpenAPI. Record as current-state inventory. |
| R2 | `domain-agent` | complete | Pure fire-and-forget usage events cannot enforce no-negative normal paid usage. Reserve-before-execution, billing-issued spend rights, and hybrid reserve plus async terminal settlement remain domain-viable. |
| R3 | `distributed-agent` | complete | Redpanda can carry commands/facts/replay only. Durable inbox/outbox, business idempotency, DB-before-offset commit, and Postgres-owned reconciliation are required for any event slice. |
| R4 | `data-agent` | complete | Current money-core schema has useful balance, idempotency, hold, ledger, terminal, and reconciliation primitives, but no event inbox/outbox or spend-token/lease tables. |
| R5 | `api-agent` | complete | Billing-service has no current business money API. Specification must choose between compatibility bridge, target internal-money route set, or a new OpenAPI shape, and must settle status/error/idempotency/readback semantics. |
| R6 | `performance-agent` | complete | Fully async is fastest but unsafe without prior preauthorization. Hybrid reserve-before-execution with async terminal settlement is the strongest current performance frontier unless reserve benchmarks force spend-token/window complexity. |
| R7 | `reliability-agent` | complete | Paid admission must fail closed when billing/Postgres/reserve authority is unavailable. Terminal facts after execution need either sync fallback, durable proxy outbox, or billing reconciliation to avoid stranded holds. |
| R8 | `security-agent` | complete | New money APIs/events need explicit service auth, route scopes, producer authenticity, account-scope binding, privacy-safe payloads, route-template logging, and no raw body/credential telemetry. |
| R9 | `qa-agent` | complete | Later `test-plan.md` must be layered: money math, Postgres invariants, API contracts, event replay, concurrency, privacy, outage behavior, and proxy performance. Exact budgets remain a spec/design input. |

No lane was expanded, merged, or skipped. All lanes were read-only and advisory.

## Fan-In Synthesis

Consensus findings:
- Billing-service Postgres must remain the customer-money authority for balance, holds, ledger effects, stored outcomes, and reconciliation.
- Redpanda cannot be the no-negative correctness boundary. It can transport reserve commands only if the proxy waits for a durable billing outcome before execution, or transport terminal facts after a durable reserve already exists.
- Fully async fire-and-forget usage settlement is rejected as the normal paid-usage architecture unless another prior billing-owned reservation, spend token, or account lease already bounded exposure.
- The strongest candidate for specification is a hybrid: synchronous or equivalently blocking billing-owned reserve before execution, durable terminal finalization/write-off submission after execution, inbox/outbox replay for async terminal settlement, and Postgres-owned stale-hold reconciliation.
- Spend tokens, cached allowances, and account leases are viable only as billing-minted, bounded, expiring, account-scoped rights whose full exposure is already reflected in billing's reserved balance. They are complexity to justify with performance evidence, not the default correctness store.
- Current provider/consumer drift is real: billing-service OpenAPI has only system/sample endpoints, proxy has both TypeBox internal-money contracts and older bridge paths, and pricing selector evidence has USD/USDT drift.

Specification must decide:
- Whether the target route contract is the existing proxy internal-money TypeBox model, the older shared-balance bridge, a new billing OpenAPI surface, or a planned compatibility bridge with explicit exit criteria.
- Where account scope is resolved, how API-key spend-limit outcomes relate to final money checks, and which service owns money-backed spend aggregates.
- Exact reserve/finalize/write-off/readback status and error semantics, including idempotency conflict, ambiguous timeout, `not_ready`, and deadline behavior.
- Whether Redpanda is used for terminal usage events, reserve commands with proxy wait, billing-emitted facts, or only later top-up/payment evidence.
- Required event identity, partition key, envelope authentication, inbox dedupe, outbox publish, DLQ/quarantine, retention, replay, and reconciliation rules.
- Performance budgets for reserve-before-upstream, terminal settlement, same-account contention, proxy first-token impact, stale-hold lag, and acceptable spend-window exposure if windows remain in scope.
- Privacy rules for payloads, stored safe metadata, route/path logging, qualified inference evidence locators, and bridge logging.

## Research Blockers And Reopen Targets

Research blockers:
- None. Research is complete enough to start specification.

Accepted research limits:
- No live deployment, production DB, or traffic inspection was performed; evidence is current local repository and sibling repository source/contracts.
- Historical `specs/billing-money-core/` material was used only as pattern/context evidence and is not active authority.
- No exact performance thresholds are approved yet; specification or later design must set them.
- No active payment business OpenAPI exists, so payments/top-up evidence remains future-compatible context unless the specification deliberately adopts the proxy internal-money payments contract.

Reopen research if:
- Specification cannot choose a contract or architecture because it lacks current provider-contract evidence from a sibling repository or generated/live contract endpoint.
- A later phase requires live deployment evidence, production latency distributions, or current DB rows that were outside this static research phase.
- A sibling provider contract changes materially before specification approval.

## Completion Marker

Complete because:
- R1-R9 lane outputs were received and reconciled.
- `../research/current-provider-consumer-evidence.md` and `../research/fan-in-synthesis.md` preserve compact findings.
- `../workflow-plan.md` was updated to mark research complete and route to specification.
- No later-phase artifacts or runtime files were created or modified.

## Stop Rule

Stop after research. Do not write `spec.md`, design files, `tasks.md`, `test-plan.md`, `rollout.md`, migrations, generated SQL, runtime adapters, tests, or implementation code in this session.

## Next Action

Start the specification phase in the next session. Read the workflow authority, this research record, and the two research notes first; then draft `spec.md` only, run the required formal spec clarification challenge for protected-domain work, update workflow state, and stop before technical design.
