# Balance And Usage Authority Cutover Test Plan

Status: approved for implementation planning
Date: 2026-06-02
Owner: orchestrator

## Purpose

This plan defines the proof required before the balance and usage authority
cutover can claim implementation readiness or completion. It consumes the
approved `spec.md`, reviewed `design/` bundle, and technical-design-review gate
status `CONCERNS`.

This file is a planning artifact only. No validation has been run in the
planning phase.

## Required Proof Classes

### Contract And Service Auth

Acceptance criteria:

- `api/openapi/service.yaml` is the REST source of truth for account resolve,
  balance read, usage reserve/finalize/write-off/reversal, usage readback,
  operation readback, reconciliation readback, and admin readbacks.
- Generated OpenAPI bindings are derived from the contract and not edited by
  hand.
- Every protected route uses scoped JWT/JWKS service auth.
- `/internal/billing/v1/operations/readback` enforces
  `billing.operations.read` in OpenAPI, middleware scope mapping, route-scope
  tests, and proxy caller scopes.
- Transport validation rejects missing idempotency, changed fingerprints,
  non-USD money inputs, unsafe metadata, body/path identity mismatches, and
  missing represented-user/account context where required.

Planned proof:

- `rtk make openapi-check`
- targeted `go test` for `internal/infra/http` and service-auth scope mapping
- OpenAPI breaking check against a PR base when a base contract is available:
  `BASE_OPENAPI=<base-file> rtk make openapi-breaking`

Carries: TDR-C04.

### Data, SQLC, And Money Transactions

Acceptance criteria:

- Account resolve and rollout gates can read latest accepted import/parity
  state from `legacy_import_batches` and `legacy_balance_imports` using
  deterministic indexed queries.
- If that readback cannot be expressed without ambiguity or expensive scans,
  implementation stops and reopens technical design rather than inventing a
  projection ad hoc.
- Migrated generic usage reserve/readback records durable
  `usage_operation_id` linkage for generated usage-operation lookup paths.
- Account balance, usage operation, terminal outcome, idempotency, operation
  outcome, ledger, reconciliation, inbox/outbox, microlease, and child-debit
  writes happen in short Postgres transactions with row locks where invariants
  require single-writer behavior.
- No cross-service HTTP, Redpanda publish, JWKS fetch, pricing call, proxy
  call, or Redis operation runs while a money transaction is open.

Planned proof:

- `rtk make sqlc-check`
- `rtk make migration-validate` or `rtk make docker-migration-validate`
- targeted repository and integration tests for account resolve, balance read,
  import/parity readback, usage linkage, terminal settlement, rollback,
  inbox/outbox, row locks, duplicate replay, changed-fingerprint conflict,
  non-negative balances, and ledger conservation

Carries: TDR-C01 and TDR-C02.

### App And Domain Behavior

Acceptance criteria:

- `internal/app/billingauthority` or the chosen equivalent app package owns
  transport-agnostic account resolve, balance read, usage lifecycle,
  operation readback, reconciliation readback, and admin readback behavior.
- Migrated reserve uses
  `billing_microlease_with_proxy_child_debit` lineage and never falls back to a
  direct account-balance reserve for migrated proxy callers.
- Active microlease exposure and active usage exposure remain visible reserved
  exposure until terminal, close, release, write-off, reversal, compensation,
  or reconciliation proof exists.
- Expiry alone never releases customer money.
- Replays return stored outcomes and changed fingerprints return conflicts.
- Support-safe reconciliation cases are opened or updated for stale,
  ambiguous, over-cap, missing-evidence, import-mismatch, inbox conflict, and
  terminal-lag conditions.

Planned proof:

- targeted `go test` for app/domain packages such as
  `internal/app/billingauthority`, `internal/app/microlease`,
  `internal/app/reconciliation`, and `internal/domain/money`
- property or table tests for USD atom parsing/formatting, active exposure
  conservation, non-negative available balance, cap enforcement,
  replay/conflict handling, stale operation policy, write-off, reversal, and
  compensation behavior

### HTTP Runtime And Bootstrap

