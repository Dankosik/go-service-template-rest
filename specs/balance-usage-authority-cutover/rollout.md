# Balance And Usage Authority Cutover Rollout Plan

Status: approved for implementation planning
Date: 2026-06-02
Owner: orchestrator

## Purpose

This rollout plan defines the mixed-version deployment, gating, rollback, and
operator-readback requirements for moving migrated `gonka-proxy` paid cohorts
to `billing-service` balance and usage authority.

This file is a planning artifact only. No deployment, migration, validation, or
live cutover has been run in the planning phase.

## Rollout Principles

- Billing-service Postgres remains customer-money authority.
- Migrated paid admission is microlease-first and requires durable proxy child
  debit plus terminal obligation before external execution.
- Proxy-local balance writes are legacy-only for migrated account scopes.
- Direct per-request reserve fallback is disabled for migrated cohorts.
- Redis and process memory may deny, cache, precheck, or shape only over
  durable authority. They cannot mint customer-money spend authority.
- Redpanda transports evidence. It is not money mutation authority.
- Critical uncertainty fails migrated paid admission closed.

## Rollout Modes

### 1. inert_expand

Purpose: deploy schema, contracts, config, worker binaries, and proxy adapters
without moving paid spend authority.

Required conditions:

- Billing broader authority runtime gate defaults disabled.
- Billing worker runtime defaults disabled.
- Redpanda topics and consumer groups may be declared but do not admit migrated
  paid traffic.
- Proxy adapter code is dark-launched or disabled.
- No migrated paid admission uses billing authority.

Exit proof:

- contract, SQLC, migration, config, worker bootstrap, and privacy tests pass;
- disabled runtime returns fail-closed readiness/readbacks where applicable;
- no proxy local money path changes are active for legacy cohorts.

### 2. shadow_no_spend

Purpose: prove account resolve, balance read, import/parity, and readback
visibility without admitting migrated paid external execution.

Required conditions:

- Account import/backfill has applied candidate proxy balance snapshots.
- Billing readbacks return import/parity, active exposure, stale operation,
  worker/admission, reconciliation, and manual-review state.
- Proxy old writers still own legacy paid cohorts.
- Shadow-only billing state cannot admit migrated paid external execution.

Exit proof:

- imported account scopes have active billing accounts and initialized USD atom
  balances;
- parity status passes for candidate internal accounts;
- operation/admin readbacks expose lag, stale exposure, inbox/outbox, and
  reconciliation state;
- any mismatch opens support-safe reconciliation rather than enabling spend.

### 3. internal_cohort

Purpose: enable a small internal cohort with billing authority.

Required conditions:

- Old proxy money writers are disabled for those account scopes.
- Direct reserve fallback is disabled.
- Service JWT/JWKS auth is configured for proxy-to-billing calls.
- Billing HTTP runtime and billing worker runtime are ready.
- Redpanda terminal, checkpoint, close, and billing fact topics are healthy.
- Operator readbacks are available before traffic moves.

Exit proof:

- internal completion and web-search paid paths commit durable child debit and
  terminal obligation before external execution;
- no migrated local `balanceNgonka`, in-memory reservation authority, or local
  `BalanceTransaction` money write occurs;
- terminal/checkpoint/close events settle or reconcile through billing;
- rollback mode has been exercised in dry-run or targeted tests.

### 4. migrated

Purpose: move approved paid cohorts to billing-service balance and usage
authority.

Required conditions:

- Import/parity accepted for every migrated account scope.
- Billing account state is active.
- Account balance is initialized in USD atoms.
- No blocking reconciliation/manual-review state exists.
- Old proxy writer is disabled for the account scope.
- Direct reserve fallback is disabled.
- Scoped service auth, billing HTTP runtime, billing worker runtime,
  Redpanda health, admission controls, and operator readbacks pass.
- Terminal lag, stale exposure, inbox/outbox backlog, and reconciliation
  backlog are below critical gates.

Exit proof:

- migrated paid admission fails closed on any critical gate failure;
- same-identity retry/readback handles ambiguous outcomes;
- billing readbacks show active exposure and terminal settlement;
- proxy proof shows no dual writer for migrated scopes.

### 5. rollback

Purpose: stop or narrow migrated traffic without creating dual money authority.

Allowed rollback behavior:

- fail migrated paid admission closed; or
- allow only already minted, valid microleases until debit cutoff/cap while old
  proxy money writers remain disabled for migrated account scopes.

Rejected rollback behavior:

- silently restoring proxy-local money writes for migrated accounts;
- using local in-memory reservations as spend authority;
- enabling direct per-request reserve fallback;
- releasing expired microlease exposure without terminal, close, release,
  write-off, reversal, compensation, or reconciliation proof.

Reopen specification if rollback requires proxy-local money writes for migrated
account scopes.

## Deployment Order

1. Land billing-service contract/data/runtime foundation disabled:
   OpenAPI, event/proto, SQLC, migrations, config defaults, service-auth
   scopes, and runtime gates.
