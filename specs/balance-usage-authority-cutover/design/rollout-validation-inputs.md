# Rollout And Validation Design Inputs

Status: review-ready
Date: 2026-06-02

## Purpose

This file records design inputs for later `test-plan.md`, `rollout.md`, and
`tasks.md`. It is not a test plan, rollout plan, task ledger, validation
transcript, or implementation handoff.

## Rollout Modes

Later rollout planning must preserve these modes:

1. `inert_expand`
   - Billing schema, contracts, worker config, and proxy adapters may be
     deployed disabled.
   - No migrated paid admission uses billing authority.
2. `shadow_no_spend`
   - Billing account resolve and balance read run for parity.
   - Proxy old writers still own legacy cohorts.
   - No migrated paid external execution is admitted by shadow-only
     microlease state.
3. `internal_cohort`
   - Limited internal accounts use billing microlease authority.
   - Old proxy money writers must be disabled for those account scopes.
   - Parity, worker health, and operator readbacks are required.
4. `migrated`
   - Migrated paid cohorts require billing microlease plus proxy durable child
     debit before external execution.
   - Direct reserve fallback and proxy-local money writes are disabled.
5. `rollback`
   - Does not restore proxy-local writes for migrated accounts.
   - Either fails paid admission closed or allows only already minted valid
     microleases until cutoff/cap while old proxy writers remain disabled.

## Rollout Gates

Migrated enablement requires:

- account import batch applied and parity accepted;
- billing account state active;
- account balance initialized in USD atoms;
- no blocking reconciliation/manual-review state;
- old proxy writer disabled for the account scope;
- direct reserve fallback disabled;
- service JWT/JWKS auth configured;
- billing HTTP runtime ready;
- billing worker runtime ready with concrete tasks;
- Redpanda topics and consumer group healthy;
- terminal lag, stale exposure, inbox/outbox backlog, and reconciliation
  backlog below configured critical gates;
- operator readbacks available for balance, exposure, ledger, worker lag,
  inbox/outbox, and reconciliation.

Any critical gate failure blocks paid admission for migrated cohorts.

## Validation Classes For Later Test Plan

Billing-service proof:

- OpenAPI generation/drift/runtime contract/lint/schema validation;
- route-scope and service-auth tests for JWT/JWKS, scope separation,
  operation-readback `billing.operations.read`, account binding, and
  represented user context;
- SQLC generation/drift and migration rehearsal;
- Postgres integration tests for account resolve, balance read, import parity,
  row locks, non-negative balances, idempotency replay/conflict, operation
  outcomes, terminal settlement, rollback, inbox/outbox, and reconciliation;
- app/domain tests for USD atom parsing, reserve/finalize/write-off/reversal,
  microlease exposure, replay/conflict, stale/ambiguous operation policy, and
  active exposure conservation;
- HTTP tests for status/result mapping, ambiguous timeout readback, body/path
  identifier rules, bounded Problems, low-cardinality route labels, and
  privacy rejection;
- worker/Redpanda tests for terminal/checkpoint/close consumers, inbox retry,
  outbox relay, quarantine/redrive, offset commit after store effect, worker
  readiness, bounded concurrency, and shutdown;
- reconciliation/admin tests for stale operation visibility, import/parity
  state, ledger history, and support-safe metadata;
- performance benchmarks at least matching the approved microlease budgets
  unless a later review-approved design records stricter or updated budgets;
- privacy/security proof with repository-owned `rtk make go-security`,
  `rtk make secret-scan`, and targeted privacy assertions;
- repository validation with `rtk make check`, `rtk make openapi-check`,
  `rtk make sqlc-check`, `rtk make migration-validate` or Docker equivalent,
  targeted integration tests, and `rtk make check-full` when Docker/context
  permits.

Proxy proof when cross-repo implementation is authorized:

- migrated completion path uses billing account resolve/readback and durable
  microlease/child debit authority;
- migrated web-search path no longer fails because local billing is blocked,
  and instead uses the billing authority path or fails closed before external
  execution;
- no migrated proxy-local `balanceNgonka` deduction, in-memory reservation
  authority, or local `BalanceTransaction` money write;
- direct reserve fallback disabled for migrated cohorts;
- durable child debit and terminal obligation committed before external
  execution;
- same-identity retry/readback after ambiguous billing outcome;
- legacy cohort isolation;
- privacy-safe local child debit, terminal, checkpoint, and event rows.

## Performance Budgets

Use the predecessor microlease budgets as minimum proof:

- billing issue/replenish p95 under 100 ms and p99 under 250 ms;
- proxy durable child allocation p95 under 10 ms and p99 under 25 ms;
- cold replenishment p95 under 250 ms and p99 under 500 ms;
- first-token added latency p95 under 25 ms;
- terminal ingestion, checkpoint/close cadence, stale reconciliation scan, and
  account contention measured under targeted integration or benchmark tests.

If success requires memory-only or Redis-only spend, reopen specification.

## Rollout Artifact Inputs

Later `rollout.md` must define:

- deploy order for billing service, billing worker, Redpanda topics, and proxy
  adapter changes;
- expand/backfill/parity/contract sequence for account imports;
- migrated cohort selection and operator approval gates;
- mixed-version behavior between old proxy adapters and new billing routes;
- rollback behavior that avoids dual writers;
- fail-closed operator playbooks for worker lag, billing outage, auth failure,
  import mismatch, stale terminal, inbox quarantine, outbox backlog, and
  reconciliation backlog;
- production readiness readbacks and alert thresholds.

## Reopen Conditions

Reopen specification instead of planning if rollout or validation requires:

- direct per-request reserve fallback;
- proxy-local money writes for migrated cohorts;
- non-JWT bearer-key production auth;
- top-up/payment ownership;
- organization charging;
- Redis or memory spend authority;
- weaker privacy policy;
- a runtime shape that cannot fail closed.
