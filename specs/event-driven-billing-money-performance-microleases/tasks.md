# Event-Driven Billing Money Performance Microleases Tasks

Status: approved for implementation
Task ledger review: PASS
Implementation readiness: PASS
Date: 2026-06-01
Owner: orchestrator

## Goal Contract

Goal objective: Complete the approved durable billing-issued microlease architecture by executing this ledger from T001 through final validation.
Stopping condition: all required tasks are checked, every task evidence line is current, required proof passes or records a concrete blocker, and ledger-owned closeout updates are complete.
Read first: `AGENTS.md`, `docs/spec-first-workflow.md`, this `tasks.md`, `spec.md`, `workflow-plans/technical-design-review.md`, `design/`, `test-plan.md`, `rollout.md`, and `docs/build-test-and-development-commands.md`.
Do not change: the approved zero-unbacked-spend authority model, billing PostgreSQL customer-money authority, durable proxy child debit and terminal obligation before external paid execution, Redis absence from the first runtime target, no direct per-request reserve fallback for migrated paid cohorts, payment/top-up scope, pricing/API-key ownership boundaries, and the privacy exclusions listed in `spec.md`.
Progress log: after each checkpoint or task proof, update the task checkbox and `Evidence` line with the command or concrete blocker. Keep unrelated dirty work out of the implementation diff.
Blocked-stop rule: if a task requires memory-only or Redis-only spend, direct reserve fallback, weaker billing authority, weaker proxy durable lineage, broader service ownership, a missing technical-design decision, or pricing evidence that cannot support USD-compatible immutable snapshots, stop and record the blocker with the reopen target named by that task.

## Implementation Handoff

Consumes: approved `spec.md`, reviewed split `design/`, `workflow-plans/technical-design-review.md`, `test-plan.md`, `rollout.md`, and this ledger.
Task ledger review: PASS.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: none. Required proof obligations are executable ledger tasks.
Workflow-plan adequacy: local self-check PASS; no subagent spawned because the active tool policy allows subagents only on explicit user request.
Separate review/validation phase files: not expected. The implementation ledger carries task proof, final validation, and closeout.
Reopen target: planning for task coverage/order/proof gaps; technical-design for missing package, contract, data, worker, rollout, or validation context; specification for authority, scope, fallback, or ownership changes.

## Tasks

- [x] T001 [Checkpoint A] Refresh cross-service and current-contract evidence before changing sources of truth.
  Files: read-only evidence from `api/openapi/service.yaml`, `api/proto/service/v1/service.proto`, `env/migrations/000003_billing_money_core.up.sql`, `internal/infra/postgres/queries/billing_money_core.sql`, `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/_shared/v1/index.ts`, `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`, `/Users/daniil/Projects/GonkaGate/pricing-service/README.md`, `/Users/daniil/Projects/GonkaGate/pricing-service/internal/infra/http/pricing_handlers.go`, `/Users/daniil/Projects/GonkaGate/api-key-service/api/openapi/service.yaml`, and `/Users/daniil/Projects/GonkaGate/payments-service/api/openapi/service.yaml`.
  Depends on: none.
  Proof: record the exact inspected revisions or paths and confirm pricing can provide or attest USD-compatible immutable snapshot evidence; if false, stop and reopen `specification`.
  Evidence: 2026-06-01 `rtk git rev-parse --short HEAD` / targeted status and `rtk nl` reads inspected billing-service `0772a8a` with local modified/untracked listed contract/schema sources; gonka-proxy `c6d1f5e3` shared/billing internal-money TypeBox contracts; pricing-service `ad5e1ab` README and modified `internal/infra/http/pricing_handlers.go`; api-key-service `bbaf57d` OpenAPI; payments-service `e89aa3b` OpenAPI. Pricing evidence is sufficient to continue: README lines 122-128 require lineage-bearing billing flows to persist `pricingSnapshotId`, `snapshotFingerprint`, `policyVersion`, authoritative `decisionAt`, selector/use-class context or selector key, and contract metadata before minting money lineage; handler lines 253-277 and 310-326 expose the current GNK/USDT quote selector plus snapshot ID, fingerprint, policy version, normalized price, use classes, freshness, and selector fields. No pricing reopen blocker found.