Acceptance criteria:

- Enabled cutover routes are backed by concrete app services wired through
  bootstrap.
- Handler-level `503` is used only for disabled, not-ready, unhealthy, or
  intentionally absent runtime state, not as enabled production behavior.
- The broader balance/usage authority runtime gate defaults disabled and has
  explicit config keys, env documentation, validation, readiness, and
  admission behavior.
- Enabled migrated authority requires Postgres, scoped service auth,
  microlease runtime, billing worker readiness, Redpanda health, admission
  controls, and Redis-not-authority validation.

Planned proof:

- targeted `go test` for `internal/config`,
  `cmd/service/internal/bootstrap`, and `internal/infra/http`
- config snapshot/default/env tests proving default-disabled behavior,
  required dependencies, readiness gates, and Redis-not-authority checks

Carries: TDR-C03.

### Worker, Redpanda, Inbox, Outbox, And Reconciliation

Acceptance criteria:

- `cmd/billing-worker` wires concrete tasks for terminal consumer, checkpoint
  consumer, close consumer, inbox retry, outbox relay, stale reconciliation,
  and admission-control renewal when enabled.
- Enabled-but-no-op worker construction is rejected by tests and readiness.
- Redpanda terminal, checkpoint, close, and billing-fact topics use the selected
  topic family, including `billing.microlease.facts.v1`.
- Safe topic labels, config defaults, adapters, outbox relay, fixtures, and
  metrics labels agree on the selected topic names.
- Consumers apply, duplicate, or quarantine before committing offsets.
- Outbox publish failure schedules retry without rolling back committed money
  state.
- Worker shutdown is context-aware and leaves uncommitted work replayable.

Planned proof:

- `rtk make proto-check` when event contract sources or generated event DTOs
  change
- targeted `go test` for `internal/infra/redpanda`,
  `internal/app/microleaseworker`, `cmd/billing-worker/...`, and affected
  Postgres repository packages
- worker lifecycle tests for readiness, dependency probes, bounded
  concurrency, cancellation, shutdown, replay, quarantine, redrive, lag gates,
  and low-cardinality metrics

Carries: TDR-C05.

### Proxy Cutover Proof

Acceptance criteria:

- Migrated completion and web-search cohorts use billing account resolve,
  billing balance/readback, billing-issued microlease authority, durable proxy
  child debit, and terminal obligation before external execution.
- Migrated cohorts do not use local `balanceNgonka`, in-memory reservations,
  local `BalanceTransaction` money writes, or direct reserve fallback.
- Proxy replaces or adapts the old bearer-key `/api/v1/usage/*`
  shared-balance bridge to the billing-service `/internal/billing/v1`
  contract and scoped service JWT auth.
- Proxy uses `billing.operations.read` for operation-readback calls.
- Ambiguous billing outcomes retry or read back with the same operation
  identity and do not mint a fresh money operation.
- Legacy cohort behavior remains isolated.
- Proxy local rows, terminal/checkpoint/close events, logs, and metrics stay
  privacy-safe.

Planned proof:

- targeted `rtk bun test` commands in
  `/Users/daniil/Projects/GonkaGate/gonka-proxy` for the billing client,
  microlease allocator, migrated-cohort policy, completion billing, web-search
  billing guards, terminal/checkpoint/close publication, and privacy fixtures
- proxy typecheck may be attempted with `rtk bun run typecheck`; if it remains
  blocked by pre-existing unrelated TypeScript errors, record the exact files
  and keep the proxy readiness claim limited to the targeted cutover proof
- do not run `npm run build`, `npm run test`, `npm run check`, or
  `node scripts/check-fastify-hooks.mjs` in `gonka-proxy` without explicit
  user approval, per that repository's `AGENTS.md`

### Rollout, Failback, And Operator Readbacks

Acceptance criteria:

- `inert_expand`, `shadow_no_spend`, `internal_cohort`, `migrated`, and
  `rollback` modes are represented by config, docs/runbooks, tests, or
  dry-run evidence.
