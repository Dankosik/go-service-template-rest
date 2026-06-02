# Event-Driven Billing Money Performance Microleases Workflow Plan

Mode: full orchestrated
Status: planning complete with PASS; implementation ready
Current phase: planning
Phase status: complete with PASS
Owner: orchestrator
Date: 2026-06-01

## Objective

Research whether the next billing money specification should move from the
existing billing-issued spending lease design toward a more performance-oriented
prepaid admission design based on escrow, bounded counters, microleases,
token-bucket spend inside bounded lease authority, async terminal usage events,
bounded write-off/reconciliation risk, and strict fallback only for risky cases.

This research track is separate from
`specs/event-driven-billing-money-architecture`. The existing architecture
packet remains read-only context for this session.

## Why Full Orchestrated

Full orchestration is required because this question touches protected domains:

- customer money, prepaid balances, reservations, quotas, credits, leases,
  write-offs, reversals, and entitlements;
- persisted ledger state, idempotency, inbox/outbox, event retention,
  reconciliation, and readback;
- distributed admission, local/global counters, Redis, Redpanda, Kafka-style
  replay, terminal lag, crash recovery, and worker lifecycle;
- proxy request-path performance, fail-closed behavior, abuse handling, rollout,
  rollback, and mixed-version behavior;
- cross-service boundaries with `gonka-proxy`, pricing/account attribution,
  API-key policy, and future metering or analytics stores.

## Research Mode

Research mode: fan-out plus local orchestrator synthesis.

Why:

- the evidence spans performance, distributed consistency, business-money
  invariants, data ownership, and reliability controls;
- the prompt explicitly allowed read-only subagents;
- preserved notes are needed before the next specification phase because this
  is a protected-money architecture question.

Lane summary:

| Lane | Role | Status | Research use |
| --- | --- | --- | --- |
| L1 | performance-agent | complete | Token-bucket, Redis, sharding, batching, and latency tradeoffs. |
| L2 | distributed-agent | complete | Escrow, bounded counters, leases, fencing, broker EOS limits, inbox/outbox. |
| L3 | domain-agent | complete | Prepaid invariants, exposure caps, low-balance strict mode, abuse controls. |
| L4 | data-agent | complete | Postgres authority, Redis/ClickHouse/Redpanda/proxy rows as non-authoritative surfaces. |
| L5 | reliability-agent | closed without report after timeout | Optional lane. Covered locally from Redis, Kafka/Redpanda, outbox/inbox, Envoy, Stripe, Metronome, OpenMeter, and existing workflow evidence. |
| L6 | orchestrator local research | complete | External source notes, repository evidence, options/risk matrices, and fan-in. |

## Preserved Research Artifacts

| Artifact | Status | Purpose |
| --- | --- | --- |
| `workflow-plans/research.md` | complete | Phase-local lane plan, fan-in path, source coverage, and stop rule. |
| `research/source-notes.md` | complete | External and repository source notes with URLs and relevance summaries. |
| `research/pattern-catalog.md` | complete | Pattern vocabulary and fit for prepaid billing admission. |
| `research/architecture-options-matrix.md` | complete | Comparison of strict reserve, microlease, Redis, checkpoint, pure async, and hybrid variants. |
| `research/risk-control-matrix.md` | complete | Business, reliability, security, and rollout controls for microlease candidates. |
| `research/fan-in-synthesis.md` | complete | Orchestrator synthesis from local research and subagent lane outputs. |

## Research Summary

Research consensus:

- The closest known pattern is escrow or bounded-counter rights allocation:
  billing durably allocates bounded spend authority first, then a local actor can
  spend only within its assigned rights.
- Token buckets, local/global rate limiters, Redis/Lua, sharded counters, Kafka,
  Redpanda, ClickHouse, Stripe Meter Events, Metronome, and OpenMeter all help
  with performance, metering, replay, aggregation, or abuse control. None of
  them replaces durable prepaid money authority.
- Redis and in-memory buckets are acceptable only as limiters/caches over
  already-reserved lease capacity, or as explicitly bounded platform write-off
  exposure if the future spec accepts that risk.
- Pure async metering is strong for terminal usage ingestion and invoices, but
  not sufficient for prepaid admission before external cost unless prior
  billing-owned authority already exists.
- Durable proxy checkpoint batches are useful release/proof inputs, not the
  first paid-admission authority.
- The next specification should compare a small set of candidates, with the
  strongest candidate being billing-issued microleases with tiny bounded
  exposure, durable proxy allocator/child-debit lineage, optional in-memory
  hot-path cache, async terminal facts, strict low-balance/risky modes, and
  Postgres-owned reconciliation.

These research conclusions have now been reconciled into approved
specification decisions in `spec.md`.