- [x] T002 [Checkpoint A] Add protected microlease HTTP contract source and generated server bindings.
  Files: `api/openapi/service.yaml`, `internal/api/openapi.gen.go`, `internal/api/`, `internal/infra/http/openapi_contract_test.go`.
  Depends on: T001.
  Proof: `rtk make openapi-check`; run OpenAPI breaking compatibility against the PR base when `BASE_OPENAPI` or base refs are available.
  Evidence: 2026-06-01 `rtk make openapi-check` passed after regenerating and staging `internal/api/openapi.gen.go`; runtime contract test passed, Redocly lint/validate passed with one non-failing warning for unused `RouteContractId`. `BASE_OPENAPI`/base refs were not available, so OpenAPI breaking compatibility proof was not run.

- [x] T003 [Checkpoint A] Add event contract source, generated event DTOs, and proto drift tooling.
  Files: `api/proto/events/v1/*.proto`, generated event DTO package such as `internal/api/events/v1`, `Makefile`, `scripts/`, and event contract tests or fixtures.
  Depends on: T001.
  Proof: repository-owned proto lint/generate/drift target added by this task, generated DTO drift check passes, and event fixtures exclude raw prompts, completions, SSE chunks, tokens, secrets, raw payloads, and dynamic proof URLs.
  Evidence: 2026-06-01 `rtk make proto-check` passed. Added `api/proto/events/v1/microlease_events.proto`, deterministic generated DTOs in `internal/api/events/v1/microlease_events.gen.go`, Makefile `proto-generate`/`proto-drift-check`/`proto-check`, and privacy contract tests/fixture that reject raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event payloads, dynamic proof URLs, and sensitive request bodies.

- [x] T004 [Checkpoint A] Expand billing PostgreSQL schema, SQLC queries, and migration proof for microleases.
  Files: `env/migrations/*.sql`, `internal/infra/postgres/queries/*.sql`, `internal/infra/postgres/sqlcgen/*`, `test/*integration*_test.go`.
  Depends on: T001.
  Proof: `rtk make sqlc-check`; `rtk make migration-validate` or `rtk make docker-migration-validate`; targeted Postgres integration tests for microlease tables, constraints, row locks, inbox/outbox, admission controls, ledger effects, retention, and rollback.
  Evidence: 2026-06-01 added `env/migrations/000004_billing_microleases.{up,down}.sql`, `internal/infra/postgres/queries/billing_microleases.sql`, generated SQLC artifacts, and `test/billing_microleases_integration_test.go`. `rtk make sqlc-check` passed. `rtk make migration-validate` passed with Docker migration rehearsal applying `000001`-`000004`, downing `000004`, then re-applying `000004`. `rtk go test -tags=integration ./test -run '^TestBillingMicroleaseSchemaConstraintsAndReplayState$' -count=1` passed, covering microlease table constraints, row locks, inbox replay keys, admission controls, ledger effect checks, reconciliation lineage, and rollback through migration down/up proof.

- [x] T005 [Checkpoint B] Implement app-owned microlease and reconciliation behavior without infra imports.
  Files: `internal/app/microlease`, `internal/app/reconciliation`, `internal/domain/money`, app-owned port/type files, and focused app tests.
  Depends on: T002, T004.
  Proof: targeted unit/property tests for USD atom parsing, cap formulas, fail-closed budgets, idempotency replay/conflict, non-negative available balance, active exposure conservation, close release rules, child/parent cap enforcement, write-off/reversal/compensation decisions, and privacy-safe metadata.
  Evidence: 2026-06-01 added transport-free `internal/app/microlease` and `internal/app/reconciliation` decision logic plus focused tests. `rtk go test ./internal/app/microlease ./internal/app/reconciliation ./internal/domain/money` passed, covering USD atom vectors, cap formula and safety-floor reductions, fail-closed and strict gates, idempotency replay/conflict, property-style non-negative available/exposure conservation checks, close release and unresolved-reserve rules, child/parent cap reconciliation, expiry-without-proof no-release behavior, stale reconciliation decisions, and support-safe metadata rejection.

