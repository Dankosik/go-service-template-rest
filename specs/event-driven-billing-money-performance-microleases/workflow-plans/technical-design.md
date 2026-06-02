# Technical Design Phase Plan And Completion

Phase: technical design
Status: complete; ready for technical design review
Date: 2026-06-01
Owner: orchestrator
Parent workflow: `../workflow-plan.md`

## Scope

Create the review-ready technical design for durable billing-issued
microleases with zero unbacked spend exposure.

This phase consumes the approved `../spec.md`, preserved research evidence, and
read-only context from `../../event-driven-billing-money-architecture`.

This phase does not write `tasks.md`, migrations, runtime schemas, generated
artifacts, adapters, tests, implementation code, or edits to the existing
`event-driven-billing-money-architecture` packet.

## Allowed Writes Used

- `../design/overview.md`
- `../design/component-map.md`
- `../design/sequence.md`
- `../design/ownership-map.md`
- `../design/data-model.md`
- `../design/dependency-graph.md`
- `../design/contracts/protected-http.md`
- `../design/contracts/redpanda-events.md`
- `../test-plan.md`
- `../rollout.md`
- `../workflow-plan.md`
- `workflow-plans/technical-design.md`

No runtime files were edited.

## Inputs Used

Repository and product context:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `docs/critical-billing-context.md`
- `docs/PRD.md`
- `docs/build-test-and-development-commands.md`

Task-local workflow and decisions:

- `../workflow-plan.md`
- `workflow-plans/specification.md`
- `../spec.md`
- `../research/source-notes.md`
- `../research/pattern-catalog.md`
- `../research/architecture-options-matrix.md`
- `../research/risk-control-matrix.md`
- `../research/fan-in-synthesis.md`

Read-only existing lease packet context:

- `../../event-driven-billing-money-architecture/workflow-plan.md`
- `../../event-driven-billing-money-architecture/spec.md`
- `../../event-driven-billing-money-architecture/design/overview.md`
- `../../event-driven-billing-money-architecture/workflow-plans/technical-design-review.md`

Current local and sibling contract evidence checked read-only:

- `api/openapi/service.yaml`
- `env/migrations/000003_billing_money_core.up.sql`
- `internal/infra/postgres/queries/billing_money_core.sql`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/pricing-service/README.md`
- `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go`
- `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml`

Skills and references used:

- `.agents/skills/technical-design-session/SKILL.md`
- `.agents/skills/go-design-spec/SKILL.md`
- `technical-design-session/references/conditional-design-artifact-triggers.md`
- `go-design-spec/references/design-bundle-assembly.md`

## Design Result

Technical design status: complete and review-ready.

Selected design:

- billing issues small owner-fenced microleases through protected HTTP and
  reserves full USD exposure in PostgreSQL before proxy can spend;
- proxy must commit durable child debit and terminal obligation before external
  paid execution;
- memory is cache/precheck only and cannot authorize execution;
- Redis is absent from the first target runtime path; any later Redis use must
  be limiter/cache/backpressure only and requires design review;
- terminal, checkpoint, close, and billing facts use durable outbox/inbox/event
  mechanics;
- strict mode is cache bypass, smaller/shorter microlease issuance, durable
  child-debit-only admission, or fail closed, not direct per-request reserve
  fallback;
- `test-plan.md` and `rollout.md` are triggered and written because validation
  and rollout shape are planning-critical.

## Artifact Status

| Artifact | Status | Trigger / Notes |
| --- | --- | --- |
| `../design/overview.md` | review-ready | Entry point, selected approach, budgets, artifact index, lease-packet reconciliation, and review handoff. |
| `../design/component-map.md` | review-ready | Split artifact for billing/proxy components, generated surfaces, workers, Redis decision, and stable non-touches. |
| `../design/sequence.md` | review-ready | Split artifact for issue/replenish, child debit, terminal settlement, checkpoint/close, expiry, reconciliation, strict mode, and outages. |
| `../design/ownership-map.md` | review-ready | Split artifact for authority classes, source-of-truth ownership, dependency direction, generated-code authority, and explicit non-owners. |
| `../design/data-model.md` | triggered, review-ready | Persisted microlease state, child terminal projection, checkpoint/close evidence, inbox/outbox, cache contract, replay, migration, and retention. |
| `../design/dependency-graph.md` | triggered, review-ready | Package/runtime dependency shape, worker lifecycle, cross-service contract edges, and no Redis runtime dependency. |
| `../design/contracts/protected-http.md` | triggered, review-ready | Protected microlease issue/replenish/readback/close design. Runtime authority remains future OpenAPI edits. |
| `../design/contracts/redpanda-events.md` | triggered, review-ready | Terminal/checkpoint/close/billing fact event design. Runtime authority remains future proto inputs. |
| `../test-plan.md` | triggered, review-ready | Proof obligations are too broad for `tasks.md` alone. |
| `../rollout.md` | triggered, review-ready | Migration/cutover/failback choreography affects money correctness. |
| `../tasks.md` | missing, later expected | Blocked until technical design review passes with `PASS` or eligible `CONCERNS`. |

## Blockers And Reopen Conditions

Blockers:

- None for starting technical design review.

Accepted assumptions:

- Current evidence is static repository, sibling repositories, and preserved
  research. No live traffic, production DB, deployment, or benchmark evidence
  was used.
- Pricing-service can provide or attest USD-compatible immutable pricing
  snapshot evidence. If false, reopen specification.
- Proxy durable storage can meet the active-path performance envelope with
  child debit before execution. If false, reopen specification rather than move
  authority to memory or Redis.

Reopen specification if a later phase needs memory-only or Redis-only spend,
direct per-request reserve fallback for migrated paid cohorts, nonzero
unrecorded spend budget, weaker billing PostgreSQL authority, weaker proxy
durable lineage, broader payment/top-up/account/pricing/API-key authority, or
weaker privacy/outage policy.

Reopen technical design if review or planning cannot task package boundaries,
contracts, persisted state, worker lifecycle, proxy allocator obligations,
failure semantics, rollout gates, or validation proof without choosing a new
design.

## Phase Status

Current phase: technical design.
Phase status: complete.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: technical design review.

## Next Session Context Bundle

The technical-design-review session should read:

1. `AGENTS.md` and `docs/spec-first-workflow.md` for review gate rules and stop
   boundaries.
2. `docs/repo-architecture.md`, `docs/critical-billing-context.md`, and
   `docs/PRD.md` for repository boundaries, money invariants, fail-closed
   product constraints, and privacy constraints.
3. `specs/event-driven-billing-money-performance-microleases/workflow-plan.md`
   and `workflow-plans/technical-design.md` for current phase state and review
   packet.
4. `specs/event-driven-billing-money-performance-microleases/spec.md` because
   it is the approved decision record.
5. `specs/event-driven-billing-money-performance-microleases/design/overview.md`,
   `component-map.md`, `sequence.md`, `ownership-map.md`, `data-model.md`,
   `dependency-graph.md`, `contracts/protected-http.md`, and
   `contracts/redpanda-events.md`.
6. `specs/event-driven-billing-money-performance-microleases/test-plan.md` and
   `rollout.md`.
7. The preserved research bundle under
   `specs/event-driven-billing-money-performance-microleases/research/`.
8. `specs/event-driven-billing-money-architecture/workflow-plan.md`,
   `spec.md`, `design/overview.md`, and
   `workflow-plans/technical-design-review.md` as read-only context only.

## Stop Rule

Technical design complete. Stop before technical design review, planning,
`tasks.md`, implementation, validation, migrations, runtime schemas, generated
artifacts, adapters, or tests.
