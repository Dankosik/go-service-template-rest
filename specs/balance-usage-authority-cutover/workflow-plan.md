# Balance And Usage Authority Cutover Workflow Plan

Mode: full orchestrated
Status: planning complete
Current phase: planning
Phase status: complete
Owner: orchestrator
Date: 2026-06-02

## Objective

Make `billing-service` production-ready as the authoritative balance and paid
usage service for `gonka-proxy`, excluding top-ups and `payments-service`
integration.

The work must move durable balance and usage authority to `billing-service` for
migrated paid cohorts. `gonka-proxy` remains the public gateway and execution
coordinator, but paid work must not proceed for migrated cohorts unless
`billing-service` has durably accepted the required reservation, microlease, or
usage obligation defined by the approved specification and technical design.

This is a new task-local workflow. It must not overwrite the completed
`specs/event-driven-billing-money-performance-microleases` packet. That packet
is read-first context and a preserved authority constraint for the existing
durable billing-issued microlease architecture.

## Scope

In scope:

- account resolve API;
- balance read API;
- usage reserve, finalize, write-off, reversal, operation readback, and
  reconciliation/admin readback surfaces needed for proxy cutover;
- durable Postgres-backed balance, usage, idempotency, and stored outcomes;
- stale reservation, stale or ambiguous operation, microlease exposure, retry,
  recovery, and worker reconciliation paths;
- service-to-service auth or the repository-approved internal auth model;
- Redpanda topic ownership/config, worker deployment wiring, alerts, metrics,
  logs, runbook/readiness checks, rollout gates, and operator readbacks;
- real HTTP handler backing and real worker runtime wiring in
  `billing-service`;
- current `gonka-proxy` balance and usage path research, with either approved
  cross-repo edits or an exact proxy handoff prompt.

Out of scope:

- top-up lifecycle;
- payment-service integration;
- payment evidence, external payment provider flows, and customer deposit or
  credit application;
- pricing ownership transfer;
- API-key policy ownership transfer;
- Redis or memory-backed spend authority;
- direct per-request reserve fallback for migrated paid cohorts;
- any weaker privacy, outage, or money-authority policy than the approved
  microlease specification.

## Why Full Orchestrated

Full orchestration is required. This task touches every protected trigger that
blocks direct or lean execution:

- public/internal API contracts and generated bindings;
- persisted money and usage data, migrations, idempotency, retention, and
  reconciliation;
- auth, tenant/account attribution, service-to-service trust, and privacy;
- money, billing, quotas, reserves, balances, credits, and entitlements;
- background workers, retries, lifecycle, shutdown, Redpanda, and cross-request
  state;
- rollout, rollback, mixed-version behavior, and proxy cutover;
- cross-service boundaries with `gonka-proxy`, pricing, API-key/identity, and
  the already completed microlease architecture.

## Current State

Workflow-planning started from:

- `AGENTS.md`;
- `docs/spec-first-workflow.md`;
- `docs/PRD.md`;
- `docs/repo-architecture.md`;
- `docs/critical-billing-context.md`;
- `docs/build-test-and-development-commands.md`;
- `specs/event-driven-billing-money-performance-microleases/workflow-plan.md`;
- `specs/event-driven-billing-money-performance-microleases/spec.md`;
- `specs/event-driven-billing-money-performance-microleases/tasks.md`.

Current observations:

- `specs/event-driven-billing-money-performance-microleases/tasks.md` is
  complete through T018 with final validation evidence. It is preserved context,
  not the active ledger for this broader balance/usage authority cutover.
- The new active task directory is
  `specs/balance-usage-authority-cutover`.
- Research completed locally without subagent fan-out because the user did not
  explicitly authorize subagents in the research session.
- Research notes now exist under
  `specs/balance-usage-authority-cutover/research/`.
- `specs/balance-usage-authority-cutover/spec.md` is approved as of
  2026-06-02.
- Technical design completed on 2026-06-02 with a review-ready split bundle
  under `specs/balance-usage-authority-cutover/design/`.