- Migrated enablement is blocked unless import/parity, account state, balance
  initialization, old proxy-writer disablement, direct reserve fallback
  disablement, service auth, billing HTTP runtime, billing worker runtime,
  Redpanda health, lag/backlog gates, and operator readbacks pass.
- Rollback never silently restores proxy-local money writes for already
  migrated accounts.
- Critical worker lag, stale exposure, inbox/outbox backlog, auth failure,
  import mismatch, quarantine, or reconciliation backlog fails paid admission
  closed for migrated cohorts.

Planned proof:

- targeted tests for rollout policy and admission gates
- dry-run or fixture proof for account import/parity and cohort enablement
- runbook/readback checks in `rollout.md` and any implementation-owned docs

### Privacy And Security

Acceptance criteria:

- APIs, events, logs, traces, metrics, durable rows, inbox/outbox, audit rows,
  proxy local rows, reconciliation notes, fixtures, workflow artifacts, and
  runbooks exclude raw prompts, completions, SSE chunks, bearer tokens, API
  keys, DSNs, payment secrets, raw provider payloads, raw event payloads,
  dynamic proof URLs, and sensitive request bodies.
- Metrics labels are low cardinality and do not use raw account IDs,
  request IDs, idempotency keys, public request bodies, tokens, or secrets.

Planned proof:

- targeted privacy assertions in HTTP, app, Redpanda, Postgres, worker, and
  proxy packages
- `rtk make go-security`
- `rtk make secret-scan`
- targeted `rtk rg` privacy scan over changed implementation surfaces and
  task-local artifacts, with matches classified as policy text, generated
  identifiers, safe test sentinels, or blockers

### Performance

Acceptance criteria:

- Billing issue/replenish p95 stays under 100 ms and p99 under 250 ms.
- Proxy durable child allocation p95 stays under 10 ms and p99 under 25 ms.
- Cold replenishment p95 stays under 250 ms and p99 under 500 ms.
- First-token added latency p95 stays under 25 ms.
- Terminal ingestion, checkpoint/close cadence, stale reconciliation scans,
  account contention, and worker lag gates are measured.
- No benchmark success depends on memory-only or Redis-only spend authority.

Planned proof:

- billing integration or benchmark tests for issue/replenish, terminal
  ingestion, checkpoint/close, stale scans, account contention, and worker
  throughput
- proxy targeted performance tests for durable child allocation, active
  admission with optional memory precheck, cold replenishment, and first-token
  impact
- reopen specification if meeting the budget requires unbacked memory or Redis
  spend

## Final Validation Bundle

Minimum billing-service proof before closeout:

- `rtk make check`
- `rtk make openapi-check`
- `rtk make proto-check` when event/proto surfaces change
- `rtk make sqlc-check`
- `rtk make migration-validate` or `rtk make docker-migration-validate`
- targeted integration tests for the new account, balance, usage, worker,
  Redpanda, import/parity, and reconciliation surfaces
- targeted worker/event tests
- `rtk make go-security`
- `rtk make secret-scan`
- `rtk make check-full` when Docker and local context permit

Minimum proxy proof before full authority-cutover closeout:

- targeted migrated completion and web-search tests
- targeted billing client/JWT/scope tests
- targeted microlease allocator, terminal obligation, checkpoint/close, and
  no-local-writer tests
- targeted proxy performance proof for durable child allocation and first-token
  impact

## Readiness Review

Task-ledger review result: PASS.

Implementation readiness contribution: PASS.

Rationale: each required proof class maps to executable tasks in `tasks.md`.
TDR-C01 through TDR-C05 are carried as named proof obligations and no test-plan
item asks implementation to choose new architecture, ownership, contract,
data, runtime, rollout, or validation policy.

Reopen technical design if an implementation task cannot satisfy a proof class
without a new ownership, data, contract, runtime, rollout, or validation
decision.

Reopen specification if proof requires direct per-request reserve fallback,
proxy-local money writes for migrated cohorts, non-JWT bearer-key production
auth, top-up/payment ownership, organization charging, Redis or memory spend
authority, weaker privacy policy, or runtime behavior that cannot fail closed.
