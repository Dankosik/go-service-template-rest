# Billing Money Core Data-Model Tasks

Status: approved for implementation  
Planning phase: complete  
Task-ledger review: PASS  
Implementation readiness: PASS  
Owner: orchestrator

## Goal Contract

Goal objective: Complete the approved billing-money-core data-model implementation by executing this ledger from `T001` through final validation.

Stopping condition: all required tasks are checked, required migration/sqlc/test/benchmark proof passes or records a concrete blocker, and ledger-owned closeout evidence is current.

Read first:
- `specs/billing-money-core/tasks.md` because it is the approved implementation ledger and source of truth.
- `specs/billing-money-core/spec.md` because it is the canonical data-model decision record.
- `specs/billing-money-core/design/data-model.md` because it is the reviewed table, constraint, index, transaction, readback, and test-obligation design.
- `specs/billing-money-core/workflow-plans/technical-design-review.md` because it records the follow-up PASS and confirms there are no accepted `CONCERNS`.
- `docs/build-test-and-development-commands.md` because it owns validation command semantics.
- `docs/repo-architecture.md` because it owns migration/sqlc/source-of-truth placement.

Do not change:
- HTTP/OpenAPI contracts, public/internal route names, generated API bindings, runtime adapters, worker design, broad service architecture, or cross-service contract ownership.
- The approved boundaries: USD-only customer ledger, GNK inventory outside customer balance truth, `request_id` correlation-only, qualified inference evidence for settlement, database-backed idempotency and stored outcomes, account-first locking, one short transaction per money command, support-safe readback, and legacy `balanceNgonka`/`lockedRateUsd` as import evidence only.
- Privacy posture: no raw prompts, completions, SSE payloads, bearer tokens, API keys, DSNs, payment secrets, or raw PSP webhook bodies in tests, fixtures, logs, audit metadata, or closeout notes.

Progress log: update each task checkbox and `Evidence:` line after its proof. After each checkpoint, record the exact command or manual proof result. If blocked, leave the task unchecked and add `Blocked:` with the missing decision, missing artifact, failing command, or unavailable dependency.

Blocked-stop rule: if implementation needs a schema, ownership, API, runtime, worker, rollout, or validation decision not present in the approved spec/design/review packet, stop and reopen the named earlier phase instead of inventing it in code.

## Implementation Handoff

Consumes: approved `spec.md`, reviewed `design/data-model.md`, follow-up technical-design-review PASS, this task ledger, `docs/build-test-and-development-commands.md`, and `docs/repo-architecture.md`.

Task-ledger review: PASS.

Implementation readiness: PASS.

First executable task: T001.

Accepted concerns: none. The follow-up technical design review is PASS and names no `CONCERNS`.

Proof obligations:
- schema migration rehearsal and rollback/reapply proof;
- SQLC generation/drift proof for any query sources added by this ledger;
- integration proof for schema constraints, ledger delta patterns, reconciliation dedupe, idempotency, concurrency, legacy import, and support readback indexes;
- benchmark/performance evidence for reserve, finalize, write-off, and top-up evidence data paths;
- normal Go unit/race proof for any hand-written helper or repository code added by this ledger;
- final `rtk git diff --check`.

Reopen target:
- Reopen `technical design` if a task requires a table, column, constraint, index, dedupe key, lock order, transaction rule, query shape, or proof class not specified by `design/data-model.md`.
- Reopen `specification` if implementation requires changing source-of-truth ownership, customer-money units, settlement identity, idempotency semantics, privacy exclusions, or the data-model slice boundary.
- Reopen `planning` if the approved decisions are stable but task order, file ownership, or proof commands are insufficient.

## Planning Review

Review result: PASS.

Review basis:
- The ledger is limited to data-model implementation surfaces: `env/migrations/`, `internal/infra/postgres/queries/`, derived `internal/infra/postgres/sqlcgen/`, data-model package/repository code only when directly required by the approved schema/query proof, and tests/benchmarks.
- Every follow-up TDR blocker is represented in executable tasking: TDR-F01 reconciliation dedupe, TDR-F02 constrained-text coverage, and TDR-F03 ledger delta zeroing.
- No task asks implementation to design HTTP/OpenAPI contracts, runtime adapters, workers, broad package architecture, migration cutover choreography, or external provider contracts.
- `test-plan.md`, `rollout.md`, `design/contracts/`, and `design/dependency-graph.md` remain not expected for this data-model-only implementation slice; their triggers stay outside this ledger.

Implementation may start from this ledger.

## Tasks

- [x] T001 [Checkpoint 1: schema source of truth] Add the billing money core migration under `env/migrations/` with deterministic up/down SQL for the approved account, balance, ledger, idempotency, outcome, usage, top-up, evidence, reconciliation, audit, and legacy import tables.
  Depends on: none.
  Proof: `rtk make migration-validate`.
  Evidence: `rtk make migration-validate` passed; applied `000003_billing_money_core`, rolled it down, and reapplied it.