- Technical design review completed on 2026-06-02 with gate status
  `CONCERNS`. Planning may start only if it carries the named proof
  obligations from
  `specs/balance-usage-authority-cutover/workflow-plans/technical-design-review.md`.
- Planning completed on 2026-06-02.
- `tasks.md`, `test-plan.md`, and `rollout.md` now exist for this cutover
  scope.
- Task-ledger review is `PASS`.
- Implementation readiness is `PASS`.
- No migrations, generated code, tests, source code, proxy files, or runtime
  validation work started in the planning phase.

## Research Mode

Research mode: local read-only research completed against the planned evidence
lanes.

Reason:

- the research phase had to verify current provider/consumer contracts before
  any API, data, or implementation decision;
- the evidence spans independent approval-risk domains: proxy money paths,
  billing runtime surface, API contracts, data authority, security/privacy,
  reliability/workers, rollout/delivery, performance, and QA proof;
- preserved research notes are needed before specification because this is
  protected money and cross-service cutover work.

Research lanes completed in this session:

| Lane | Role | Owned Question | Skill | Expected Output |
| --- | --- | --- | --- | --- |
| L1 | `explorer` or local orchestrator | What `billing-service` contract, migration, handler, worker, and config surfaces already exist after the completed microlease work, and what gaps remain for account resolve, balance read, and usage lifecycle authority? | `research-session` | Evidence-only current-state notes with file paths and open decision inputs. |
| L2 | `explorer` or local orchestrator | What are the current `gonka-proxy` balance, usage, reserve/finalize/write-off, fallback, and local money write paths that must map to billing-service? | `research-session` | Proxy path map with exact files/routes/tests and no edits. |
| L3 | `api-agent` | What internal REST resources, idempotency rules, status mappings, generated contracts, and compatibility constraints are required before spec approval? | `api-contract-designer-spec` | API decision options, risks, and contract evidence only. |
| L4 | `data-agent` | What source-of-truth, schema, SQLC, transaction, retention, and reconciliation questions must be decided for durable balance and usage authority? | `go-data-architect-spec` | Data decision options and blockers only. |
| L5 | `distributed-agent` | What saga, inbox/outbox, Redpanda, retry, replay, ordering, and ambiguous-outcome semantics are required across proxy and billing? | `go-distributed-architect-spec` | Distributed-flow options and must-decide-now risks only. |
| L6 | `security-agent` | What trust-boundary, service-auth, tenant/account attribution, privacy, and abuse controls affect scope and validation? | `go-security-spec` | Security constraints, blockers, and proof obligations only. |
| L7 | `reliability-agent` | What timeout, fail-closed, worker lifecycle, readiness, rollback, and degraded-mode behavior must be specified? | `go-reliability-spec` | Reliability decisions and reopen triggers only. |
| L8 | `qa-agent` | What unit, integration, contract, worker, replay, privacy, performance, and cross-repo proof obligations must be carried into specification and planning? | `go-qa-tester-spec` | Test strategy inputs and validation risks only. |
| L9 | `performance-agent` | What hot-path budgets and benchmark proof are required for proxy paid-usage cutover without unbacked memory or Redis spend authority? | `go-performance-spec` | Performance budgets and measurement obligations only. |

Scoped-down rationale: subagent fan-out was not run because the user did not
explicitly authorize subagents. The orchestrator performed local read-only
research against the same L1-L9 lane questions and recorded evidence notes plus
fan-in synthesis before specification.

## Artifact Status