- [x] T006 [Checkpoint B] Implement Postgres repositories and transaction boundaries for issue, replenish, readback, close, terminal settlement, checkpoints, inbox/outbox, and admission controls.
  Files: `internal/infra/postgres`, `internal/infra/postgres/queries/*.sql`, `internal/infra/postgres/sqlcgen/*`, `test/*integration*_test.go`.
  Depends on: T004, T005.
  Proof: targeted repository and integration tests prove one short transaction per money command, account balance row locks, no external calls while holding a transaction, stored outcomes, duplicate/conflict behavior, rollback on failure, and over-child or over-parent reconciliation.
  Evidence: 2026-06-01 added `internal/infra/postgres/microlease_repository.go` with short transaction methods for issue/readback/close, terminal settlement, checkpoints, inbox/outbox, and admission controls, plus typed SQLC-backed helpers. `rtk go test ./internal/infra/postgres` passed. `rtk go test ./internal/infra/postgres ./test -tags=integration -run 'TestBillingMicroleaseRepositoryTransactions|^$' -count=1` passed, covering account balance row locks through money commands, stored issue/close outcomes, duplicate rollback, terminal rollback on over-child evidence, DB commit before inbox-applied state, outbox creation/claim/publish, close release, admission controls, and no external-callback transaction API.

- [x] T007 [Checkpoint B] Add fail-closed microlease configuration, bootstrap validation, dependency admission, and readiness gates.
  Files: `internal/config`, `env/config/default.yaml`, `env/.env.example`, `cmd/service/internal/bootstrap`, and config/bootstrap tests.
  Depends on: T005.
  Proof: targeted config and bootstrap tests reject absent or malformed caps, TTL, cutoff, terminal deadline, stale-age, reconciliation SLA, service-auth, Redpanda, and admission-control settings; first-rollout caps remain at or below the approved budget unless rollout risk acceptance is recorded.
  Evidence: 2026-06-01 added default-closed microlease, service-auth, and Redpanda config/default/env surfaces plus bootstrap readiness budget validation. `rtk go test ./internal/config ./cmd/service/internal/bootstrap` passed with 364 tests, covering absent or malformed dependent service auth, Redpanda, Redis-first-runtime rejection, cap/safety-floor ranges, TTL/cutoff/deadline/stale-age/SLA/admission-control settings, snapshot/default/env coverage, and first-rollout cap limits at the approved budget unless explicit rollout risk acceptance is recorded.

- [x] T008 [Checkpoint C] Wire protected HTTP handlers, auth scopes, Problem mapping, and low-cardinality route telemetry.
  Files: `internal/infra/http`, `cmd/service/internal/bootstrap`, `internal/api`, `internal/infra/telemetry`.
  Depends on: T002, T005, T006, T007.
  Proof: targeted HTTP tests for 401/403, body identifier placement, idempotency, ambiguous-timeout readback, status mapping, route labels, metrics labels, bounded Problems, and no sensitive payload leakage; `rtk make openapi-check`.
  Evidence: 2026-06-01 added service-auth middleware, route-scope gates, JWKS-backed service JWT verifier, protected microlease handler service port, request-envelope validation, and bounded Problem response mapping. `rtk go test ./internal/infra/http ./cmd/service/internal/bootstrap` passed with 251 tests, covering 401/403, service call suppression, body-carried identifiers, idempotency/readback identity forwarding, ambiguous-operation readback, status mapping for conflict/422/429/503/internal errors, route labels, metrics labels, and no sensitive payload leakage in Problems or metrics. `rtk make openapi-check` passed; Redocly still reports only the existing non-failing unused `RouteContractId` component warning.

- [x] T009 [Checkpoint C] Add event adapters and billing-side inbox/outbox mechanics.
  Files: `internal/infra/redpanda`, generated event DTO package, `internal/app/microlease` adapter ports, `internal/infra/postgres`, telemetry tests.
  Depends on: T003, T005, T006, T007.
  Proof: targeted adapter tests prove producer authenticity, event fingerprints, duplicate replay, changed-fingerprint conflicts, DB commit before offset commit, quarantine/redrive, bounded retry/backoff, support-safe failure metadata, and low-cardinality telemetry.
  Evidence: 2026-06-01 added `internal/infra/redpanda` terminal consumer and outbox relay adapters with deterministic event fingerprinting, producer authenticity checks, support-safe quarantine metadata, offset-commit discipline, bounded retry/backoff, and low-cardinality observer labels. `rtk go test ./internal/infra/redpanda` passed with 9 tests, covering accepted apply before offset commit, duplicate replay, changed-fingerprint conflict quarantine, invalid producer, fingerprint mismatch, retry without offset commit, outbox producer identity headers, outbox fingerprint proof, and retry scheduling without raw payload or identifier leakage.

