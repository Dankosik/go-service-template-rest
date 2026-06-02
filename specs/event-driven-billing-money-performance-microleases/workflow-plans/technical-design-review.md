# Durable Microlease Technical Design Review

Phase: technical design review
Status: complete with PASS
Latest verdict: PASS
Reopen target: none for planning entry
Owner: orchestrator
Review type: local read-only technical design review with `go-design-review` stance and targeted money, data, distributed, security, reliability, performance, QA, and delivery lenses
Date: 2026-06-01
Reviewed packet: durable billing-issued performance microlease packet handed off by `workflow-plans/technical-design.md`

## Review Record Structure

This file is the mandatory technical design review record for
`specs/event-driven-billing-money-performance-microleases`.

The existing `specs/event-driven-billing-money-architecture` packet remains
read-only context. This review does not edit, supersede, or reopen that
workflow. This review decides only whether this microlease packet can enter
planning without asking planning or implementation to invent ownership,
contracts, data shape, sequence/failure behavior, rollout policy, or validation
proof.

## Reviewed Packet

Workflow authority:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-design-review/SKILL.md`

Repository and product baseline:

- `docs/repo-architecture.md`
- `docs/critical-billing-context.md`
- `docs/PRD.md`
- `docs/build-test-and-development-commands.md`

Task-local workflow and decision record:

- `workflow-plan.md`
- `workflow-plans/research.md`
- `workflow-plans/specification.md`
- `workflow-plans/technical-design.md`
- `spec.md`

Task-local design packet:

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

Preserved research:

- `research/source-notes.md`
- `research/pattern-catalog.md`
- `research/architecture-options-matrix.md`
- `research/risk-control-matrix.md`
- `research/fan-in-synthesis.md`

Read-only prior lease-packet context:

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

## Review Scope

Reviewed for planning-entry readiness only:

- source-of-truth ownership and authority classes;
- dependency direction, package boundaries, and worker lifecycle;
- protected HTTP contract authority, route security, readback, status mapping,
  and current runtime OpenAPI state;
- Redpanda event contract authority, event identity, producer authenticity,
  inbox/outbox, offset discipline, and replay semantics;
- Postgres data shape, ledger effects, transaction boundaries, admission
  controls, retention, and migration shape;
- proxy-owned durable grant, child debit, terminal obligation, checkpoint, and
  close obligations;
- microlease cap, active exposure, TTL, cutoff, terminal deadline, stale lag,
  reconciliation, and fail-closed defaults;
- memory and Redis classification;
- rollout, rollback, failback, mixed-version, no-dual-writer, and no direct
  reserve fallback policy;
- validation and performance proof handoff;
- whether planning can draft `tasks.md` without adding new design decisions.

Out of scope:

- design repair;
- `tasks.md`;
- runtime code, migrations, schemas, generated artifacts, adapters, workers,
  tests, or validation commands;
- edits to `specs/event-driven-billing-money-architecture`;
- changing the approved `spec.md`.

Local-review rationale:

- This review is distinct from the completed technical-design phase and made no
  design edits.
- No delegated subagent fan-out was used because the active tool policy permits
  spawning agents only on explicit delegation requests, and the user requested
  this phase rather than delegated agent work. The review still applies the
  mandatory gate locally against the full packet and current contract evidence.

## Follow-Up Alternative Architecture Review

Status: complete.
Date: 2026-06-01.
Trigger: user explicitly requested subagent review after the PASS gate to
compare the approved durable child-debit architecture against a faster
memory/Redis hot-path spend plus periodic durable checkpoint design.

This follow-up did not reopen specification or technical design. It reviewed
whether any documentation should change before planning and whether the
approved packet should be replaced by:

- A. the approved design: billing reserves the microlease in Postgres, and
  proxy commits durable child debit plus terminal obligation before external
  paid execution;
- B. the proposed fast-checkpoint design: billing reserves the microlease, proxy
  spends in memory or Redis, and proxy later writes periodic durable
  checkpoint/batch summaries;
- C. a hybrid or third option.

Read-only subagent fan-out:

| Lane | Result |
| --- | --- |
| Performance | Prefer A with hot-path allocator engineering. B is faster only in theory until benchmarks prove the durable proxy allocator misses latency targets. |
| Distributed systems | Prefer A or a hybrid that keeps A's durable commit point. B moves the crash boundary after external execution and requires explicit platform write-off risk acceptance. |
| Domain invariants | Prefer A. B weakens prepaid fairness, low-balance behavior, and customer-charge explainability unless a nonzero leak budget is approved. |
| Data | Prefer A. Batch-only summaries are insufficient for per-request audit, idempotency, fingerprint conflict detection, and terminal obligation rebuild. |
| Reliability and delivery | Prefer A. Redis absence from the first target gives simpler outage behavior; Redis can be added later only as rebuildable limiter/cache/backpressure. |

Orchestrator decision:

- Keep the approved architecture for planning.
- Treat the best hybrid as A-plus: optional memory or Redis precheck, throttle,
  deny, or cache; optimized minimal durable proxy child debit and terminal
  obligation before external execution; terminal, checkpoint, and close facts
  batched asynchronously for settlement, release, and reconciliation.
- Do not adopt B under the current approved target because it changes the
  accepted unbacked exposure budget from `0 USD` to a nonzero platform
  write-off/reconciliation model.
- Reopen specification, not planning, if product/platform explicitly wants B.
  That reopen must name owner, nonzero cap, metrics, alerts, budget burn
  behavior, reconciliation treatment, rollback rules, low-balance strict
  disablement, Redis failure assumptions, and the rule that ambiguous
  uncheckpointed spend is never retroactively charged above durable authority.
- Benchmark the approved durable proxy allocator before using performance
  intuition as a reason to reopen. If A cannot meet the latency envelope, the
  reopen question is whether to accept a bounded platform-loss architecture,
  not whether planning may silently move authority into memory or Redis.

Gate impact: no change. The final gate remains `PASS`, planning remains ready,
and B remains a specification-reopen candidate only.

## Gate Summary

Verdict: PASS.

The packet is coherent enough for planning. It selects one target authority
model: billing-service PostgreSQL reserves the full USD parent microlease
exposure, `gonka-proxy` must commit durable child debit and terminal obligation
before paid external execution, memory is cache/precheck only, Redis is absent
from the first target, and terminal/checkpoint/close facts settle through
durable event/inbox/outbox mechanics.

Why this is not `CONCERNS`:

- The remaining risks are already named as planning-input proof obligations or
  specification reopen conditions. They do not require planning or
  implementation to choose a missing architecture, ownership model, runtime
  contract source, data model class, failure policy, rollout policy, or proof
  class.

Why this is not `FAIL`:

- No reviewed artifact requires memory-only or Redis-only spend authority,
  direct per-request billing reserve fallback, a second money authority, hidden
  HTTP worker loops, Redpanda reserve authority, payment/top-up scope, API-key
  policy ownership transfer, pricing ownership transfer, or weaker privacy and
  outage semantics.
- Current provider-contract evidence matches the design's classification: the
  billing OpenAPI is still system-only and must be extended; proxy TypeBox
  reserve/finalize/write-off contracts are compatibility inputs only; pricing
  owns snapshot evidence with a USD-compatible proof obligation; API-key
  attribution can return `spend_limit_check_required`; payments remains outside
  this microlease runtime scope.

## Findings

No `blocks_planning`, `reopens_design`, or `reopens_spec` findings.

| Finding | Classification | Resolution | Evidence |
| --- | --- | --- | --- |
| Billing-issued microlease authority | `record_only` | Planning may task one authority path: billing reserves full USD parent exposure in Postgres before proxy spends, and every paid execution requires a durable proxy child debit plus terminal obligation. | `spec.md:79`, `spec.md:85`, `spec.md:117`, `design/overview.md:16`, `design/overview.md:19`, `design/sequence.md:22`, `design/sequence.md:54` |
| Zero unbacked spend and memory/Redis limits | `record_only` | Planning must preserve the approved `0 USD` unbacked exposure budget. Process memory remains cache/precheck only, Redis is absent from the first target, and any later Redis spend-authority proposal reopens specification. | `spec.md:88`, `spec.md:148`, `design/overview.md:22`, `design/overview.md:24`, `design/ownership-map.md:107`, `design/ownership-map.md:109`, `design/dependency-graph.md:39` |
| Microlease budgets and fail-closed defaults | `proof_obligation` | Planning must carry the first-rollout cap, active exposure cap, TTL, cutoff, terminal deadline, stale-age, reconciliation-SLA, and fail-closed malformed-config rules into tasks, tests, config validation, and rollout gates. Raising first-rollout caps needs later review or recorded rollout risk acceptance. | `design/overview.md:75`, `design/overview.md:81`, `design/overview.md:86`, `design/overview.md:92`, `design/overview.md:96`, `rollout.md:70` |
| Billing data model and SQL authority | `proof_obligation` | Planning can task microlease tables, child settlement projection, checkpoints, inbox/outbox, admission controls, ledger effects, SQLC queries, constraints, retention, and privacy. It must keep billing-side child-cap fields as settlement/projection/release proof, not as proxy active-path spend authority. | `docs/repo-architecture.md:29`, `docs/repo-architecture.md:40`, `design/data-model.md:30`, `design/data-model.md:66`, `design/data-model.md:105`, `design/data-model.md:137`, `design/data-model.md:169`, `design/data-model.md:185` |
| Current billing OpenAPI gap | `proof_obligation` | Planning must add protected OpenAPI microlease/readback/close/operation routes with real security, 401/403 Problems, body identifier placement, idempotency, ambiguous-timeout readback, status mapping, and generated drift proof. The current runtime OpenAPI remains only public/system sample endpoints. | `docs/repo-architecture.md:21`, `docs/repo-architecture.md:92`, `api/openapi/service.yaml:11`, `api/openapi/service.yaml:17`, `design/contracts/protected-http.md:11`, `design/contracts/protected-http.md:25`, `design/contracts/protected-http.md:114` |
| No synchronous terminal mutation path | `record_only` | The target microlease path intentionally settles terminal facts through events, not protected HTTP finalize/write-off. If planning needs synchronous terminal mutation, reopen specification rather than adding a second mutation ingress. | `design/contracts/protected-http.md:36`, `design/sequence.md:91`, `design/contracts/redpanda-events.md:26`, `design/contracts/redpanda-events.md:188` |
| Redpanda event and worker semantics | `proof_obligation` | Planning must add proto source inputs, generated DTOs, producer authenticity, proxy terminal/checkpoint/close outbox, billing inbox, DB-before-offset discipline, quarantine/redrive, billing outbox, worker readiness, shutdown, bounded retry, and retention tasks. | `docs/repo-architecture.md:110`, `docs/repo-architecture.md:131`, `design/contracts/redpanda-events.md:7`, `design/contracts/redpanda-events.md:12`, `design/contracts/redpanda-events.md:167`, `design/dependency-graph.md:72`, `test-plan.md:98` |
| Proxy durable allocator and fallback disablement | `proof_obligation` | Planning must include cross-repo proxy proof or tasks for durable grant storage, single-writer child debit allocation, terminal obligation before execution, crash/restart recovery, local backlog policy, memory rebuild, no direct reserve fallback for migrated cohorts, and privacy-safe local rows. | `design/component-map.md:37`, `design/component-map.md:39`, `design/sequence.md:47`, `design/sequence.md:60`, `design/sequence.md:67`, `test-plan.md:67`, `rollout.md:81` |
| Pricing-service USD-compatible snapshot evidence | `proof_obligation` | Planning must verify or task proof that pricing-service can provide or attest USD-compatible immutable snapshot evidence for issue/replenish, child caps, and terminal settlement. The current GNK/USDT compatibility selector is a proof/reopen condition, not a planning blocker. | `docs/critical-billing-context.md:77`, `design/dependency-graph.md:68`, `/Users/daniil/Projects/GonkaGate/pricing-service/README.md:36`, `/Users/daniil/Projects/GonkaGate/pricing-service/README.md:123`, `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go:253` |
| API-key spend/account split | `record_only` | Current API-key evidence supports the packet's split: API-key-service owns lifecycle and attribution; callers/billing/proxy must perform final spend/account/usage checks when attribution says `spend_limit_check_required`. | `docs/PRD.md:43`, `docs/PRD.md:90`, `design/ownership-map.md:15`, `design/dependency-graph.md:69`, `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml:2157`, `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml:2267` |
| Payments/top-up runtime scope | `record_only` | The packet does not import payment-provider lifecycle, raw PSP evidence, top-up runtime flows, or payment reversal/refund scope. Current payments-service OpenAPI evidence remains template/system-only and does not force a payment-authority decision here. | `spec.md:66`, `docs/PRD.md:42`, `design/ownership-map.md:17`, `design/dependency-graph.md:70`, `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml:1` |
| Rollout and mixed-version safety | `proof_obligation` | Planning must carry default-closed controls, shadow/parity, cohort enablement, no dual writer, old proxy writer disablement, direct reserve fallback disablement, rollback/failback, operator gates, and mixed-version compatibility into executable tasks. | `rollout.md:9`, `rollout.md:24`, `rollout.md:42`, `rollout.md:68`, `rollout.md:108`, `rollout.md:143`, `rollout.md:156` |
| Validation proof breadth | `proof_obligation` | Planning must turn `test-plan.md` into taskable proof across money math, persistence, protected HTTP, proxy allocator, memory/Redis absence, events/workers, privacy/security, performance, and rollout. | `test-plan.md:10`, `test-plan.md:24`, `test-plan.md:48`, `test-plan.md:67`, `test-plan.md:82`, `test-plan.md:98`, `test-plan.md:113`, `test-plan.md:133`, `test-plan.md:159` |

## Planning-Input Obligations

Planning must carry these obligations into `tasks.md`, companion proof entries,
or task-ledger review/readiness:

1. Add protected billing OpenAPI microlease issue/replenish, readback, close,
   and operation readback tasks with real service auth, route scopes, 401/403
   Problems, body identifier placement, route labels, idempotency,
   ambiguous-timeout readback, status mapping, bounded Problems, and generated
   drift proof.
2. Add data tasks for `spending_microleases`, billing-side
   `microlease_child_debits` settlement projection, `microlease_checkpoints`,
   `billing_event_inbox`, `billing_outbox`, `billing_admission_controls`,
   microlease ledger effects, retention, privacy constraints, migrations, SQLC
   queries, repositories, and migration validation.
3. Add config and bootstrap tasks for fail-closed caps, active exposure cap,
   safety floor, TTL, debit cutoff, terminal deadline, stale-age gates,
   reconciliation SLA, dependency admission, readiness, and startup validation.
4. Add worker tasks for terminal/checkpoint/close consumers, inbox retry,
   outbox relay, stale microlease/debit reconciliation, admission-control
   renewal/closure, bounded retry/backoff, readiness probes, signal-aware
   shutdown, and low-cardinality telemetry.
5. Add event contract tasks under future `api/proto/events/v1/*.proto` inputs
   with generated DTOs, adapter mapping, schema lint/generate/drift checks,
   producer identity/authenticity, event fingerprinting, partition/retention,
   quarantine/redrive, and safe payload policy.
6. Add cross-repo `gonka-proxy` implementation tasks or explicit proof
   obligations for durable grant storage, child debit CAS or row-lock
   allocation, durable terminal obligation before execution, terminal/checkpoint
   outbox, crash/restart recovery, local backlog gates, memory cache rebuild,
   no memory-only/Redis-only spend, no direct reserve fallback for migrated
   cohorts, bridge exit, and privacy-safe local rows.
7. Add pricing-service verification that pricing can provide or attest
   USD-compatible immutable snapshot evidence. If false, stop and reopen
   specification before implementation readiness.
8. Add API-key attribution integration obligations that preserve API-key-service
   lifecycle/policy ownership while ensuring final spend/account/usage checks
   happen in the billing/proxy authority path.
9. Add rollout tasks for default-closed controls, inert expand, shadow/parity,
   conservative internal cohort enablement, gradual expansion gates, legacy path
   contraction, rollback/failback, no dual writer, no old proxy writer, and
   no direct reserve fallback for migrated cohorts.
10. Add privacy/security proof that APIs, events, logs, traces, metrics,
    inbox/outbox rows, audit rows, proxy local rows, reconciliation records,
    test fixtures, and workflow artifacts exclude raw prompts, completions, SSE
    chunks, bearer tokens, API keys, DSNs, payment secrets, raw provider
    payloads, raw event payloads, dynamic proof URLs, sensitive request bodies,
    and high-cardinality labels.
11. Add performance proof for active microlease admission with and without
    memory precheck, proxy durable child allocation, cold issue/replenishment,
    account contention, terminal ingestion, checkpoint/close cadence, stale
    reconciliation, and first-token impact against the budgets in
    `design/overview.md` and `test-plan.md`.

## Design Escalations

None.

Planning should reopen technical design if it cannot convert an obligation
above into executable tasks without choosing new package boundaries, contract
sources, schema classes, worker lifecycle, failure semantics, rollout policy, or
validation proof.

Planning should reopen specification if tasking requires memory-only or
Redis-only spend, direct per-request reserve fallback for migrated paid cohorts,
nonzero unbacked spend exposure, weaker billing PostgreSQL authority, weaker
proxy durable lineage, broader payment/top-up/account/pricing/API-key
authority, weaker outage/privacy policy, or performance success that depends on
moving money authority into memory or Redis.

## Residual Risks

Record-only for planning:

- No live traffic, production database rows, deployment evidence, or benchmark
  evidence was used in this review. Planning must treat benchmark and rollout
  proof as future validation gates, not current readiness evidence.
- Current billing runtime contracts are still system/sample only. The design
  correctly routes protected business money contracts to OpenAPI planning and
  implementation rather than treating design-only routes as runtime authority.
- Current pricing evidence includes a GNK/USDT compatibility selector while the
  money path requires USD-compatible immutable pricing evidence. This is already
  a named proof/reopen obligation, not a design blocker.
- Exact SQL enum names, topic names, proto package names, and generated DTO
  paths remain planning/implementation details under the selected authorities.

## Validation Commands

Read-only review phase. No tests, code generation, migrations, benchmark,
runtime validation, or implementation readiness checks were run.

Read-only evidence commands used:

```bash
rtk git status --short
rtk wc -l AGENTS.md docs/spec-first-workflow.md docs/repo-architecture.md docs/critical-billing-context.md docs/PRD.md specs/event-driven-billing-money-performance-microleases/workflow-plan.md specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design.md specs/event-driven-billing-money-performance-microleases/spec.md specs/event-driven-billing-money-performance-microleases/design/overview.md specs/event-driven-billing-money-performance-microleases/design/component-map.md specs/event-driven-billing-money-performance-microleases/design/sequence.md specs/event-driven-billing-money-performance-microleases/design/ownership-map.md specs/event-driven-billing-money-performance-microleases/design/data-model.md specs/event-driven-billing-money-performance-microleases/design/dependency-graph.md specs/event-driven-billing-money-performance-microleases/design/contracts/protected-http.md specs/event-driven-billing-money-performance-microleases/design/contracts/redpanda-events.md specs/event-driven-billing-money-performance-microleases/test-plan.md specs/event-driven-billing-money-performance-microleases/rollout.md specs/event-driven-billing-money-architecture/workflow-plan.md specs/event-driven-billing-money-architecture/spec.md specs/event-driven-billing-money-architecture/design/overview.md specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md
rtk sed -n '1,230p' AGENTS.md
rtk sed -n '1,760p' docs/spec-first-workflow.md
rtk sed -n '1,220p' docs/repo-architecture.md
rtk sed -n '1,260p' docs/critical-billing-context.md
rtk sed -n '1,320p' docs/PRD.md
rtk sed -n '1,340p' specs/event-driven-billing-money-performance-microleases/workflow-plan.md
rtk sed -n '1,230p' specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design.md
rtk sed -n '1,520p' specs/event-driven-billing-money-performance-microleases/spec.md
rtk sed -n '1,220p' specs/event-driven-billing-money-performance-microleases/design/overview.md
rtk sed -n '1,140p' specs/event-driven-billing-money-performance-microleases/design/component-map.md
rtk sed -n '1,230p' specs/event-driven-billing-money-performance-microleases/design/sequence.md
rtk sed -n '1,180p' specs/event-driven-billing-money-performance-microleases/design/ownership-map.md
rtk sed -n '1,320p' specs/event-driven-billing-money-performance-microleases/design/data-model.md
rtk sed -n '1,160p' specs/event-driven-billing-money-performance-microleases/design/dependency-graph.md
rtk sed -n '1,220p' specs/event-driven-billing-money-performance-microleases/design/contracts/protected-http.md
rtk sed -n '1,240p' specs/event-driven-billing-money-performance-microleases/design/contracts/redpanda-events.md
rtk sed -n '1,230p' specs/event-driven-billing-money-performance-microleases/test-plan.md
rtk sed -n '1,230p' specs/event-driven-billing-money-performance-microleases/rollout.md
rtk sed -n '1,260p' specs/event-driven-billing-money-performance-microleases/research/source-notes.md
rtk sed -n '1,260p' specs/event-driven-billing-money-performance-microleases/research/pattern-catalog.md
rtk sed -n '1,260p' specs/event-driven-billing-money-performance-microleases/research/architecture-options-matrix.md
rtk sed -n '1,260p' specs/event-driven-billing-money-performance-microleases/research/fan-in-synthesis.md
rtk sed -n '1,220p' specs/event-driven-billing-money-performance-microleases/research/risk-control-matrix.md
rtk sed -n '1,280p' specs/event-driven-billing-money-architecture/workflow-plan.md
rtk sed -n '1,920p' specs/event-driven-billing-money-architecture/spec.md
rtk sed -n '1,260p' specs/event-driven-billing-money-architecture/design/overview.md
rtk sed -n '1,380p' specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md
rtk rg -n "paths:|/api/v1/ping|/health|security:|billing|microlease|lease|reserve|final|write|usage|money|spending|Account|Balance" api/openapi/service.yaml
rtk rg -n "billing_accounts|account_balances|idempotency_records|operation_outcomes|usage_operations|usage_holds|usage_terminal_outcomes|ledger|reconciliation|inbox|outbox|microlease|lease" env/migrations/000003_billing_money_core.up.sql internal/infra/postgres/queries/billing_money_core.sql
rtk rg -n "routeContractId|reserve|final|write.?off|Idempotency|Pricing|snapshot|USD|deadline|contract|operation|Request|fingerprint|account" /Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts /Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts
rtk rg -n "spend_limit_check_required|spend|account|usage|billing|limit|policy|balance" /Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml
rtk rg -n "GNK|USDT|USD|snapshot|fingerprint|pricing|quote|market-rate|rate|selector|decision|policy|contract" /Users/daniil/Projects/GonkaGate/pricing-service/README.md /Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go
rtk rg -n "paths:|payment|topup|webhook|health|ping|security" /Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml
```

## Orchestrator Resolution

Gate status: PASS.

Planning may start from the approved `spec.md`, review-ready design packet,
`test-plan.md`, `rollout.md`, preserved research, read-only prior lease-packet
context, and the planning-input obligations above.

Do not start implementation, migrations, generated SQL, runtime adapters, tests,
runtime event schemas, generated artifacts, or code until planning creates
`tasks.md` and task-ledger review/readiness passes.

Reopen target: none for planning entry.

## Workflow State

Current phase: technical design review.
Phase status: complete with PASS.
Next phase: planning.
Planning status: ready to start.
`tasks.md` status: missing, expected next.
Reopen target: none for planning entry.
Required next gate: task-ledger review/readiness after `tasks.md` is drafted.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: planning for `specs/event-driven-billing-money-performance-microleases`.

Read first:
- `AGENTS.md` and `docs/spec-first-workflow.md` for phase boundaries, artifact rules, planning rules, task-ledger review/readiness, and stop rules.
- `docs/repo-architecture.md`, `docs/critical-billing-context.md`, `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for repository boundaries, money invariants, worker/contract seams, rollout constraints, and repo-owned validation entrypoints.
- `specs/event-driven-billing-money-performance-microleases/workflow-plan.md` for current workflow state and planning routing.
- `specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design-review.md` because it records the PASS gate and planning-input obligations.
- `specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design.md` because it records the completed design phase and review packet.
- `specs/event-driven-billing-money-performance-microleases/workflow-plans/specification.md` and `spec.md` because `spec.md` is the canonical decision record.
- `specs/event-driven-billing-money-performance-microleases/design/overview.md`, `component-map.md`, `sequence.md`, `ownership-map.md`, `data-model.md`, `dependency-graph.md`, `contracts/protected-http.md`, and `contracts/redpanda-events.md` because planning must task the reviewed design without inventing new architecture.
- `specs/event-driven-billing-money-performance-microleases/test-plan.md` and `rollout.md` because planning must carry their proof and rollout obligations into `tasks.md`.
- `specs/event-driven-billing-money-performance-microleases/research/source-notes.md`, `pattern-catalog.md`, `architecture-options-matrix.md`, `risk-control-matrix.md`, and `fan-in-synthesis.md` only as needed for preserved evidence.
- `specs/event-driven-billing-money-architecture/workflow-plan.md`, `spec.md`, `design/overview.md`, and `workflow-plans/technical-design-review.md` as read-only context only.

Objective:
Create and review `specs/event-driven-billing-money-performance-microleases/tasks.md` for the approved durable billing-issued microlease architecture. The ledger must cover the approved `spec.md`, reviewed design bundle, technical-design-review PASS planning obligations, `test-plan.md`, and `rollout.md`, with no open questions or implementation-time design decisions.

Constraints:
- Do not implement runtime code, migrations, generated SQL, runtime adapters, tests, runtime event schemas, generated artifacts, or code in the planning phase.
- Do not broaden scope into memory-only or Redis-only spend, direct per-request reserve fallback for migrated paid cohorts, nonzero unbacked spend exposure, weaker billing PostgreSQL authority, weaker proxy durable child debit and terminal obligation, broader payment/top-up/account/pricing/API-key authority, or weaker privacy/outage policy.
- If planning cannot task a proof obligation without choosing new architecture, reopen technical design. If tasking would require a spec-scope change, reopen specification.

Expected output:
- `specs/event-driven-billing-money-performance-microleases/tasks.md` with Goal-ready task ledger, dependencies, proof obligations, evidence slots, explicit constraints, and planned validation commands.
- Task-ledger review/readiness result: `PASS`, eligible `CONCERNS` with named proof obligations, or `FAIL` with reopen target.
- Updated workflow state and the next-session prompt. If readiness passes, the next prompt must be an implementation prompt composed with `codex-goal-prompt-composer`.

Stop rule:
Complete planning and task-ledger review/readiness only, then stop before implementation.
```