| Artifact | Status | Rationale |
| --- | --- | --- |
| `workflow-plan.md` | complete | This file owns cross-phase control for the new cutover scope and now routes to implementation. |
| `workflow-plans/workflow-planning.md` | complete | Phase-local routing, lane table, self-check, and stop rule are recorded. |
| `workflow-plans/research.md` | complete | Research mode, lane completion, fan-in, evidence limits, and stop rule are recorded. |
| `research/*.md` | complete | Local research notes preserve current billing/proxy/provider evidence used by specification. |
| `spec.md` | approved | Canonical decision record for account resolve, balance read, usage lifecycle, reconciliation, auth, worker/runtime ownership, proxy cutover, fail-closed behavior, and validation obligations. |
| `workflow-plans/specification.md` | not expected | The user-requested output for this phase was limited to `spec.md` plus this master workflow update; formal clarification status is recorded in both files. |
| `workflow-plans/technical-design.md` | complete | Phase-local technical-design state, artifact statuses, stop rule, and review handoff are recorded. |
| `design/overview.md` | review-ready | Entry point, chosen approach, live-fork decisions, artifact index, and review readiness. |
| `design/component-map.md` | review-ready | Billing-service and proxy component responsibilities, changed surfaces, stable surfaces, and intentional non-touches. |
| `design/sequence.md` | review-ready | Account import, resolve, balance, migrated paid admission, usage convergence, terminal, readback, worker, rollback, and failure behavior. |
| `design/ownership-map.md` | review-ready | Source-of-truth, dependency direction, generated-code authority, auth ownership, runtime ownership, proxy ownership, and explicit non-owners. |
| `design/data-model.md` | review-ready | Triggered by account import/parity, balance readbacks, usage operation lineage, microlease child debits, inbox/outbox, and reconciliation state. |
| `design/dependency-graph.md` | review-ready | Triggered by new app services, generated contract flow, repository boundaries, worker adapters, and proxy adapter coupling. |
| `design/contracts/http-api.md` | review-ready | Triggered by new internal account, balance, usage, operation, reconciliation, admin, and auth-scope REST contract surfaces. |
| `design/contracts/events.md` | review-ready | Triggered by Redpanda terminal, checkpoint, close, and billing-fact event ownership. |
| `design/worker-runtime.md` | review-ready | Triggered by the current no-op billing-worker bootstrap and required runtime ownership. |
| `design/rollout-validation-inputs.md` | review-ready | Triggered by mixed-mode proxy cutover, failback constraints, cross-repo proof, layered validation, and future rollout/test-plan needs. |
| `workflow-plans/technical-design-review.md` | complete, gate `CONCERNS` | Mandatory technical-design-review gate completed. Planning may start only with the named proof obligations carried forward. |
| `tasks.md` | approved for implementation | Goal-ready dependency-ordered ledger with task-ledger review `PASS`, implementation readiness `PASS`, TDR-C01 through TDR-C05 mapping, proof obligations, evidence fields, and reopen targets. |
| `test-plan.md` | approved for implementation | Triggered by money/data/API/worker/replay/performance proof complexity and created during planning. |
| `rollout.md` | approved for implementation | Triggered by proxy cutover, rollback, mixed-version, and cohort gating and created during planning. |
| review phase files | conditional, trigger unknown | Planning may create named review files only if a later multi-session review phase is required. |
| validation phase files | conditional, trigger unknown | Planning may create named validation files only if the approved ledger requires separate validation routing. |

## Phase Workflow Plans

- `workflow-plans/workflow-planning.md`: complete; owns this session's routing,
  planned lanes, adequacy self-check, completion marker, and stop rule.
- `workflow-plans/research.md`: complete; owns the research session mode,
  scoped-down rationale, lane outputs, fan-in summary, evidence limits,
  completion marker, and stop rule.
- `workflow-plans/technical-design.md`: complete; owns the technical-design
  pass type, artifact statuses, blockers, completion marker, stop rule, and
  technical-design-review handoff.
- `workflow-plans/technical-design-review.md`: complete; owns the review
  packet, scoped-down local-review rationale, findings, gate status
  `CONCERNS`, planning-input obligations, and stop rule.

No specification phase-local workflow plan was created because this session's
requested outputs were `spec.md` and this master workflow update only.

No planning, validation, or rollout phase-local workflow plans have been
created for this task. Planning kept routing in this master workflow file
because `tasks.md`, `test-plan.md`, and `rollout.md` are sufficient for the
implementation handoff.