- [x] T010 [Checkpoint C] Add explicit `cmd/billing-worker` lifecycle and worker orchestration.
  Files: `cmd/billing-worker`, worker bootstrap packages, `internal/infra/redpanda`, `internal/app/microlease`, `internal/app/reconciliation`, `internal/infra/postgres`, `internal/infra/telemetry`.
  Depends on: T006, T007, T009.
  Proof: worker tests cover readiness/dependency probes, signal-aware shutdown, bounded concurrency where row locks matter, terminal/checkpoint/close consumers, inbox retry, outbox relay, stale microlease/debit reconciliation, admission-control renewal/closure, and goroutine leak checks where existing package patterns use them.
  Evidence: 2026-06-01 added `internal/app/microleaseworker` orchestration and explicit `cmd/billing-worker` entrypoint. `rtk go test ./internal/app/microleaseworker ./cmd/billing-worker/...` passed with 4 worker lifecycle tests across 3 packages, covering dependency readiness probes, required terminal/checkpoint/close/inbox-retry/outbox-relay/stale-reconciliation/admission-control task roles, signal/context-aware shutdown, terminal task concurrency capped for row-lock-sensitive work, safe worker labels, and goleak verification.

- [x] T011 [Checkpoint D] Implement billing close, expiry, reconciliation, and operator-safe readback behavior end to end.
  Files: `internal/app/reconciliation`, `internal/app/microlease`, `internal/infra/postgres`, `internal/infra/http`, `cmd/billing-worker`, support-safe docs or readback tests as needed.
  Depends on: T008, T010.
  Proof: targeted tests prove expiry alone does not release money, close releases only proven unallocated capacity, unresolved child cap stays reserved, stale exposure opens or updates reconciliation within the approved SLA, and operator notes/audit metadata stay support-safe.
  Evidence: 2026-06-01 added close/readback adapter tests and unsafe-operator-metadata rejection on protected microlease commands while preserving app-owned close/expiry/reconciliation decisions. `rtk go test ./internal/app/microlease ./internal/app/reconciliation ./internal/infra/http ./internal/app/microleaseworker ./internal/infra/postgres` passed with 194 tests, covering expiry-without-proof no release, close release only for proven unallocated capacity, unresolved child cap remaining reserved, reconciliation case decisions within SLA, protected close/readback body identity forwarding, and support-safe operator metadata rejection. `rtk go test ./test -tags=integration -run 'TestBillingMicroleaseRepositoryTransactions|TestBillingMicroleaseSchemaConstraintsAndReplayState' -count=1` passed with 2 integration tests covering repository/schema close and replay behavior.