2. Deploy billing-service HTTP with broader authority runtime disabled.
3. Deploy billing-worker binary with worker runtime disabled.
4. Create or verify Redpanda topics and consumer groups for terminal,
   checkpoint, close, and billing facts.
5. Enable shadow account import/parity jobs or tools for selected internal
   account scopes.
6. Deploy proxy adapter code disabled or in shadow mode, including scoped JWT
   auth and `/internal/billing/v1` client support.
7. Enable billing HTTP broader authority for internal shadow readbacks only.
8. Enable billing worker concrete tasks for internal shadow and readback proof.
9. Enable internal cohort mode for explicitly selected account scopes.
10. Expand migrated mode only after all gates and proof in this plan pass.

## Gate Matrix

| Gate | inert_expand | shadow_no_spend | internal_cohort | migrated |
| --- | --- | --- | --- | --- |
| Billing runtime gate | disabled | readback only | enabled | enabled |
| Billing worker concrete tasks | disabled or dry | enabled for readback/lag proof | enabled | enabled |
| Account import/parity | not required | required for shadow accounts | required | required |
| Old proxy writer disabled | not required | not required for legacy cohorts | required for cohort | required |
| Direct reserve fallback disabled | required for migrated paths | required for shadow migrated paths | required | required |
| Scoped JWT/JWKS auth | contract proof | readback proof | required | required |
| Redpanda health | config proof | readback proof | required | required |
| Operator readbacks | contract proof | required | required | required |
| Paid external execution | legacy only | legacy only | internal cohort only | migrated cohorts |

## Operator Readbacks

Before any migrated paid cohort is enabled, operators must be able to read:

- account resolve and import/parity status;
- account balance settled/reserved/available/pending USD atom state;
- active usage holds, active microleases, child debits, terminal lag, and close
  gaps;
- operation outcomes by billing operation ID, usage operation ID, microlease
  ID, child debit ID, terminal outcome ID, idempotency key, or reconciliation
  case link according to generated contract rules;
- ledger and balance-version history;
- reconciliation cases by reason, severity, state, and account;
- inbox conflict/quarantine backlog;
- outbox retry backlog;
- worker lag and admission-control freshness;
- Redpanda topic and consumer group health;
- proxy cohort mode, old-writer disabled state, direct reserve fallback state,
  and terminal/checkpoint/close publication health.

## Alerts And Fail-Closed Triggers

Critical conditions:

- billing-service unavailable or not ready;
- scoped service auth missing, invalid, or missing required scope;
- represented user/account binding mismatch;
- account missing, not imported, suspended, manual review, or reconciliation
  required;
- import parity mismatch;
- stale pricing or missing pricing lineage;
- missing, stale, expired, cutoff, over-cap, or conflicting microlease/child
  debit lineage;
- missing durable child debit or terminal obligation before external execution;
- worker runtime disabled, no-op, unhealthy, or lagging beyond critical gates;
- terminal lag, stale exposure, inbox quarantine, outbox backlog, or
  reconciliation backlog above critical gates;
- Redis or memory configured as spend authority;
- direct reserve fallback enabled for migrated cohorts;
- proxy-local money writer enabled for migrated account scopes.

Required response:

- migrated paid admission fails closed;
- callers retry or read back with the same operation identity after ambiguous
  outcomes;
- operators use reconciliation/admin readbacks to diagnose;
- no new billing operation identity is minted after possible acceptance.

## Rollback Proof

Implementation must prove:

- old proxy writers remain disabled for migrated account scopes during rollback;
- direct reserve fallback remains disabled;
- already minted valid microleases are either honored only until cutoff/cap or
  migrated paid admission fails closed;
- active exposure remains visible in billing readbacks until durable terminal,
  close, release, write-off, reversal, compensation, or reconciliation proof;
- rollback does not create duplicate ledger effects or a dual writer.

## Task Mapping

- Contract and service auth: `tasks.md` T002, T003, T010, T016.
- Import/parity and data gates: `tasks.md` T004, T005, T006, T007.
- Runtime gates and worker readiness: `tasks.md` T004, T011, T012, T013.
- Redpanda topic consistency: `tasks.md` T003, T012, T013.
- Proxy cutover: `tasks.md` T016, T017, T018, T019.
- Operator readbacks and rollout gates: `tasks.md` T009, T014, T020.
- Final proof and closeout: `tasks.md` T021, T022, T023, T024, T025.

## Readiness Review

Task-ledger review result: PASS.

Implementation readiness contribution: PASS.

Rationale: rollout modes, gates, failback rules, operator readbacks, and
critical fail-closed triggers are represented in executable tasks and proof
obligations. This rollout plan does not require implementation to choose a new
rollout policy.

Reopen technical design if implementation needs a new mixed-version,
readiness, worker, Redpanda, or operator-readback policy.

Reopen specification if rollout requires direct per-request reserve fallback,
proxy-local money writes for migrated cohorts, non-JWT bearer-key production
auth, top-up/payment ownership, organization charging, Redis or memory spend
authority, weaker privacy policy, or runtime behavior that cannot fail closed.
