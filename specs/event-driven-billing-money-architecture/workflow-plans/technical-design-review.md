# Event-Driven Billing Money Architecture Technical Design Review

Phase: technical design review
Status: fresh lease-packet review complete with PASS
Latest verdict: PASS
Reopen target: none for planning entry
Owner: orchestrator
Review type: local read-only technical design review with `go-design-review` and targeted domain/data/distributed/security/reliability/QA/delivery lenses
Date: 2026-06-01
Reviewed packet: repaired billing-issued spending lease packet handed off by `workflow-plans/technical-design.md`

## Review Record Structure

This file is the current technical design review record for the repaired
billing-issued spending lease packet.

The earlier `FAIL` and follow-up `PASS` for the superseded per-request reserve
packet are historical context only. They are not the active planning gate after
the reopened specification selected billing-issued account-scoped spending
leases.

## Reviewed Packet

Workflow authority:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-design-review/SKILL.md`
- targeted local review lenses from `.agents/skills/go-domain-invariant-review/SKILL.md`,
  `.agents/skills/go-db-cache-review/SKILL.md`,
  `.agents/skills/go-distributed-review/SKILL.md`,
  `.agents/skills/go-reliability-review/SKILL.md`,
  `.agents/skills/go-security-review/SKILL.md`,
  `.agents/skills/go-qa-review/SKILL.md`, and
  `.agents/skills/go-devops-review/SKILL.md`

Workflow and decision record:

- `workflow-plan.md`
- `workflow-plans/technical-design.md`
- `workflow-plans/specification.md`
- previous `workflow-plans/technical-design-review.md` as historical context
- `spec.md`

Repaired design packet:

- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `design/data-model.md`
- `design/dependency-graph.md`
- `design/contracts/protected-http.md`
- `design/contracts/redpanda-events.md`
- `test-plan.md`
- `rollout.md`

Repository and product baseline:

- `docs/repo-architecture.md`
- `docs/critical-billing-context.md`
- `docs/PRD.md`
- `docs/build-test-and-development-commands.md`

Current local and sibling contract evidence checked read-only:

- `api/openapi/service.yaml`
- `api/proto/service/v1/service.proto`
- `env/migrations/000003_billing_money_core.up.sql`
- `internal/infra/postgres/queries/billing_money_core.sql`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go`
- `/Users/daniil/Projects/GonkaGate/pricing-service/README.md`
- `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`
- `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml`

## Review Scope

Reviewed for planning-entry readiness only:

- source-of-truth ownership;
- dependency direction and package/component boundaries;
- protected HTTP contract direction and current runtime source authority;
- Redpanda event contract authority, generated surfaces, and replay semantics;
- Postgres data shape, transaction boundaries, inbox/outbox, and admission-control
  state;
- proxy-owned durable lease/debit allocation and terminal submission obligations;
- failure semantics for lease issuance, child debit allocation, terminal
  settlement, checkpoint/close, expiry, lag, reconciliation, and rollout;
- security, privacy, auth, producer authenticity, and route/log payload safety;
- validation and rollout handoff;
- whether planning can write executable `tasks.md` without inventing
  architecture, ownership, contracts, data shape, failure semantics, rollout
  policy, or validation proof.

Out of scope:

- implementation code;
- design repair;
- `tasks.md`;
- migrations;
- generated SQL or generated DTOs;
- runtime adapters, schemas, workers, tests, generated artifacts, and codegen;
- changing the approved specification.

Local-review rationale:

- This review is distinct from the technical-design repair session and made no
  design edits.
- No subagent fan-out was used because the user requested this phase directly,
  the current tool policy permits spawning only on explicit delegation requests,
  and the planning-readiness questions could be closed from the review packet
  plus current repository/sibling contract evidence.

## Gate Summary

Verdict: PASS.

The repaired packet is coherent enough for planning. It selects the current
architecture, ownership, contract authorities, data deltas, failure semantics,
rollout gates, and validation proof obligations needed for a task ledger.

Why this is not `CONCERNS`:

- The remaining risks are already named as planning-input proof obligations or
  specification reopen conditions. They do not require implementation to choose
  a missing architecture, ownership, contract, data, rollout, or validation
  policy.