- [x] T002 [Checkpoint 1: schema source of truth] Encode the approved table-level constraints and indexes in the migration, including account scope uniqueness, non-negative balance checks, `available = settled - reserved`, constrained-text checks, ledger `settlement_effect_id` uniqueness, payment evidence fingerprint uniqueness, idempotency uniqueness, terminal outcome uniqueness, stale-hold/reconciliation claim indexes, and support readback keyset indexes.
  Depends on: T001.
  Proof: `rtk make migration-validate` plus targeted migration diff read against `specs/billing-money-core/design/data-model.md`.
  Evidence: `rtk make migration-validate` passed; targeted migration read confirmed account/balance uniqueness and checks, ledger delta and settlement uniqueness, payment evidence fingerprint uniqueness, idempotency uniqueness, terminal uniqueness, reconciliation partial uniqueness, and declared support/readback indexes in `env/migrations/000003_billing_money_core.up.sql`.

- [x] T003 [Checkpoint 1: schema source of truth] Implement the approved ledger delta-pattern checks for every effect type, including explicit zeroing of disallowed settled/reserved/pending components and the repaired `usage_charge` rule that forbids pending-balance mutation.
  Depends on: T002.
  Proof: migration-level negative tests in T008 plus `rtk make migration-validate`.
  Evidence: `rtk make migration-validate` passed; `rtk make test-integration` passed with negative ledger delta coverage proving `usage_charge` rejects `pending_delta_usd_atoms` mutation and posted ledger money fields are immutable.

- [x] T004 [Checkpoint 1: schema source of truth] Implement the approved reconciliation lineage requirements and duplicate-open-case partial uniqueness rules, including usage, top-up/payment-attempt, payment evidence, settlement effect, qualified inference evidence, ledger-entry, and `legacy_balance_import_id` lineage.
  Depends on: T002.
  Proof: reconciliation integration tests in T010 plus `rtk make migration-validate`.
  Evidence: `rtk make migration-validate` passed; `rtk make test-integration` passed with stale-reservation, provider-reference, payment-evidence, missing-inference, leased-claim, and legacy-import reconciliation coverage.

- [x] T005 [Checkpoint 1: schema source of truth] Implement the approved legacy import schema surfaces: `legacy_import_batches`, `legacy_balance_imports`, migration import ledger linkage, parity status, import fingerprints, and evidence-only legacy `balanceNgonka` / `lockedRateUsd` fields.
  Depends on: T002, T004.
  Proof: legacy import integration tests in T011 plus `rtk make migration-validate`.
  Evidence: `rtk make migration-validate` passed; `rtk make test-integration` passed with import batch/account/fingerprint uniqueness, migration ledger linkage, mismatch reconciliation linkage, and live balance readback staying on `account_balances`.

- [x] T006 [Checkpoint 2: generated access] Add SQLC query sources under `internal/infra/postgres/queries/` for the approved data-model access shapes without adding HTTP handlers, runtime adapters, workers, or bootstrap wiring.
  Depends on: T001, T002, T003, T004, T005.
  Proof: `rtk make sqlc-check`.
  Evidence: `rtk make sqlc-check` passed after adding `internal/infra/postgres/queries/billing_money_core.sql`; no HTTP handlers, runtime adapters, workers, or bootstrap wiring were added.

- [x] T007 [Checkpoint 2: generated access] Regenerate derived SQLC output under `internal/infra/postgres/sqlcgen/` and keep generated artifacts aligned with the migration/query source of truth.
  Depends on: T006.
  Proof: `rtk make sqlc-check`.
  Evidence: `rtk make sqlc-check` passed; generated `internal/infra/postgres/sqlcgen/billing_money_core.sql.go` and updated `models.go` are aligned with migration/query sources.

- [x] T008 [Checkpoint 3: constraint proof] Add data-model tests for fixed-scale money representation, ledger conservation, schema constraints, and ledger delta patterns. Cover decimal parser/formatter vectors, rounding-rule vectors, recomputation of `account_balances` from ledger deltas plus active holds, no posted ledger money-field mutation path, negative constrained-text cases, non-negative balance invariants, `available = settled - reserved`, settlement/evidence/idempotency uniqueness, and disallowed ledger deltas.
  Depends on: T001, T002, T003.
  Proof: `rtk go test ./...` and `rtk make test-integration`.
  Evidence: `rtk go test ./...` passed; `rtk make test-integration` passed with money parser/formatter/rounding vectors, schema constraint negatives, ledger conservation recompute, immutable posted ledger money fields, and uniqueness checks.

- [x] T009 [Checkpoint 3: idempotency proof] Add data-model tests for durable idempotency and stored outcomes, covering same key/fingerprint replay, changed-fingerprint conflict, stored failure replay, one stored outcome per idempotency record, and no money mutation on conflict for reserve, finalize, write-off, reversal/compensation, top-up evidence, migration import, and reconciliation correction shapes.
  Depends on: T001, T002, T006, T007.
  Proof: `rtk make test-integration`.
  Evidence: `rtk make test-integration` passed with durable idempotency replay/conflict, stored failure replay, one-outcome uniqueness, and conflict/no-ledger-mutation coverage across required money operation kinds.