- [x] T012 [Checkpoint D] Implement or coordinate `gonka-proxy` durable allocator, terminal obligation, checkpoint/close, and migrated-cohort fallback disablement.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy` durable grant/debit/terminal/checkpoint surfaces, proxy contract adapters, proxy tests/benchmarks, and bridge/removal controls.
  Depends on: T002, T003, T008, T009.
  Proof: proxy evidence proves grant persistence before spend, single-writer row-lock or CAS child allocation, terminal obligation before external execution, crash/restart recovery, memory cache rebuild from durable rows, no memory-only or Redis-only spend, direct reserve fallback disabled for migrated cohorts, no old proxy-local money writer for migrated cohorts, and privacy-safe local rows.
  Evidence: 2026-06-01 added `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/microlease/durable-microlease-allocator.ts`, migrated-cohort policy, and targeted proxy tests. `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts` passed with 7 tests, proving durable repository commit is required before spend authority is returned, terminal obligation must be committed, memory cache is only a deny/precheck and cannot authorize spend alone, crash/restart cache rebuild comes from durable grant rows, migrated paid cohorts disable direct reserve fallback, shadow/legacy modes avoid dual writers, and local child debit rows reject unsafe metadata. `rtk bun run typecheck` was attempted after fixing the new-file type error; it remains blocked by unrelated pre-existing errors in `src/errors/normalization/api/api-error-formatter.ts`, `src/middleware/anthropic-api-key-credential-extractor.ts`, dashboard activity-usage files, `src/services/admin-user.service.ts`, `src/services/api-keys/api-key-mutation-parser.ts`, and `src/utils/max-tokens.ts`.

- [x] T013 [Checkpoint D] Integrate pricing snapshot and API-key attribution semantics without moving ownership.
  Files: billing request/adapter/app surfaces, `/Users/daniil/Projects/GonkaGate/gonka-proxy` integration surfaces, and read-only provider contract evidence from pricing-service and api-key-service.
  Depends on: T001, T008, T012.
  Proof: tests or recorded contract proof show pricing snapshot identity, fingerprint, policy version, decision time, selector/use-class context, and contract metadata are persisted before money lineage; `spend_limit_check_required` still leaves final spend/account/usage checks in the proxy/billing authority path.
  Evidence: 2026-06-01 tightened billing HTTP issue validation for immutable pricing snapshot identity, fingerprint, policy version, decision time, selector/use-class context, and contract metadata before service handling. Added proxy microlease pricing/API-key lineage context preserving API-key policy authority separately from final spend authority. `rtk go test ./internal/infra/http ./internal/app/microlease ./internal/infra/postgres` passed with 190 tests. `rtk go test ./test -tags=integration -run 'TestBillingMicroleaseRepositoryTransactions' -count=1` passed, re-proving repository persistence of pricing snapshot fields before stored money lineage. In `/Users/daniil/Projects/GonkaGate/gonka-proxy`, `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts` passed with 10 tests, including `spendLimitCheckRequired=true` remaining policy context while final spend authority stays `billing_microlease_with_proxy_child_debit`.

- [x] T014 [Checkpoint E] Add default-closed rollout controls, shadow/parity, cohort gates, rollback/failback, and operational readbacks.
  Files: `env/config/default.yaml`, `env/.env.example`, `cmd/service/internal/bootstrap`, `cmd/billing-worker`, `internal/config`, billing/proxy gate code, and `docs/` runbook or operator documentation when needed.
  Depends on: T011, T012, T013.
  Proof: rollout tests or recorded dry-run evidence cover inert expand, shadow/no-spend mode, balance/exposure parity, conservative internal cohort enablement, no dual writer, old proxy writer disablement, no direct reserve fallback for migrated cohorts, rollback that fails paid admission closed or uses only already-minted valid microleases until cutoff/cap, and operator-visible lag/stale/reconciliation gates.
  Evidence: 2026-06-01 added app-owned rollout gate policy and `docs/microlease-rollout-runbook.md`. `rtk go test ./internal/app/microlease ./internal/config ./cmd/service/internal/bootstrap ./internal/app/microleaseworker` passed with 383 tests, covering default-closed inert expand, shadow/no-spend parity, internal cohort gates, migrated no-dual-writer/no-direct-reserve-fallback requirements, rollback using only already-minted valid microleases until cutoff, critical lag/stale/reconciliation operator gates, and fail-closed behavior. Proxy targeted proof was re-run with `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts`, passing 10 tests for migrated-cohort fallback and lineage policy.

- [x] T015 [Checkpoint E] Prove privacy and security across APIs, events, telemetry, durable rows, fixtures, and workflow artifacts.
  Files: API/event schemas, HTTP/Redpanda/Postgres adapters, logging/metrics/tracing code, tests, fixtures, and workflow artifacts touched by implementation.
  Depends on: T008, T009, T012, T014.
  Proof: targeted privacy assertions plus `rtk make go-security` and `rtk make secret-scan` or Docker equivalents prove no raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event payloads, dynamic proof URLs, sensitive request bodies, account IDs, or raw request IDs leak into prohibited surfaces or high-cardinality labels.
  Evidence: 2026-06-01 ran a targeted privacy scan over implementation surfaces with `rtk rg -n "raw_prompt|raw prompt|completion text|sse chunk|Bearer |sk-|postgres://|mysql://|payment_secret|raw_provider_payload|raw event|request_body|dynamic proof|account_id|request_id" internal api env/migrations specs/event-driven-billing-money-performance-microleases docs/microlease-rollout-runbook.md`; matches were limited to policy text, generated/SQL identifiers, test-only sentinel strings, and documented prohibited-surface proof obligations, with no raw prompt/completion/SSE/provider payload, token, API key, DSN, payment secret, dynamic proof URL, or sensitive request body persisted in implementation code paths. Hardened gosec findings in USD atom parsing/formatting, validated JWKS URL handling, and event DTO generator permissions. `rtk go test ./internal/domain/money ./internal/infra/http` passed with 120 tests. `rtk go test ./internal/api/events/v1 ./internal/app/microlease ./internal/infra/http ./internal/infra/redpanda` passed with 119 tests. `rtk make go-security` passed: govulncheck reported code affected by 0 vulnerabilities and gosec reported 0 issues across 67 files. `rtk make secret-scan` passed: gitleaks scanned 175 commits / ~40.64 MB and found no leaks. Proxy targeted privacy/security proof was re-run with `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts`, passing 10 tests.

- [x] T016 [Checkpoint E] Prove performance budgets and reopen specification if the approved authority model cannot meet them.
  Files: billing benchmarks, proxy benchmarks under `/Users/daniil/Projects/GonkaGate/gonka-proxy`, performance fixtures, and benchmark documentation.
  Depends on: T011, T012, T014.
  Proof: benchmarks measure billing issue/replenish p95 under 100 ms and p99 under 250 ms, proxy durable child allocation p95 under 10 ms and p99 under 25 ms, active admission with and without memory precheck, cold replenishment p95 under 250 ms and p99 under 500 ms, account contention, terminal ingestion, checkpoint/close cadence, stale reconciliation scan cost, and first-token added latency p95 under 25 ms. If success requires unbacked memory/Redis spend, stop and reopen `specification`.
  Evidence: 2026-06-01 added `test/billing_microlease_performance_integration_test.go`, proxy allocator performance proof in `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts`, and `docs/microlease-performance-proof.md`. `rtk bash -lc 'go test ./test -tags=integration -run "^TestBillingMicroleasePerformanceBudgets$" -count=1 -v'` passed. Measured billing p95/p99: issue/replenish 1.541875ms/4.212583ms against 100ms/250ms, terminal ingestion 2.740375ms/3.122167ms, checkpoint cadence 1.2655ms/1.412541ms, close cadence 1.231416ms/1.277917ms, cold replenishment 1.49525ms/1.49525ms against 250ms/500ms, stale reconciliation scan 105.75us/107.25us, and account contention 30.935709ms/30.935709ms. In `/Users/daniil/Projects/GonkaGate/gonka-proxy`, `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts` passed with 2 tests and measured proxy p95/p99: active without memory precheck 0.004ms/0.155ms, active with memory precheck 0.003ms/0.019ms, first-token added latency 0.003ms/0.018ms, and cold replenishment 0.011ms/0.026ms. No Redis or unbacked memory spend was used; every successful proxy allocation still committed through the durable repository and received a terminal obligation before execution.

- [x] T017 [Final Validation] Run the repository-owned validation bundle for the changed surfaces.
  Files: no source files unless validation exposes a ledger-approved fix; update task evidence only after proof.
  Depends on: T002, T003, T004, T008, T010, T015, T016.
  Proof: at minimum `rtk make check`, `rtk make openapi-check`, `rtk make sqlc-check`, migration validation, targeted integration tests, worker/event tests, proxy proof commands from T012/T016, `rtk make go-security`, and `rtk make secret-scan`; run `rtk make check-full` or Docker/PR-parity equivalents when Docker/base refs are available.
  Evidence: 2026-06-01 final validation passed. `rtk make check` passed with golangci-lint reporting 0 issues and `go test ./...` passing across all packages. `rtk make openapi-check` passed; Redocly emitted the existing non-failing `RouteContractId` unused-component warning. `rtk make proto-check` passed. `rtk make sqlc-check` passed. `rtk make migration-validate` passed via docker migration rehearsal through 000004 up/down/up. `rtk go test ./test -tags=integration -run '^(TestBillingMicroleaseRepositoryTransactions|TestBillingMicroleaseSchemaConstraintsAndReplayState|TestBillingMicroleasePerformanceBudgets)$' -count=1 -v` passed with 3 integration tests. `rtk go test ./internal/api/events/v1 ./internal/app/microleaseworker ./internal/infra/redpanda ./cmd/billing-worker/...` passed with 17 tests across 5 packages. `rtk make go-security` passed with govulncheck code affected by 0 vulnerabilities and gosec 0 issues. `rtk make secret-scan` passed with gitleaks scanning 175 commits / ~40.64 MB and finding no leaks. In `/Users/daniil/Projects/GonkaGate/gonka-proxy`, `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts` passed with 12 tests and 33 assertions. Supplemental proxy `rtk bun run typecheck` still fails on pre-existing non-microlease TypeScript errors in `src/errors/normalization/api/api-error-formatter.ts`, `src/middleware/anthropic-api-key-credential-extractor.ts`, dashboard activity-usage files, `src/services/admin-user.service.ts`, `src/services/api-keys/api-key-mutation-parser.ts`, and `src/utils/max-tokens.ts`. `rtk make check-full` passed after staging the newly generated/module metadata artifacts required by its drift guards; Docker CI included module verification, guardrails, agent/skill checks, lint, unit/race/coverage, SQLC/OpenAPI/security/secret checks, integration tests, migration rehearsal, Docker image build, and Trivy container scan.

- [x] T018 [Closeout] Update ledger-owned closeout evidence and approved outcome records.
  Files: this `tasks.md`, `spec.md` `Validation`/`Outcome`, and any runtime docs/runbooks explicitly changed by T014.
  Depends on: T017.
  Proof: final `rtk git diff --check`; task evidence lines cite the fresh commands or blockers; `spec.md` records only privacy-safe validation evidence and outcome, with no raw payloads, secrets, prompts, SSE, DSNs, bearer tokens, API keys, raw event payloads, dynamic proof URLs, or sensitive request bodies.
  Evidence: 2026-06-01 updated `spec.md` Validation and Outcome with privacy-safe implementation evidence only. Final whitespace proof passed with `rtk git diff --check` in `/Users/daniil/Projects/GonkaGate/billing-service` and `/Users/daniil/Projects/GonkaGate/gonka-proxy`. Ledger T001 through T018 is checked with current evidence. No workflow-plan or workflow-plans files were updated during implementation.

## Task-Ledger Review

Review result: PASS.

Reviewed against:

- `spec.md` approved durable billing-issued microlease decisions and reopen conditions;
- split `design/` bundle for component ownership, sequences, data model, dependency graph, protected HTTP, and Redpanda events;
- `workflow-plans/technical-design-review.md` PASS gate and planning-input obligations;
- `test-plan.md` proof obligations;
- `rollout.md` rollout, rollback, mixed-version, and no-dual-writer gates;
- `docs/build-test-and-development-commands.md` repository validation entrypoints.

Coverage result:

- Protected OpenAPI work is represented by T002 and T008.
- Event/proto/worker work is represented by T003, T009, and T010.
- Postgres schema, SQLC, ledger effects, idempotency, inbox/outbox, and admission controls are represented by T004 and T006.
- Config, bootstrap, fail-closed budgets, and readiness are represented by T007.
- App-level money invariants and reconciliation semantics are represented by T005 and T011.
- Cross-repo proxy durable allocator, terminal obligation, checkpoint/close, and no-fallback obligations are represented by T012.
- Pricing and API-key attribution obligations are represented by T001 and T013.
- Rollout choreography is represented by T014.
- Privacy/security and performance proof are represented by T015 and T016.
- Final repository validation and closeout are represented by T017 and T018.

Implementation readiness: PASS.

Rationale: the ledger covers the accepted target-state scope without asking implementation to choose a new architecture, ownership model, contract source, data class, failure policy, rollout policy, or validation class. The pricing and performance reopen conditions are carried as explicit stop rules inside T001 and T016.