Why this is not `FAIL`:

- The packet preserves the approved uniform billing-issued lease path and rejects
  direct per-request reserve fallback, branchy paid admission, pure cached
  balance authority, proxy-local mutable balance authority, and Redpanda reserve
  commands.
- Billing-service Postgres remains the customer-money correctness boundary.
- Proxy durable lease/debit state remains proof and recovery state, not visible
  balance truth.
- Terminal mutation for the target lease path is Redpanda-only through proxy
  durable terminal submission, billing durable inbox, Postgres ledger effects,
  outbox facts, and reconciliation.
- Protected HTTP is selected for lease issue/replenish/readback/close with
  billing OpenAPI as runtime contract authority.
- Runtime Redpanda schema authority is selected as protobuf inputs under
  `api/proto/events/v1/*.proto`, with generated DTOs adapter-owned.
- Admission backpressure is selected as billing-owned
  `billing_admission_controls` in Postgres, read by `cmd/service` during lease
  issue/replenish and renewed or closed by `cmd/billing-worker` or protected
  operator/admin authority.

## Findings

No `blocks_planning`, `reopens_design`, or `reopens_spec` findings.

| Finding | Classification | Resolution | Evidence |
| --- | --- | --- | --- |
| Uniform lease path and prohibited fallbacks | `record_only` | Planning may task one target path: billing mints reserved leases, proxy spends durable child debits, and no migrated paid cohort uses direct reserve fallback or pure cached balance. | `spec.md:19`, `spec.md:85`, `spec.md:116`, `design/overview.md:10`, `design/overview.md:196` |
| Billing Postgres money authority and data shape | `record_only` | Planning has concrete table/migration/query surfaces for leases, debit settlements, checkpoints, inbox/outbox, admission controls, lineage, constraints, migration sequence, retention, and privacy. | `spec.md:493`, `design/data-model.md:59`, `design/data-model.md:229`, `design/data-model.md:320` |
| Protected HTTP contract authority | `record_only` | Planning can task OpenAPI routes, auth scopes, idempotency, body-identifier placement, readback, status mapping, and Problem responses without selecting a new contract model. | `design/contracts/protected-http.md:15`, `design/contracts/protected-http.md:31`, `design/contracts/protected-http.md:69`, `design/contracts/protected-http.md:200` |
| Redpanda event contract authority | `record_only` | Planning can task protobuf event inputs, generated DTOs, adapter mapping, lint/generate/drift/compatibility checks, producer authenticity, replay, retention, and topic evolution. | `design/contracts/redpanda-events.md:35`, `design/contracts/redpanda-events.md:74`, `design/contracts/redpanda-events.md:121`, `design/contracts/redpanda-events.md:363` |
| Proxy durable allocator and terminal submission | `proof_obligation` | Planning must carry cross-repo proxy proof for durable lease grant storage, child debit CAS/row-lock allocation, terminal submission before execution, restart recovery, local terminal backlog policy, and no direct reserve fallback. This is taskable from the design and does not require a new billing design decision. | `design/component-map.md:109`, `design/sequence.md:70`, `design/sequence.md:90`, `test-plan.md:96` |
| Pricing-service USD-compatible immutable snapshot evidence | `proof_obligation` | Planning must verify or task proof that pricing-service can provide or attest USD-compatible immutable pricing snapshot evidence. If false, reopen specification before implementation readiness. | `spec.md:417`, `spec.md:439`, `design/overview.md:161`, `pricing-service/internal/infra/http/pricing_handlers.go:253`, `pricing-service/README.md:122` |
| API-key spend-limit split | `record_only` | Current API-key contract evidence supports the design split: API-key-service owns key lifecycle/policy attribution, while callers must perform final spend/account/usage checks when `spend_limit_check_required` appears. | `spec.md:391`, `design/ownership-map.md:22`, `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml:2157`, `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml:2267` |
| Payments/top-up authority | `record_only` | Payment/top-up runtime scope remains out; current payments OpenAPI evidence is template-level and does not force a payment-authority decision into this lease packet. | `spec.md:77`, `design/overview.md:27`, `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml:1` |

## Planning-Input Obligations