## Specification Completion

Specification status: approved.

Specification session boundary: reached previously.

Specification handoff target: technical design. That handoff has now been
consumed by the completed technical design phase below.

Specification result:

- accepted durable billing-issued microleases as escrowed USD spend rights;
- accepted zero unbacked spend exposure for the target path;
- preserved billing-service PostgreSQL as customer-money authority;
- preserved durable proxy child debit lineage before external paid execution;
- limited memory and Redis to cache, limiter, projection, or backpressure over
  already-reserved authority;
- rejected memory-only or Redis-only spend authority without an explicit
  product-owned write-off budget and specification reopen;
- rejected direct per-request reserve fallback as a hidden or routine alternate
  path for migrated paid cohorts.

Formal clarification gate:

- Status: complete.
- Method: local read-only formal clarification using the
  `spec-clarification-challenge` rubric across the default five lenses.
- Scoped-down rationale: the available multi-agent spawn tool is restricted to
  explicit user authorization for subagents, and the user requested a
  specification-only phase without delegation.
- Resolution: no unresolved blocker for specification approval.

## Technical Design Completion

Technical design status: complete.

Design result:

- created a split `design/` bundle for durable billing-issued microleases;
- preserved billing PostgreSQL as customer-money authority and active exposure
  as visible reserved balance;
- kept durable proxy child debit and terminal obligation before external paid
  execution;
- mapped process memory to cache/precheck only;
- selected Redis absent from the first target runtime path, with any later Redis
  use limited to rebuildable limiter/cache/backpressure only;
- selected short-lived formula-bounded microlease budgets: first rollout
  per-microlease cap at or below 1.00 USD, per-account active exposure cap at
  or below 2.00 USD, TTL 30 seconds, debit cutoff 25 seconds, terminal deadline
  120 seconds, and reconciliation opening within 5 minutes of critical stale or
  lag breach;
- triggered and wrote `test-plan.md` and `rollout.md` because validation and
  rollout shape are planning-critical.

## Technical Design Review Completion

Technical design review status: complete with PASS.

Review result:

- created `workflow-plans/technical-design-review.md`;
- reviewed the approved `spec.md`, split design bundle, `test-plan.md`,
  `rollout.md`, preserved research, and read-only existing lease packet context;
- verified current local and sibling contract surfaces read-only:
  `api/openapi/service.yaml`,
  `env/migrations/000003_billing_money_core.up.sql`,
  `internal/infra/postgres/queries/billing_money_core.sql`,
  `gonka-proxy` internal-money TypeBox contracts,
  `pricing-service` README and pricing HTTP handler,
  `api-key-service` OpenAPI, and `payments-service` OpenAPI;
- found no `blocks_planning`, `reopens_design`, or `reopens_spec` findings;
- recorded planning-input proof obligations for protected OpenAPI routes,
  microlease schema/SQLC/migration work, config/bootstrap fail-closed budgets,
  Redpanda proto/event/worker work, cross-repo proxy durable allocator proof,
  pricing USD-compatible snapshot proof, API-key attribution integration,
  rollout choreography, privacy/security proof, and performance benchmarks.

Gate status: PASS.

Planning consumed this gate. Implementation is now controlled by the approved
`tasks.md` ledger and may start in a later session from T001.

## Follow-Up Alternative Architecture Review

Follow-up review status: complete.

Trigger:

- After the technical design review PASS, the user explicitly asked for a
  read-only subagent-backed comparison of the approved durable child-debit
  architecture against a faster memory/Redis hot-path spend plus periodic
  durable checkpoint/batch summary architecture.

Fan-out:

- performance-agent;
- distributed-agent;
- domain-agent;
- data-agent;
- reliability-agent.

Result:

- All lanes preferred the approved architecture or an A-compatible hybrid over
  memory/Redis spend with later checkpoint as the current production target.
- The accepted hybrid remains within the approved design: memory or Redis may
  precheck, throttle, deny, or cache; the paid-execution commit point remains a
  durable proxy child debit plus durable terminal obligation before external
  execution; terminal, checkpoint, and close facts may be batched
  asynchronously.
- The proposed memory/Redis hot-path spend plus periodic durable summary design
  is a valid future specification-reopen candidate only if product/platform
  explicitly accepts a nonzero bounded platform write-off/reconciliation budget.
- Performance intuition alone is not enough to reopen. Planning must preserve
  the benchmark proof for the approved proxy durable allocator first.

Gate impact:

- No change to the technical design review verdict.
- Gate status remains PASS.
- Planning remains ready.
- If planning or implementation needs memory-only or Redis-only spend before a
  durable child debit, reopen specification instead of treating it as a tasking
  detail.

