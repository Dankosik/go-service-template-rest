# Balance And Usage Authority Cutover Technical Design Review

Phase: technical-design-review
Status: complete
Gate status: CONCERNS
Owner: orchestrator
Date: 2026-06-02

## Reviewed Packet

Reviewed artifacts:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/build-test-and-development-commands.md`
- `specs/balance-usage-authority-cutover/workflow-plan.md`
- `specs/balance-usage-authority-cutover/spec.md`
- `specs/balance-usage-authority-cutover/workflow-plans/research.md`
- `specs/balance-usage-authority-cutover/workflow-plans/technical-design.md`
- `specs/balance-usage-authority-cutover/research/*.md`
- `specs/balance-usage-authority-cutover/design/*.md`
- `specs/balance-usage-authority-cutover/design/contracts/*.md`
- `specs/event-driven-billing-money-performance-microleases/spec.md`
- `specs/event-driven-billing-money-performance-microleases/tasks.md`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`
- Targeted current-source evidence in `api/openapi/service.yaml`,
  `internal/infra/http/service_auth.go`, `internal/infra/http/handlers.go`,
  `cmd/service/internal/bootstrap/run.go`,
  `cmd/billing-worker/internal/bootstrap/run.go`, migrations, SQL query
  sources, and selected current `gonka-proxy` money-path files.

Review scope:

- source-of-truth ownership;
- dependency direction;
- API/event contract coherence;
- data model adequacy;
- sequence and failure behavior;
- service-auth scope alignment;
- concrete HTTP app service wiring;
- billing-worker, Redpanda, inbox/outbox, and reconciliation runtime ownership;
- proxy cutover behavior;
- fail-closed behavior;
- rollout and validation inputs;
- accidental complexity and hidden implementation-time decisions.

## Method

This was a distinct local read-only review pass after technical design
completion. No code, migrations, generated files, tests, proxy files,
`spec.md`, design files, `tasks.md`, `test-plan.md`, or `rollout.md` were
changed.

Scoped-down rationale: no subagent fan-out was explicitly authorized for this
phase, and previous workflow state used local review when subagents were not
authorized. The review therefore stayed in the orchestrator flow, but remained
separate from the design-writing pass and used the technical-design-review gate
rules from `docs/spec-first-workflow.md`.

## Gate Decision

Gate status: CONCERNS.

Planning may start only if it carries the planning-input obligations below into
`tasks.md`, `test-plan.md`, `rollout.md`, or the task-ledger review/readiness
record as appropriate.

Why not PASS:

- The design packet is coherent, but several details are intentionally recorded
  as conditional planning inputs. They are acceptable only if planning turns
  them into explicit tasks or proof obligations before implementation starts.

Why not FAIL:

- No reviewed artifact leaves a live architecture, ownership, contract,
  runtime, rollout, or validation fork that planning must choose from scratch.
- The design preserves the approved microlease authority model, keeps
  billing-service Postgres as customer-money authority, rejects direct
  per-request reserve fallback for migrated proxy cohorts, rejects proxy-local
  migrated money writes, and names reopen conditions for any weaker design.

Reopen target: none now.

Follow-up review required: no, unless planning or implementation discovery
changes the design packet, requires a specification reopen condition, or cannot
close one of the planning-input obligations without inventing new design
policy.

## Findings

### TDR-C01 Import/parity readback projection must be decided before coding

Classification: proof_obligation

Evidence:

- `design/data-model.md:54` says schema expansion is triggered if planning
  cannot express the latest accepted import/parity state from
  `legacy_import_batches` and `legacy_balance_imports`.
- `design/component-map.md:86` similarly allows migrations for missing
  readback/import/runtime fields.
- Current migrations already define `legacy_import_batches` and
  `legacy_balance_imports` with state, parity, and account-scope indexes
  (`env/migrations/000003_billing_money_core.up.sql:624`,
  `env/migrations/000003_billing_money_core.up.sql:643`).

Issue:

The source of truth is clear, but the design leaves the exact read path as a
conditional implementation choice.

Impact:

If planning leaves this conditional, implementation could silently decide
whether account resolve uses direct indexed reads or introduces a derived
import-state projection, which is a schema/readback decision.

Resolution:

Planning must include an explicit pre-code proof or task: either prove the
latest accepted import/parity state is expressible with existing tables and
indexes, or add a narrow derived projection migration and SQLC/repository
tasks. The projection, if added, remains derived readback state and not a
second balance authority.

### TDR-C02 Usage-operation linkage for generic readback must be explicit

Classification: proof_obligation

Evidence:

- `design/data-model.md:95` says `microlease_child_debits.usage_operation_id`
  may link to `usage_operations` when generic usage readback identity is
  needed.
- `design/contracts/http-api.md:59` adds usage readback, and
  `design/contracts/http-api.md:74` promises operations readback by usage
  operation ID, microlease ID, child debit ID, terminal outcome ID, and related
  identities.
- Current microlease schema has nullable
  `microlease_child_debits.usage_operation_id`
  (`env/migrations/000004_billing_microleases.up.sql:328`).

Issue:

The HTTP/readback design promises usage-operation lookup, while the data design
uses conditional language around the migrated child-debit to usage-operation
link.

Impact:

If unresolved, OpenAPI authoring or repository implementation could diverge:
one path might expose usage-operation lookup while another treats that linkage
as optional and therefore not always readbackable.

Resolution:

Planning must require one explicit contract/data choice before implementation:
for migrated generic usage reserve/readback, either persist the
`usage_operation_id` linkage as mandatory for the generated contract path, or
constrain the generated route so it does not promise lookup identities that are
not durably linked.

### TDR-C03 Runtime gate config names and docs must be owned by planning

Classification: proof_obligation

Evidence:

- `design/component-map.md:137` defers exact broader balance/usage authority
  runtime gate key names to implementation.
- `design/worker-runtime.md:34` requires enabled worker runtime to validate
  dependencies, and `design/worker-runtime.md:40` requires all seven concrete
  worker roles.
- Current source still has no-op worker task construction in
  `cmd/billing-worker/internal/bootstrap/run.go:46` and
  `cmd/billing-worker/internal/bootstrap/run.go:85`.

Issue:

The runtime policy is selected, but exact config surface ownership is not yet
turned into concrete tasking.

Impact:

Config keys, default-disabled behavior, env docs, validation tests, and
readiness gates are part of the production runtime contract. Leaving them to
ad hoc implementation would blur config-source ownership.

Resolution:

Planning must include explicit config tasks covering the broader
balance/usage-authority gate, default-disabled behavior, required Postgres,
service-auth, microlease, worker, Redpanda, admission-control, and
Redis-not-authority validation, plus `env/config/default.yaml` and
`env/.env.example` documentation.

### TDR-C04 Operation-readback scope repair must be carried as a contract task

Classification: proof_obligation

Evidence:

- Current OpenAPI declares `/internal/billing/v1/operations/readback` with
  `billing.operations.read` (`api/openapi/service.yaml:228`,
  `api/openapi/service.yaml:235`).
- Current middleware still groups operations readback under the microlease read
  route scope (`internal/infra/http/service_auth.go:302`).
- The design names the repair in `design/contracts/http-api.md:79`.

Issue:

The design correctly identifies the mismatch, but planning must make it an
explicit generated-contract and middleware task.

Impact:

If omitted, a route could satisfy the generated OpenAPI contract but enforce
the wrong production scope, weakening least-privilege read access.

Resolution:

Planning must include OpenAPI, middleware constants, route-scope tests, and
proxy caller-scope proof for `billing.operations.read`.

### TDR-C05 Event topic label mismatch must be closed in implementation proof

Classification: proof_obligation

Evidence:

- `design/contracts/events.md:20` selects `billing.microlease.facts.v1`.
- Current config defaults use `billing.microlease.facts.v1`
  (`env/config/default.yaml:69`, `internal/config/defaults.go:67`).
- Current Redpanda safe-topic handling still recognizes `billing.facts.v1`
  (`internal/infra/redpanda/consumer.go:314`).
- The design explicitly says that mismatch must be repaired if implementation
  keeps the old safe-label code (`design/contracts/events.md:22`).

Issue:

The topic authority is chosen, but generated/runtime label consistency is a
named proof obligation.

Impact:

If planning misses it, telemetry, safe labels, tests, or producer/consumer
config could disagree about the billing facts topic.

Resolution:

Planning must include topic/config/safe-label tests proving the selected topic
family is consistent across config defaults, Redpanda adapters, outbox relay,
fixtures, and metrics labels.

## Handoffs

Planning handoff:

- Write `tasks.md` only after carrying TDR-C01 through TDR-C05 into executable
  tasks, `test-plan.md`, `rollout.md`, or the task-ledger review/readiness
  record.
- Planning may create `test-plan.md` and `rollout.md`; both are already
  triggered by the reviewed design and workflow state.
- Planning must either include authorized cross-repo `gonka-proxy` tasks and
  proof, or stop with a precise proxy implementation handoff. Billing-service
  proof alone cannot claim cutover completion.

## Design Escalations

None now.

Reopen technical design if planning cannot close TDR-C01, TDR-C02, TDR-C03,
TDR-C04, or TDR-C05 without choosing new ownership, data, contract, runtime, or
rollout policy.

Reopen specification if planning or implementation requires direct per-request
reserve fallback, proxy-local money writes for migrated cohorts, non-JWT
bearer-key production auth, top-up/payment ownership, organization charging,
Redis or memory spend authority, weaker privacy policy, or runtime behavior
that cannot fail closed.

## Residual Risks

- The review did not run tests, generation, migrations, or builds because this
  phase is read-only design review.
- The checkout is dirty with existing implementation and workflow artifacts;
  this review treated those files as current local evidence and did not revert
  or stage anything.
- Proxy evidence was local source inspection only; no proxy tests or live
  traffic proof were run in this phase.

## Validation Commands

Read-only inspection only:

- `rtk rg` and `rtk sed` over the reviewed workflow/design docs.
- `rtk rg` and targeted reads over current billing-service OpenAPI, middleware,
  handlers, bootstrap, worker bootstrap, migrations, SQL queries, config, and
  Redpanda code.
- `rtk rg` over current `gonka-proxy` shared-balance, microlease, completion,
  web-search, balance, Prisma, and internal-money contract surfaces.

No validation command was run or claimed as implementation readiness proof.

## Completion Marker

Technical design review is complete when this file records the reviewed packet,
method, findings, orchestrator resolution, gate status, planning-input
obligations, residual risks, and next phase.

Completion status: complete.

## Next Phase

Next phase: planning only.

Planning may start from this reviewed packet with gate status `CONCERNS`.
Planning must stop after writing and reviewing `tasks.md`, plus any required
`test-plan.md` and `rollout.md`, and must not begin implementation.