Planning must carry these into `tasks.md`, `test-plan.md`, `rollout.md`, or the
task-ledger review/readiness record:

1. Add protected OpenAPI lease/readback/close/reconciliation routes with real
   security, 401/403 Problems, body identifier placement, idempotency,
   ambiguous-timeout readback, status mapping, and generated drift proof.
2. Add spending lease, debit settlement, checkpoint, inbox/outbox,
   `billing_admission_controls`, lineage, enum, constraint, query, repository,
   migration, and retention tasks with SQLC and migration validation proof.
3. Add worker tasks for terminal/checkpoint consumers, inbox retry, outbox relay,
   stale lease/debit reconciliation, admission-control renewal/close, readiness,
   shutdown, and bounded retry/backoff.
4. Add protobuf event contract tasks under `api/proto/events/v1/*.proto`,
   generated DTO tasks, adapter mapping, lint/generate/drift/compatibility
   checks, synthetic fixture policy, producer authority checks, retention, and
   evolution rules.
5. Add cross-repo `gonka-proxy` tasks or explicit proof obligations for durable
   lease grant persistence, single-writer child debit allocation, terminal
   submission before external execution, crash recovery, local terminal backlog
   policy, direct-reserve fallback disablement, bridge exit, and privacy-safe
   evidence.
6. Add pricing-service USD-compatible immutable snapshot verification and a
   specification reopen stop if that evidence cannot be provided or attested.
7. Add API-key-service final spend/account/usage check integration obligations
   without moving API-key policy lifecycle or configuration authority into
   billing.
8. Add rollout tasks for default-closed admission controls, import/parity,
   shadow mode, cohort gates, no dual writer, old proxy writer disablement,
   direct-reserve fallback disablement, rollback/failback, and operational gates.
9. Add privacy/security proof that APIs, events, logs, traces, metrics,
   inbox/outbox rows, audit rows, proxy local rows, and reconciliation records do
   not carry raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
   payment secrets, raw event payloads, dynamic proof URLs, or full provider
   payloads.

## Design Escalations

None.

Planning should reopen technical design if it cannot map an obligation above
into executable tasking without choosing new architecture, ownership, contract,
data, failure, rollout, or validation policy.

Planning should reopen specification if tasking requires direct per-request
reserve fallback, risk-based branchy paid admission, pure cached balance
authority, process-local spend allowance, Redpanda reserve commands, payment or
top-up scope, changed pricing/account/API-key authority, weaker terminal
durability, weaker billing Postgres money authority, or weaker outage/privacy
policy.

## Residual Risks

Record-only for planning:

- Pricing-service currently has `GNK/USDT` selector evidence in the compatibility
  current-market-rate handler. The approved design treats this as a
  pre-implementation proof or specification-reopen condition, not a design
  blocker.
- Current billing runtime sources only include the system/sample HTTP surface,
  a ping protobuf service, and historical money-core migration/query inputs. The
  design correctly routes new business HTTP contracts, event protobufs,
  migrations, SQLC, workers, and adapters to planning and implementation.
- The current proxy TypeBox internal-money billing contract remains source
  evidence and compatibility input only; billing-service OpenAPI is the target
  HTTP authority.

## Validation Commands

Read-only review phase. No tests, code generation, migrations, or runtime checks
were run, and no implementation readiness is claimed.

Read-only evidence commands used:

```bash
rtk rg --files specs/event-driven-billing-money-architecture
rtk rg -n "Technical Design Review|technical design review|PASS|CONCERNS|FAIL|planning" docs/spec-first-workflow.md
rtk nl -ba specs/event-driven-billing-money-architecture/workflow-plan.md
rtk nl -ba specs/event-driven-billing-money-architecture/workflow-plans/technical-design.md
rtk nl -ba specs/event-driven-billing-money-architecture/workflow-plans/specification.md
rtk nl -ba specs/event-driven-billing-money-architecture/spec.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/overview.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/component-map.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/sequence.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/ownership-map.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/data-model.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/dependency-graph.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/contracts/protected-http.md
rtk nl -ba specs/event-driven-billing-money-architecture/design/contracts/redpanda-events.md
rtk nl -ba specs/event-driven-billing-money-architecture/test-plan.md
rtk nl -ba specs/event-driven-billing-money-architecture/rollout.md
rtk nl -ba docs/repo-architecture.md
rtk nl -ba docs/critical-billing-context.md
rtk nl -ba docs/PRD.md
rtk rg --files api/proto api/openapi env/migrations internal/infra/postgres/queries
rtk rg -n "spend_limit_check_required|spend|account|usage|final|limit|billing" /Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml
rtk rg -n "market-rate|GNK|USDT|USD|snapshot|pricing|quote|currency|rate" /Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go /Users/daniil/Projects/GonkaGate/pricing-service/README.md
rtk nl -ba /Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts
rtk nl -ba /Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts
rtk nl -ba /Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml
```