## Adequacy Challenge

Status: local self-check PASS.

Formal read-only `workflow-plan-adequacy-challenge` was not spawned because the
available multi-agent tool requires explicit user authorization for subagents.
The self-check reviewed the master and phase-local workflow files for:

- execution shape and protected-domain triggers;
- research mode and lane ownership;
- artifact expectations;
- boundary compliance;
- next-session handoff;
- preservation of the completed microlease packet as context, not an overwrite
  target.

Blocking findings: none.

Non-blocking concern from workflow-planning is closed: research recorded a
local-research scoped-down rationale because subagent fan-out was not explicitly
authorized.

## Specification Clarification Gate

Status: complete.

Formal `spec-clarification-challenge` was required because this is
full-orchestrated protected-money work touching internal API contracts,
persisted state, service auth, worker/runtime behavior, performance,
validation, rollout, and cross-repo proxy cutover.

Method: local read-only formal clarification using the
`spec-clarification-challenge` rubric.

Scoped-down rationale: the available multi-agent tool says `spawn_agent` may be
used only when the user explicitly asks for subagents, delegation, or parallel
agent work. The user requested a specification-only phase and did not authorize
subagents, so the orchestrator ran the formal clarification locally across the
default five lenses.

Lanes: local orchestrator using `spec-clarification-challenge`.

Lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Resolution: PASS for specification approval. The reconciled outcomes are stored
in `spec.md` under `Formal Clarification Gate Reconciliation`; no unresolved
approval blocker remains.

Reopen specification if technical design requires direct per-request reserve
fallback, proxy-local money writes for migrated cohorts, non-JWT bearer-key
production auth, top-up/payment ownership, organization charging,
Redis/memory spend authority, weaker privacy policy, or a worker/runtime shape
that cannot fail closed.

## Technical Design Review Gate

Status: complete.

Review record:
`specs/balance-usage-authority-cutover/workflow-plans/technical-design-review.md`.

Gate status: `CONCERNS`.

Planning may start only if it carries these named proof obligations forward:

1. TDR-C01: import/parity readback path must be explicit before coding. Either
   prove existing `legacy_import_batches` and `legacy_balance_imports` support
   latest accepted import/parity readback, or task a narrow derived projection
   migration and SQLC/repository proof.
2. TDR-C02: generic usage readback identity must align with durable
   `usage_operation_id` linkage. Either make the migrated usage-operation link
   mandatory for the generated contract path, or constrain the generated
   contract so it does not promise identities that are not durably linked.
3. TDR-C03: broader authority runtime config must be task-owned, including
   default-disabled keys, env docs, validation, readiness, concrete worker
   tasks, and Redis-not-authority checks.
4. TDR-C04: operation readback scope repair must be a contract task covering
   OpenAPI, middleware constants, route-scope tests, and proxy caller scopes for
   `billing.operations.read`.
5. TDR-C05: Redpanda billing-fact topic/config/safe-label consistency must be
   proven across config defaults, adapters, outbox relay, fixtures, and metrics
   labels.

No `blocks_planning`, `reopens_design`, or `reopens_spec` finding remains open.
No follow-up technical design review is required unless planning cannot close
the obligations above without changing the approved design or specification.

Reopen technical design if planning needs a new ownership, data, contract,
runtime, rollout, or validation policy to close TDR-C01 through TDR-C05.

Reopen specification if planning or implementation requires direct per-request
reserve fallback, proxy-local money writes for migrated cohorts, non-JWT
bearer-key production auth, top-up/payment ownership, organization charging,
Redis or memory spend authority, weaker privacy policy, or runtime behavior
that cannot fail closed.

## Planning Result

Status: complete.

Created artifacts:

- `specs/balance-usage-authority-cutover/tasks.md`
- `specs/balance-usage-authority-cutover/test-plan.md`
- `specs/balance-usage-authority-cutover/rollout.md`

Task-ledger review: `PASS`.

Implementation readiness: `PASS`.