## Blockers, Assumptions, And Open Points

Blockers:

- None for starting implementation from the approved `tasks.md`.

Accepted assumptions:

- Current evidence is static repository, sibling task artifacts, external docs,
  and papers. No live deployment, production DB, traffic distribution, or
  benchmark evidence was used.
- "Microlease" means bounded account-scoped USD spend authority minted by
  billing and consumed by proxy within a fenced owner/generation.
- Async terminal usage means terminal facts are durable and replayable, but
  they do not authorize first paid execution unless prior authority exists.
- Pricing-service can provide or attest USD-compatible immutable pricing
  snapshot evidence. If false, reopen specification.
- Proxy durable storage can meet the active-path performance envelope with
  child debit before execution. If false, reopen specification with an explicit
  bounded-loss decision instead of moving authority to memory or Redis during
  planning or implementation.

Open points for specification:

- None remaining for planning entry.

Open points for technical design:

- None remaining for planning entry.

Open points for planning:

- None. Planning created and reviewed `tasks.md`.

## Planning Completion

Planning status: complete with PASS.

Result:

- created `tasks.md`;
- created `workflow-plans/planning.md`;
- ran local task-ledger review/readiness;
- recorded workflow-control adequacy self-check.

Task-ledger review: PASS.

Implementation readiness: PASS.

Gate result: implementation may start in a later session from T001 in
`tasks.md`.

Accepted risks: none.

Proof obligations: all required proof obligations are task-owned in
`tasks.md`.

Workflow-plan adequacy:

- Result: PASS.
- Method: local read-only self-check using the
  `workflow-plan-adequacy-challenge` criteria.
- Subagent status: not spawned because the active multi-agent tool policy
  permits spawning only when the user explicitly asks for subagents,
  delegation, or parallel agent work.

## Artifact State

| Artifact | State | Trigger / Notes |
| --- | --- | --- |
| `workflow-plan.md` | updated | Master state routes from planning PASS to implementation. |
| `workflow-plans/research.md` | complete | Phase-local research control and lane fan-in. |
| `research/*.md` | complete | Preserved research evidence and synthesis. |
| `workflow-plans/specification.md` | complete | Phase-local specification state, clarification gate, and handoff. |
| `spec.md` | approved | Canonical decision record for durable zero-unbacked-exposure microleases. |
| Formal spec clarification challenge | complete | Local five-lens formal challenge; no unresolved blocker. |
| `workflow-plans/technical-design.md` | complete | Phase-local technical design state and handoff to review. |
| `workflow-plans/technical-design-review.md` | complete with PASS; follow-up alternative review recorded | Mandatory pre-planning gate; no planning blockers; carries planning-input obligations and the user-requested A/B/hybrid architecture fan-in. |
| `workflow-plans/planning.md` | complete with PASS | Phase-local planning state, task-ledger review/readiness, adequacy self-check, and implementation handoff. |
| `design/overview.md` | review-ready | Entry point, selected approach, budgets, artifact index, and lease-packet reconciliation. |
| `design/component-map.md` | review-ready | Components, generated surfaces, proxy durable allocator, Redis decision, workers, and stable non-touches. |
| `design/sequence.md` | review-ready | Issue/replenish, child debit, strict mode, terminal settlement, checkpoint/close, expiry, reconciliation, and outages. |
| `design/ownership-map.md` | review-ready | Source-of-truth ownership, authority classes, dependency direction, generated-code authority, and explicit non-owners. |
| `design/data-model.md` | triggered, review-ready | Persisted microlease state, child terminal projection, checkpoint/close evidence, inbox/outbox, cache contract, replay, migration, and retention. |
| `design/dependency-graph.md` | triggered, review-ready | Runtime/package dependency graph, worker lifecycle, cross-service contract edges, and no Redis runtime dependency. |
| `design/contracts/protected-http.md` | triggered, review-ready | Protected microlease issue/replenish/readback/close contract design. Runtime authority remains future OpenAPI edits. |
| `design/contracts/redpanda-events.md` | triggered, review-ready | Terminal/checkpoint/close/billing fact event design. Runtime authority remains future proto inputs. |
| `test-plan.md` | triggered, review-ready | Proof obligations are too broad for `tasks.md` alone. |
| `rollout.md` | triggered, review-ready | Migration/cutover/failback choreography affects money correctness. |
| `tasks.md` | approved | Goal-ready implementation ledger T001 through T018; task-ledger review and implementation readiness PASS. |
| `workflow-plans/review-phase-N.md` | not expected | No separate pre-code review phase file was needed; implementation proof is ledger-owned. |
| `workflow-plans/validation-phase-N.md` | not expected | No separate validation phase file was needed; final validation and closeout are T017 and T018. |