- [x] T010 [Checkpoint 3: reconciliation proof] Add reconciliation data-model tests for stale reservation case dedupe, every approved duplicate-open-case lineage key, leased-case `FOR UPDATE SKIP LOCKED` claiming, resolution ledger linkage, duplicate payment evidence, changed-fingerprint evidence conflicts, and missing inference evidence lineage.
  Depends on: T004, T006, T007.
  Proof: `rtk make test-integration`.
  Evidence: `rtk make test-integration` passed with reconciliation dedupe, `FOR UPDATE SKIP LOCKED`, payment evidence duplicate/conflict, missing inference lineage, and legacy import mismatch coverage.

- [x] T011 [Checkpoint 3: legacy import proof] Add legacy import data-model tests proving import batch uniqueness, per-account import uniqueness, import fingerprint uniqueness, parity mismatch linkage to `legacy_balance_import_id`, explicit `migration_import` or correction ledger linkage, and no live balance read path that consumes legacy balance/rate fields.
  Depends on: T005, T006, T007.
  Proof: `rtk make test-integration`.
  Evidence: `rtk make test-integration` passed with legacy import batch/account/fingerprint uniqueness, `migration_import` ledger linkage, `legacy_balance_import_id` mismatch dedupe, and live balance readback excluding legacy balance/rate evidence.

- [x] T012 [Checkpoint 4: concurrency proof] Add concurrency tests for account-row locking and terminal uniqueness: same-account reserve races cannot make available balance negative, finalize replay creates one terminal outcome and one charge effect, finalize/write-off races allow only one terminal path, duplicate top-up evidence credits once, and lock-timeout/deadlock classifications are observable at the data-model boundary.
  Depends on: T008, T009, T010, T011.
  Proof: `rtk make test-integration` and `rtk make test-race`.
  Evidence: `rtk make test-integration` passed with same-account lock races, terminal finalize/write-off uniqueness, duplicate top-up evidence uniqueness, lock timeout SQLSTATE `55P03`, and deadlock SQLSTATE `40P01`; `rtk make test-race` passed.

- [x] T013 [Checkpoint 4: performance proof] Add benchmark or explain-plan evidence for reserve, finalize, write-off, top-up evidence application, reconciliation claim, and support readback data paths, proving O(1) lookups by declared account/operation/evidence/idempotency keys plus account-row contention measurement for same-account workloads.
  Depends on: T006, T007, T008, T009, T010, T011, T012.
  Proof: `rtk go test -tags=integration -run '^$' -bench 'BenchmarkBillingMoneyCore' ./test/...` or an equivalently named benchmark command recorded in Evidence.
  Evidence: `rtk go test -tags=integration -run '^$' -bench 'BenchmarkBillingMoneyCore' ./test/...` passed: reserve 656724 ns/op, finalize 565387 ns/op, write-off 228235 ns/op, top-up evidence 217077 ns/op, reconciliation claim 398128 ns/op, support readback 130750 ns/op.

- [x] T014 [Checkpoint 5: final validation] Run the full data-model proof set and record exact results.
  Depends on: T001 through T013.
  Proof: `rtk make sqlc-check`; `rtk make migration-validate`; `rtk go test ./...`; `rtk make test-integration`; `rtk make test-race`; `rtk go test -tags=integration -run '^$' -bench 'BenchmarkBillingMoneyCore' ./test/...`; `rtk git diff --check`.
  Evidence: `rtk make sqlc-check` passed; `rtk make migration-validate` passed; `rtk go test ./...` passed with 702 tests in 14 packages; `rtk make test-integration` passed from Go cache and supplemental `rtk go test -tags=integration -count=1 ./test/...` passed with 7 tests in 1 package; `rtk make test-race` passed from Go cache and supplemental `rtk go test -race -count=1 ./...` passed with 702 tests in 14 packages; `rtk go test -tags=integration -run '^$' -bench 'BenchmarkBillingMoneyCore' ./test/...` passed with reserve 865768 ns/op, finalize 656078 ns/op, write-off 4849535 ns/op, top-up evidence 638056 ns/op, reconciliation claim 663302 ns/op, support readback 280590 ns/op; `rtk git diff --check` passed and staged generated SQLC output also passed `rtk git diff --cached --check -- internal/infra/postgres/sqlcgen`.

- [x] T015 [Checkpoint 5: closeout] Update ledger-owned progress/evidence and task-local closeout after validation, including `specs/billing-money-core/tasks.md` checkboxes/evidence and `specs/billing-money-core/spec.md` `Outcome` only if implementation proof justifies a narrower or final data-model outcome claim.
  Depends on: T014.
  Proof: manual closeout read confirming evidence matches the changed surface and no implementation-only workflow artifact was created.
  Evidence: Manual closeout read passed: all T001-T015 checkboxes are complete with evidence, `specs/billing-money-core/spec.md` Outcome records the implemented data-model slice and proof, changed surfaces are limited to migration/query/generated SQLC, money helper/tests, integration benchmarks, Makefile SQLC host compatibility, and ledger-owned spec/task closeout; no implementation-only workflow artifact was created.