TDR obligations:

- TDR-C01 is carried by `tasks.md` T005, T006, and T021 plus
  `test-plan.md` data/SQLC proof.
- TDR-C02 is carried by `tasks.md` T006, T007, T010, and T021 plus
  generated contract and repository proof.
- TDR-C03 is carried by `tasks.md` T004, T011, T013, T020, and T021 plus
  config/readiness proof.
- TDR-C04 is carried by `tasks.md` T002, T010, T016, and T022 plus
  service-auth and proxy caller-scope proof.
- TDR-C05 is carried by `tasks.md` T003, T012, T013, and T021 plus
  Redpanda topic/config/safe-label proof.

Planning self-check:

- No open questions, `TBD` decisions, unresolved design gates, hidden
  implementation-time architecture decisions, or uncarried TDR obligations were
  found in the planning artifacts.
- `test-plan.md` and `rollout.md` were created because validation complexity
  and mixed-mode proxy cutover were already triggered.
- Cross-repo `gonka-proxy` implementation tasks are included in `tasks.md`
  because full cutover completion cannot be claimed from billing-service-only
  proof. Proxy tasks must obey
  `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`.
- No separate review or validation phase-control file is expected; the
  approved ledger carries implementation proof, final validation, and closeout.

Blockers: none for starting implementation from the approved ledger.

## Routing State

Current phase: planning.

Phase status: complete.

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: implementation.

Next session context bundle:

- `AGENTS.md`: repository workflow contract and phase boundaries.
- `specs/balance-usage-authority-cutover/tasks.md`: approved implementation
  ledger and source of truth for execution.
- `specs/balance-usage-authority-cutover/spec.md`: canonical decisions and
  reopen conditions.
- `specs/balance-usage-authority-cutover/test-plan.md`: required proof
  classes and validation commands.
- `specs/balance-usage-authority-cutover/rollout.md`: rollout modes, gates,
  failback behavior, and operator readbacks.
- `specs/balance-usage-authority-cutover/design/`: reviewed technical design
  context for ownership, sequence, data, contracts, worker runtime, and rollout
  inputs.
- `specs/balance-usage-authority-cutover/workflow-plans/technical-design-review.md`:
  gate `CONCERNS`, findings, and planning-input obligations now carried into
  the ledger.
- `docs/build-test-and-development-commands.md`: repository validation
  entrypoints.
- `specs/event-driven-billing-money-performance-microleases/spec.md` and
  `specs/event-driven-billing-money-performance-microleases/tasks.md`:
  preserved completed microlease authority context, not active tasking.
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`: proxy repository
  constraints for the cross-repo tasks in `tasks.md`.

Blockers: none for starting implementation from `tasks.md`.

Accepted assumptions:

- Top-ups and payment-service integration remain out of scope unless a later
  specification explicitly reopens scope.
- The completed microlease architecture remains preserved as the migrated paid
  admission authority model.
- Current proxy user IDs are stable enough for
  `account_scope_key=user:<proxy_user_id>` during the first cutover.
- Pricing-service can provide USD-compatible immutable pricing snapshot
  identity, fingerprint, policy version, decision time, and selector/use-class
  context before money lineage.
- API-key-service policy/readback outcomes do not replace final billing/proxy
  spend/account/usage checks for migrated paid admission.

Implementation obligations:

- Execute every required task in `tasks.md` from T001 through T025.
- Update task checkboxes and `Evidence` lines as proof runs.
- Keep `test-plan.md` and `rollout.md` as proof and rollout authorities; do
  not create new workflow/process artifacts after implementation starts unless
  the ledger explicitly reopens planning.
- Obey `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md` for proxy
  tasks, including the restriction on `npm run build`, `npm run test`,
  `npm run check`, and `node scripts/check-fastify-hooks.mjs` without explicit
  user approval.
- Do not claim cutover completion until both billing-service and proxy proof in
  `tasks.md` is fresh.

## Phased Delivery Policy

Later phases must stop at their own boundaries:

1. Research records current-state evidence and stops with a specification
   prompt.
2. Specification writes/approves `spec.md` and completes the required
   clarification challenge before technical design. This is complete.
3. Technical design writes the triggered design bundle and stops before review.
4. Technical design review records PASS, CONCERNS, or FAIL before planning.
   This is complete with `CONCERNS`.
5. Planning writes `tasks.md`, creates required `test-plan.md` and
   `rollout.md` or reviewed alternatives, runs task-ledger review/readiness,
   and stops before implementation. This is complete with readiness `PASS`.
6. Implementation may execute the approved ledger only after `tasks.md` has
   readiness `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.