## Orchestrator Resolution

Gate status: PASS.

Planning may start from the approved `spec.md`, repaired design packet,
`test-plan.md`, `rollout.md`, and the planning-input obligations above.

Do not start implementation, migrations, generated SQL, runtime adapters, tests,
runtime event schemas, generated artifacts, or code until planning creates
`tasks.md` and task-ledger review/readiness passes.

Reopen target: none for planning entry.

## Workflow State

Current phase: technical design review.
Phase status: fresh lease-packet review complete with PASS.
Next phase: planning.
Planning status: ready to start.
`tasks.md` status: missing, expected next.
Reopen target: none for planning entry.
Required next gate: task-ledger review/readiness after `tasks.md` is drafted.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: planning for `specs/event-driven-billing-money-architecture`.

Read first:
- `AGENTS.md` and `docs/spec-first-workflow.md` for phase boundaries, artifact rules, planning rules, task-ledger review/readiness, and stop rules.
- `specs/event-driven-billing-money-architecture/workflow-plan.md` for current workflow state and planning routing.
- `specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md` because it records the fresh lease-packet PASS and planning-input obligations.
- `specs/event-driven-billing-money-architecture/workflow-plans/technical-design.md` because it records the completed lease design repair and review handoff.
- `specs/event-driven-billing-money-architecture/workflow-plans/specification.md` because it records the reopened specification pass and formal clarification fan-in.
- `specs/event-driven-billing-money-architecture/spec.md` because it is the canonical approved decision record.
- `specs/event-driven-billing-money-architecture/design/overview.md`, `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/protected-http.md`, and `design/contracts/redpanda-events.md` because planning must task the reviewed design without inventing new architecture.
- `specs/event-driven-billing-money-architecture/test-plan.md` and `rollout.md` because planning must carry their proof and rollout obligations into `tasks.md`.
- `docs/repo-architecture.md`, `docs/critical-billing-context.md`, `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for repository boundaries, money invariants, worker/contract seams, rollout constraints, and repo-owned validation entrypoints.

Objective:
Create and review `specs/event-driven-billing-money-architecture/tasks.md` for the approved billing-issued spending lease architecture. The ledger must cover the approved `spec.md`, repaired design bundle, fresh technical-design-review PASS planning obligations, `test-plan.md`, and `rollout.md`, with no open questions or implementation-time design decisions.

Constraints:
- Do not implement runtime code, migrations, generated SQL, runtime adapters, tests, runtime event schemas, generated artifacts, or code in the planning phase.
- Do not broaden scope into direct per-request reserve fallback, branchy paid admission, pure cached balance authority, process-local spend allowance, Redpanda reserve commands, payment/top-up flows, payment evidence ingestion, changed pricing/account/API-key authority, weaker terminal durability, weaker billing Postgres money authority, or weaker outage/privacy policy.
- If planning cannot task a proof obligation without choosing new architecture, reopen technical design. If tasking would require a spec-scope change, reopen specification.

Expected output:
- `specs/event-driven-billing-money-architecture/tasks.md` with Goal-ready task ledger, dependencies, proof obligations, evidence slots, and explicit constraints.
- Task-ledger review/readiness result: `PASS`, eligible `CONCERNS` with named proof obligations, or `FAIL` with reopen target.
- Updated workflow state and the next-session prompt. If readiness passes, the next prompt must be an implementation prompt composed with `codex-goal-prompt-composer`.

Stop rule:
Complete planning and task-ledger review/readiness only, then stop before implementation.
```