## Routing Status

Current phase: planning.
Phase status: complete with PASS.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: implementation from T001 in `tasks.md`.

## Next Session Context Bundle

The implementation session should read:

1. `AGENTS.md` and `docs/spec-first-workflow.md` for implementation boundary,
   ledger authority, progress/evidence, validation, and closeout rules.
2. `specs/event-driven-billing-money-performance-microleases/tasks.md` because
   it is the approved implementation ledger and source of truth.
3. `specs/event-driven-billing-money-performance-microleases/spec.md` because
   it is the canonical decision record.
4. `specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design-review.md`
   because it records the PASS gate, planning-input obligations, and follow-up
   architecture review.
5. `specs/event-driven-billing-money-performance-microleases/design/` because
   it defines component ownership, data, dependency, sequence, HTTP, and event
   design.
6. `specs/event-driven-billing-money-performance-microleases/test-plan.md` and
   `rollout.md` because the ledger carries their proof and rollout
   obligations.
7. `docs/repo-architecture.md`, `docs/critical-billing-context.md`,
   `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for
   repository boundaries, money invariants, privacy constraints, and validation
   commands.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

First, set a Codex Goal for this session:
Complete the approved durable billing-issued microlease architecture by executing every required task in `specs/event-driven-billing-money-performance-microleases/tasks.md` without stopping until all required tasks are checked, required proof passes or records a concrete blocker, and ledger-owned closeout evidence is current.

After the goal is set, execute every required task in `specs/event-driven-billing-money-performance-microleases/tasks.md` from start to finish. Start at T001, continue through T018 final validation and closeout, and do not redefine success around a smaller slice.

Implementation brief:

Read first:
- `AGENTS.md` and `docs/spec-first-workflow.md` for implementation boundaries and ledger-driven closeout.
- `specs/event-driven-billing-money-performance-microleases/tasks.md` because it is the approved implementation ledger.
- `specs/event-driven-billing-money-performance-microleases/spec.md` because it is the canonical decision record.
- `specs/event-driven-billing-money-performance-microleases/workflow-plans/technical-design-review.md` because it records the PASS gate, planning-input obligations, and follow-up architecture review.
- `specs/event-driven-billing-money-performance-microleases/design/` because it defines component ownership, data, dependency, sequence, HTTP, and event design.
- `specs/event-driven-billing-money-performance-microleases/test-plan.md` and `rollout.md` because the ledger carries their proof and rollout obligations.
- `docs/repo-architecture.md`, `docs/critical-billing-context.md`, `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for repository boundaries, money invariants, privacy constraints, and validation commands.

Current state:
- Next phase: implementation.
- Task-ledger review: PASS.
- Implementation readiness: PASS.
- Start at: T001.

Preserve:
- Billing PostgreSQL remains the only customer-money authority.
- Active microlease exposure subtracts from visible available balance.
- Proxy must commit durable child debit and terminal obligation before external paid execution.
- Redis is absent from the first target runtime path; memory is cache/precheck only.
- No direct per-request reserve fallback for migrated paid cohorts.
- No payment/top-up scope, pricing ownership transfer, API-key policy ownership transfer, weaker privacy policy, or weaker outage policy.

Proof:
- Use the proof commands and evidence rules in `tasks.md`, especially `rtk make openapi-check`, `rtk make sqlc-check`, migration validation, targeted integration tests, worker/event tests, proxy proof commands, performance benchmarks, `rtk make go-security`, `rtk make secret-scan`, and the final validation bundle in T017.

Progress rule:
- Update only `tasks.md` checkbox/evidence lines during implementation progress.
- During closeout, update `spec.md` `Validation`/`Outcome` only as allowed by T018.
- Do not update `workflow-plan.md` or `workflow-plans/*` during implementation unless a future approved ledger explicitly names those files.

Blocked-stop rule:
- If T001 proves pricing cannot provide or attest USD-compatible immutable pricing snapshot evidence, stop and reopen specification.
- If T016 shows the target cannot meet performance without unbacked memory/Redis spend, stop and reopen specification.
- If any task requires memory-only or Redis-only spend, direct reserve fallback, nonzero unbacked exposure, weaker billing authority, weaker proxy durable lineage, broader service ownership, or weaker privacy/outage policy, stop and reopen specification.
- If implementation needs a missing package boundary, contract detail, data shape, worker lifecycle, failure semantic, rollout gate, or validation policy, stop and reopen technical design.
- If only task coverage, order, proof wording, evidence fields, or handoff state are insufficient, stop and reopen planning.

Stop rule:
Execute the approved ledger through T018 and the named proof, then stop with ledger evidence and closeout current. Do not create new pre-code workflow artifacts during implementation.
```