Implementation is authorized to start in the next session from the approved
`tasks.md` ledger.

## Recommended Next-Session Prompt

```text
Work in /Users/daniil/Projects/GonkaGate/billing-service.

First, set a Codex Goal for this session:
Complete the approved balance and usage authority cutover by executing every required task in `specs/balance-usage-authority-cutover/tasks.md` without stopping until all required tasks are checked, every task evidence line is current, required proof passes or records a concrete blocker, and ledger-owned closeout is current.

After the goal is set, execute every required task in `specs/balance-usage-authority-cutover/tasks.md` from start to finish. Start at T001, continue through T025 final validation and closeout, and do not redefine success around a smaller slice.

Implementation brief:

Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Read first:
- AGENTS.md
- specs/balance-usage-authority-cutover/tasks.md
- specs/balance-usage-authority-cutover/spec.md
- specs/balance-usage-authority-cutover/test-plan.md
- specs/balance-usage-authority-cutover/rollout.md
- specs/balance-usage-authority-cutover/workflow-plans/technical-design-review.md
- specs/balance-usage-authority-cutover/design/
- docs/build-test-and-development-commands.md
- specs/event-driven-billing-money-performance-microleases/spec.md
- specs/event-driven-billing-money-performance-microleases/tasks.md
- /Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md

Current state:
- Next phase: implementation.
- Task-ledger review: PASS.
- Implementation readiness: PASS.
- Start at: T001.

Preserve:
- Top-ups, payment-service integration, public OpenAI-compatible route ownership, pricing catalog ownership, API-key lifecycle ownership, and organization charging remain out of scope.
- Billing PostgreSQL is customer-money authority.
- Migrated paid admission is microlease-first with durable proxy child debit and terminal obligation before external execution.
- Direct per-request reserve fallback, proxy-local money writes for migrated cohorts, Redis/memory spend authority, weaker privacy policy, and runtime behavior that cannot fail closed are specification reopen conditions.
- TDR-C01 through TDR-C05 are accepted proof obligations and must remain represented in task proof.
- Proxy tasks are part of this ledger. Obey `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`, including not running `npm run build`, `npm run test`, `npm run check`, or `node scripts/check-fastify-hooks.mjs` without explicit user approval.

Proof:
- Run the task-specific proof in `tasks.md`.
- Use `test-plan.md` for required validation classes.
- Use `rollout.md` for rollout gates, failback proof, and operator-readback proof.
- Final validation must include the billing-service repository-owned proof in T024 and fresh proxy targeted proof from T022/T023.

Progress rule:
- After each checkpoint or task proof, update only the relevant checkbox and `Evidence` line in `tasks.md`.
- During closeout, update `spec.md` Validation/Outcome with privacy-safe evidence as required by T025.
- Do not update workflow-plan routing files during implementation unless the ledger explicitly reopens planning.

Blocked-stop rule:
- If implementation needs a new architecture, ownership, contract, data, runtime, rollout, or validation decision, stop and reopen technical design.
- If implementation requires direct per-request reserve fallback, proxy-local money writes for migrated cohorts, non-JWT bearer-key production auth, top-up/payment ownership, organization charging, Redis or memory spend authority, weaker privacy policy, or runtime behavior that cannot fail closed, stop and reopen specification.
- If proof order or task coverage is wrong, stop and reopen planning.
```
