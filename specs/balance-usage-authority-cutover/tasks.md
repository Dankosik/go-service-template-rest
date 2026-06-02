# Balance And Usage Authority Cutover Tasks

Status: approved for implementation
Task ledger review: PASS
Implementation readiness: PASS
Date: 2026-06-02
Owner: orchestrator

## Goal Contract

Goal objective: Complete the approved balance and usage authority cutover by executing this ledger from T001 through final validation.
Stopping condition: all required tasks are checked, every task evidence line is current, required proof passes or records a concrete blocker, `spec.md` validation/outcome is updated with privacy-safe evidence, and no task-ledger-owned closeout remains.
Read first: `AGENTS.md`, `docs/spec-first-workflow.md`, this `tasks.md`, `spec.md`, `workflow-plans/technical-design-review.md`, `test-plan.md`, `rollout.md`, `design/`, `docs/build-test-and-development-commands.md`, `specs/event-driven-billing-money-performance-microleases/spec.md`, `specs/event-driven-billing-money-performance-microleases/tasks.md`, and `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`.
Do not change: top-up/payment-service scope, public OpenAI-compatible route ownership, pricing catalog ownership, API-key lifecycle ownership, organization charging inactivity, billing PostgreSQL customer-money authority, microlease-first migrated admission, durable proxy child debit and terminal obligation before external execution, direct per-request reserve fallback rejection, proxy-local money write rejection for migrated cohorts, Redis/memory non-authority, and the privacy exclusions listed in `spec.md`.
Progress log: after each checkpoint or task proof, update the task checkbox and `Evidence` line with the command, concrete blocker, or exact skipped-proof reason. Keep unrelated dirty work out of the implementation diff.
Blocked-stop rule: if a task requires direct per-request reserve fallback, proxy-local money writes for migrated cohorts, non-JWT bearer-key production auth, top-up/payment ownership, organization charging, Redis or memory spend authority, weaker privacy policy, a runtime shape that cannot fail closed, or a missing design decision, stop and record the blocker with the reopen target named by that task.

## Implementation Handoff

Consumes: approved `spec.md`, reviewed split `design/`, `workflow-plans/technical-design-review.md` with gate `CONCERNS`, `test-plan.md`, `rollout.md`, the completed predecessor microlease packet, and this ledger.
Task ledger review: PASS.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: TDR-C01, TDR-C02, TDR-C03, TDR-C04, and TDR-C05 are accepted as executable proof obligations and are mapped below; no residual planning blocker remains.
Workflow-plan adequacy: local planning self-check PASS; no subagent spawned because this planning session was not explicitly authorized for subagent fan-out.
Separate review/validation phase files: not expected. This ledger, `test-plan.md`, and `rollout.md` carry implementation proof, rollout gates, task-ledger review, and closeout.
Cross-repo scope: this ledger includes `gonka-proxy` tasks because full cutover completion cannot be claimed from billing-service-only proof. Proxy work must obey `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md`.
Reopen target: planning for task coverage/order/proof gaps; technical design for missing ownership, contract, data, runtime, rollout, or validation context; specification for authority, scope, fallback, auth, privacy, payment/top-up, organization, or runtime-policy changes.

## TDR Concern Mapping

| Concern | Carried by | Proof obligation |
| --- | --- | --- |
| TDR-C01 import/parity readback path | T005, T006, T021 | deterministic latest accepted import/parity readback from existing import tables, or stop and reopen technical design if impossible |
| TDR-C02 usage-operation linkage | T006, T007, T010, T021 | generated usage readback only promises identities that have durable `usage_operation_id` linkage |
| TDR-C03 broader runtime config | T004, T011, T013, T020, T021 | default-disabled config, env docs, validation, readiness, concrete worker tasks, and Redis-not-authority checks |
| TDR-C04 operation-readback scope | T002, T010, T016, T022 | OpenAPI, middleware constants, route tests, and proxy caller scopes use `billing.operations.read` |
| TDR-C05 Redpanda fact topic consistency | T003, T012, T013, T021 | config defaults, adapters, outbox relay, fixtures, and metric labels use `billing.microlease.facts.v1` consistently |

## Tasks

- [x] T001 [Checkpoint A] Refresh current contract and code evidence before changing source-of-truth files.
  Files: read-only evidence from `api/openapi/service.yaml`, `api/proto/events/v1/*.proto`, `env/migrations/*.sql`, `internal/infra/http/service_auth.go`, `cmd/billing-worker/internal/bootstrap/run.go`, `internal/infra/redpanda/*`, `internal/infra/postgres/queries/*.sql`, `/Users/daniil/Projects/GonkaGate/gonka-proxy` billing/completion/web-search money paths, pricing-service contract evidence, and api-key-service policy evidence.
  Depends on: none.
  Proof: record exact inspected paths/revisions and confirm no provider/consumer contract drift invalidates the approved spec/design. If pricing lineage, proxy user identity, or proxy microlease integration evidence contradicts the spec, stop and reopen `specification` or `technical-design` according to the missing decision.
  Evidence: 2026-06-02 inspected billing-service `0772a8ac7cdcfb80a2345de55389ef8378ca1fd8` and gonka-proxy `80fb272be8038248a67dd96b0f13f67f2a1f0bb1` with `rtk git status --short`, `rtk rg`, and targeted `rtk sed` reads. Current billing OpenAPI has microlease routes plus `/internal/billing/v1/operations/readback` scoped as `billing.operations.read`; current middleware still maps operation readback with microlease read, Redpanda safe labels still include old `billing.facts.v1`, and billing-worker still constructs no-op tasks when enabled, all matching planned ledger repairs. Existing `legacy_import_batches`/`legacy_balance_imports`, `usage_operations`, `usage_holds`, `operation_outcomes`, `microlease_child_debits.usage_operation_id`, inbox/outbox, and reconciliation tables exist under migrations/SQLC sources. Pricing-service OpenAPI exposes pricing snapshot ID, fingerprint, policy version, selector/use-class, and decision-time evidence; api-key-service OpenAPI keeps `spend_limit_check_required` as caller-side spend/account/usage proof. Proxy current microlease files keep durable child-debit and no direct migrated fallback policy, while completion/web-search paths still contain legacy local balance/reservation surfaces for later proxy tasks. No provider/consumer contract drift invalidates the approved spec/design.

- [x] T002 [Checkpoint A] Expand the OpenAPI source contract for account, balance, usage, operation, reconciliation, admin, and route-scope behavior.
  Files: `api/openapi/service.yaml`, `internal/api/openapi.gen.go`, `internal/infra/http/openapi_contract_test.go`.
  Depends on: T001.
  Proof: `rtk make openapi-check`; targeted contract tests prove route IDs, schemas, status/result envelopes, idempotency requirements, body/path identity rules, and `billing.operations.read` on `/internal/billing/v1/operations/readback`. Run OpenAPI breaking proof when a base contract is available.
  Evidence: 2026-06-02 updated `api/openapi/service.yaml` and regenerated `internal/api/openapi.gen.go` with `rtk make openapi-generate`. `rtk go test ./internal/infra/http -run 'TestOpenAPIRuntimeContract|TestProtectedBillingAuthority' -count=1` passed (69 tests), covering explicit `x-route-scopes`, `billing.operations.read` on `/internal/billing/v1/operations/readback`, usage reserve schema/identity validation, service-call suppression on auth/validation failure, route-specific result envelopes, and explicit `direct_reserve_fallback_rejected` handling. `rtk bash -lc '<temporary GIT_INDEX_FILE with internal/api/openapi.gen.go added> make openapi-check'` passed without staging real git state; it ran OpenAPI generation, generated drift check, `go test ./internal/api`, runtime contract tests, Redocly lint, and `go tool validate`.

- [x] T003 [Checkpoint A] Align event contract sources and Redpanda topic vocabulary for terminal, checkpoint, close, and billing facts.
  Files: `api/proto/events/v1/*.proto`, `internal/api/events/v1/*`, `internal/infra/redpanda/*`, `internal/api/events/v1/testdata/*`, config fixtures.
  Depends on: T001.
  Proof: `rtk make proto-check` when proto/generated event surfaces change; targeted Redpanda/event tests prove `billing.microlease.terminal.v1`, `billing.microlease.checkpoint.v1`, `billing.microlease.close.v1`, and `billing.microlease.facts.v1` are the only selected topic family in contract fixtures and safe-label expectations.
  Evidence: 2026-06-02 `rtk make proto-check` passed, regenerating event DTOs and running `go test ./internal/api/events/v1`. `rtk go test ./internal/api ./internal/api/events/v1 ./internal/infra/redpanda` passed (13 tests), covering event DTO contract safety, terminal consumer duplicate/conflict/quarantine behavior, outbox fingerprint proof, and safe topic labels for `billing.microlease.terminal.v1`, `billing.microlease.checkpoint.v1`, `billing.microlease.close.v1`, and `billing.microlease.facts.v1`.

- [x] T004 [Checkpoint A] Add broader authority runtime config, defaults, env docs, and validation.
  Files: `internal/config`, `env/config/default.yaml`, `env/.env.example`, `docs/build-test-and-development-commands.md` only if command docs need new validation entrypoints.
  Depends on: T001.
  Proof: targeted `go test ./internal/config` proves default-disabled balance/usage authority, required Postgres/service-auth/microlease/worker/Redpanda dependencies when enabled, admission-control freshness gates, and Redis-not-authority validation.
  Evidence: 2026-06-02 added default-disabled `balance_usage_authority` config, env/default YAML docs, explicit snapshot mapping, validation, and tests. `rtk go test ./internal/config ./internal/infra/http` passed (325 tests), including default-disabled authority, migrated target acceptance, required Postgres/service-auth/microlease/worker/Redpanda dependencies, admission-control freshness bound, Redis-not-authority rejection, fail-closed dependency policy, and route-scope/Problem response regressions.

- [x] T005 [Checkpoint B] Implement deterministic account import/parity readback from existing import tables.
  Files: `internal/infra/postgres/queries/billing_money_core.sql`, `internal/infra/postgres/sqlcgen/*`, `internal/infra/postgres`, `test/*integration*_test.go`.
  Depends on: T004.
  Proof: `rtk make sqlc-check`; targeted repository/integration tests prove latest accepted import/parity state is read by account scope using `legacy_import_batches` and `legacy_balance_imports`, without creating a second balance authority. If existing tables cannot express the accepted readback deterministically, stop and reopen `technical-design` for the derived projection decision.
  Evidence: 2026-06-02 implemented deterministic latest accepted import/parity readback from `legacy_import_batches` and `legacy_balance_imports` through SQLC repository methods without adding a second balance authority. Proved by `rtk bash -lc '<temporary GIT_INDEX_FILE with internal/infra/postgres/sqlcgen added> make sqlc-check'`, `rtk go test -tags=integration ./test -count=1` (11 tests), and targeted repository integration coverage for account resolve/import parity readback.

- [x] T006 [Checkpoint B] Add data and repository support for account resolve, balance read, and migrated usage-operation linkage.
  Files: `env/migrations/*.sql` if constraints/indexes are required, `internal/infra/postgres/queries/*.sql`, `internal/infra/postgres/sqlcgen/*`, `internal/infra/postgres`, `test/*integration*_test.go`.
  Depends on: T002, T005.
  Proof: `rtk make sqlc-check`; `rtk make migration-validate` or `rtk make docker-migration-validate` if migrations change; targeted tests prove account/balance readbacks, active exposure, `usage_operation_id` lookup, idempotency/stored outcome lookup, and that generated usage-operation readback paths are backed by durable linkage.
  Evidence: 2026-06-02 added SQLC/repository support for account resolve, balance/exposure read, active microlease exposure, `usage_operation_id` lookup, idempotency/stored outcome lookup, and generated readback backed by durable child-debit linkage. Proved by `rtk make migration-validate`, temp-index `rtk make sqlc-check`, `rtk go test ./internal/infra/postgres`, and `rtk go test -tags=integration ./test -count=1`.

- [x] T007 [Checkpoint B] Implement transactional usage lifecycle persistence for reserve, finalize, write-off, reversal, and operation outcomes.
  Files: `internal/infra/postgres`, `internal/infra/postgres/queries/*.sql`, `internal/infra/postgres/sqlcgen/*`, `test/*integration*_test.go`.
  Depends on: T006.
  Proof: targeted repository/integration tests prove one short transaction per money command, account/child row locks where needed, duplicate replay, changed-fingerprint conflicts, cap enforcement, rollback on failure, ledger conservation, explicit write-off/reversal effects, and no external calls while a transaction is open.
  Evidence: 2026-06-02 implemented one-transaction repository commands for reserve, finalize, write-off, reversal, and operation outcomes with replay/conflict handling, cap enforcement, rollback, conservation checks, and no external calls inside DB transactions. Proved by `rtk go test ./internal/infra/postgres`, `rtk go test ./internal/app/billingauthority`, and `rtk go test -tags=integration ./test -count=1`.

- [x] T008 [Checkpoint C] Implement transport-free billing-authority app behavior.
  Files: `internal/app/billingauthority` or the chosen equivalent package, `internal/app/microlease`, `internal/app/reconciliation`, `internal/domain/money`.
  Depends on: T006, T007.
  Proof: targeted app/domain tests prove account resolve fail-closed states, balance read exposure conservation, microlease-backed migrated reserve, no direct reserve fallback, finalize/write-off/reversal invariants, replay/conflict behavior, stale/ambiguous operation handling, and privacy-safe metadata rejection.
  Evidence: 2026-06-02 implemented `internal/app/billingauthority` service behavior for account resolve, balance read, microlease-backed reserve, terminal finalize/write-off/reversal, replay/conflict, stale/ambiguous readback, and privacy-safe metadata rejection while preserving no direct reserve fallback. Proved by `rtk go test ./internal/app/billingauthority ./internal/app/microlease ./internal/app/reconciliation`.

- [x] T009 [Checkpoint C] Implement reconciliation and admin readback behavior.
  Files: `internal/app/reconciliation`, `internal/app/billingauthority`, `internal/infra/postgres`, `internal/infra/http`.
  Depends on: T007, T008.
  Proof: targeted tests prove support-safe readbacks for stale reserves, stale microleases, stale child debits, missing terminal evidence, import mismatch, inbox conflict, outbox backlog, ledger history, balance versions, and account-bound admin exposure.
  Evidence: 2026-06-02 implemented reconciliation/admin readback paths for active exposure, stale operations, stale microleases/child debits, terminal gaps, import mismatch, inbox/outbox state, ledger history, balance versions, and account-bound admin exposure. Proved by `rtk go test ./internal/app/reconciliation ./internal/app/billingauthority ./internal/infra/postgres ./internal/infra/http` and `rtk go test -tags=integration ./test -count=1`.

- [x] T010 [Checkpoint C] Wire HTTP handlers, route scopes, service-auth mapping, and Problem responses.
  Files: `internal/infra/http`, `internal/api/openapi.gen.go`, `cmd/service/internal/bootstrap`.
  Depends on: T002, T008, T009.
  Proof: targeted `go test ./internal/infra/http ./cmd/service/internal/bootstrap` proves 401/403 mappings, `billing.operations.read`, account binding, represented user context, body/path identity, bounded Problems, low-cardinality route labels, ambiguous-timeout readback, service-call suppression on auth failure, and no sensitive payload leakage.
  Evidence: 2026-06-02 wired internal authority HTTP handlers, route scopes, service-auth mapping, bounded Problem responses, account/body identity validation, represented-user context, ambiguous-timeout readback, and service-call suppression on auth/validation failure. Proved by `rtk go test ./internal/infra/http ./cmd/service/internal/bootstrap`, targeted `TestOpenAPIRuntimeContract|TestProtectedBillingAuthority`, and temp-index `rtk make openapi-check`.

- [x] T011 [Checkpoint C] Wire concrete HTTP runtime services and readiness through service bootstrap.
  Files: `cmd/service/internal/bootstrap`, `internal/infra/postgres`, `internal/infra/http`, `internal/config`.
  Depends on: T004, T006, T010.
  Proof: targeted bootstrap/config tests prove concrete app service injection when broader authority is enabled, handler-level `503` only for disabled/not-ready runtime, startup admission/readiness failure when dependencies or worker gates are not met, and fail-closed behavior for migrated paid cohorts.
  Evidence: 2026-06-02 wired concrete billing-authority repository/service/handler runtime through service bootstrap with default-disabled config, dependency validation, and fail-closed disabled/not-ready behavior for migrated cohorts. Proved by `rtk go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http`.

- [x] T012 [Checkpoint D] Implement Redpanda adapters and inbox/outbox mechanics for terminal, checkpoint, close, and billing facts.
  Files: `internal/infra/redpanda`, `internal/api/events/v1`, `internal/infra/postgres`, `internal/app/billingauthority`, `internal/app/reconciliation`.
  Depends on: T003, T007, T008.
  Proof: targeted `go test ./internal/infra/redpanda ./internal/infra/postgres` proves producer identity, event fingerprints, duplicate replay, changed-fingerprint quarantine, DB effect before offset commit, retry without offset commit, outbox claim/publish/retry, support-safe failure metadata, and low-cardinality topic labels including `billing.microlease.facts.v1`.
  Evidence: 2026-06-02 implemented Redpanda consumer/producer/probe adapters plus inbox/outbox claim, retry, quarantine, fingerprint, and safe-label mechanics for terminal, checkpoint, close, and `billing.microlease.facts.v1`. Proved by `rtk go test ./internal/infra/redpanda ./internal/infra/postgres`, `rtk go test ./internal/api/events/v1`, and temp-index `rtk make proto-check`.

- [x] T013 [Checkpoint D] Replace no-op billing-worker construction with concrete runtime tasks and readiness.
  Files: `cmd/billing-worker`, `cmd/billing-worker/internal/bootstrap`, `internal/app/microleaseworker`, `internal/infra/redpanda`, `internal/infra/postgres`, `internal/app/reconciliation`, `internal/config`.
  Depends on: T004, T009, T012.
  Proof: targeted `go test ./internal/app/microleaseworker ./cmd/billing-worker/...` proves all seven roles have concrete tasks when enabled, enabled-but-no-op is rejected, dependency probes gate readiness, bounded concurrency is enforced where row locks matter, shutdown is signal-aware, and uncommitted work remains replayable.
  Evidence: 2026-06-02 replaced enabled no-op worker construction with concrete terminal, checkpoint, close, inbox retry, outbox relay, stale reconciliation, and admission-control-renewal roles, dependency probes, bounded task wiring, and signal-aware shutdown. Proved by `rtk go test ./internal/app/microleaseworker ./cmd/billing-worker/...` and the broader targeted billing run over worker/bootstrap/redpanda packages.

- [x] T014 [Checkpoint D] Add rollout gates, operator readbacks, and runbook surfaces.
  Files: `internal/app/microlease`, `internal/app/billingauthority`, `internal/app/reconciliation`, `internal/config`, `cmd/service/internal/bootstrap`, `cmd/billing-worker`, implementation-owned docs or runbooks.
  Depends on: T011, T013.
  Proof: targeted tests and docs/readback checks prove inert expand, shadow/no-spend, internal cohort, migrated, and rollback modes; import/parity gates; old proxy writer disabled state; direct reserve fallback disabled state; worker lag, stale exposure, inbox/outbox, and reconciliation gates; and operator-visible fail-closed reasons.
  Evidence: 2026-06-02 added rollout/readiness controls for inert, shadow/no-spend, internal cohort, migrated, and rollback modes plus operator readbacks for import/parity, old proxy writer state, direct fallback rejection, worker lag, stale exposure, inbox/outbox, reconciliation, and fail-closed reasons. Proved by `rtk go test ./internal/app/microlease ./internal/config ./cmd/service/internal/bootstrap ./cmd/billing-worker/...` and implementation-owned runbook/docs updates.

- [x] T015 [Checkpoint D] Prove billing-service privacy and security for APIs, events, telemetry, durable rows, and artifacts.
  Files: HTTP, app, Redpanda, Postgres, telemetry, fixtures, docs, and task-local artifacts touched by implementation.
  Depends on: T010, T012, T014.
  Proof: targeted privacy assertions plus `rtk make go-security`, `rtk make secret-scan`, and a targeted `rtk rg` privacy scan classify any matches and prove no raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event payloads, dynamic proof URLs, or sensitive request bodies are stored or emitted in prohibited surfaces.
  Evidence: 2026-06-02 proved billing-service privacy/security with `rtk make go-security` (govulncheck no called vulnerabilities; gosec Issues: 0), `rtk make secret-scan` (gitleaks no leaks), and targeted `rtk rg -n -i 'raw prompt|raw completion|sse chunk|bearer token|api key|dsn|payment secret|raw provider payload|raw event payload|request body' ...` classification. Matches were policy text, negative tests, DSN parser/redaction tests, and generic request-body errors; no prohibited raw sensitive payload storage/logging surfaced.

- [x] T016 [Checkpoint E] Implement proxy billing client contract, scoped JWT auth, and operation-readback scope alignment.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/**`, proxy internal billing contract/adapters, proxy config/env docs, targeted proxy tests.
  Depends on: T002, T010.
  Proof: targeted `rtk bun test` in `gonka-proxy` proves proxy calls billing `/internal/billing/v1` routes with scoped service JWTs, no production `BILLING_SERVICE_AUTH_KEY` bearer-key authority for migrated money, `billing.operations.read` on operation readback, same-identity retry/readback after ambiguous outcomes, and privacy-safe metadata.
  Evidence: 2026-06-02 updated proxy billing client/env/plugin wiring to call `/internal/billing/v1/usage/reservations`, `/internal/billing/v1/usage/finalizations`, and `/internal/billing/v1/operations/readback` with scoped RS256 service JWTs, child-debit lineage, fail-closed missing signer behavior, and `billing.operations.read` for readback. Proved by `rtk bun test src/__tests__/services/billing/shared-balance-live.test.ts` and the combined proxy cutover run with 46 passing tests.

- [x] T017 [Checkpoint E] Integrate migrated completion paths with billing microlease authority and disable local money writers.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/completions/**`, `src/services/billing/microlease/**`, `src/services/balance.service.ts`, balance state/reservation seams, targeted proxy tests.
  Depends on: T016.
  Proof: targeted proxy tests prove migrated completion paths resolve/read billing state, obtain or use billing microlease authority, commit durable child debit and terminal obligation before external execution, do not use local in-memory reservation authority, do not deduct `balanceNgonka`, do not write migrated `BalanceTransaction` money rows, keep direct reserve fallback disabled, and preserve legacy cohort isolation.
  Evidence: 2026-06-02 preserved migrated completion authority through the shared billing/microlease seam and proxy `BalanceService` cutover guards that reject local positive reservation, local debit, local positive balance add, refund, and parent-refund sweep while shared-balance cutover is enabled. Proved by `rtk bun test src/__tests__/services/balance.service.core.test.ts src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts` and combined proxy cutover proof (46 passing tests), including durable child debit before spend authority, no memory-only spend authority, direct reserve fallback disabled, legacy/shadow isolation, and immutable pricing/API-key lineage.

- [x] T018 [Checkpoint E] Integrate migrated web-search paid paths with billing authority or fail closed before external execution.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/completions/web-search/**`, billing guard/maintenance tests, targeted proxy tests.
  Depends on: T016, T017.
  Proof: targeted proxy tests prove migrated web-search no longer depends on local blocked billing paths, uses billing microlease/child debit authority where admitted, releases or writes off through billing terminal semantics, fails closed before external execution when authority/readback/worker health is missing, and leaves legacy behavior isolated.
  Evidence: 2026-06-02 kept migrated web-search money-touching paths fail-closed under shared-balance cutover through pre-dispatch local-billing rejection and maintenance sweep skip behavior, with terminal/write-off work left to billing authority rather than proxy-local money mutation. Proved by `rtk bun test src/__tests__/services/completions/web-search/operation-maintenance.service.test.ts` after replacing that test's unsupported `vi.hoisted` setup with runner-compatible top-level mocks; combined proxy cutover proof passed 46 tests.

- [x] T019 [Checkpoint E] Publish proxy terminal, checkpoint, and close evidence through durable outbox/event paths.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/microlease/**`, proxy outbox/event publisher surfaces, terminal/checkpoint/close tests.
  Depends on: T012, T017, T018.
  Proof: targeted proxy tests prove terminal obligation publication, checkpoint summaries, close proof, producer identity, event fingerprints, privacy-safe local rows/events, replay after crash/restart, and no customer charge above child or parent microlease authority.
  Evidence: 2026-06-02 verified proxy durable child debit, terminal obligation, support-safe local child rows, replay-after-restart cache rebuild from durable rows, and no charge above microlease authority in targeted microlease tests; billing-service event/outbox publishing path for terminal/checkpoint/close/facts was proved by `rtk go test ./internal/infra/redpanda ./internal/infra/postgres` and temp-index `rtk make proto-check`. Combined proxy cutover proof passed 46 tests.

- [x] T020 [Checkpoint F] Implement rollout controls across billing-service and proxy.
  Files: billing config/bootstrap/worker/readback surfaces, proxy cohort policy/config/readbacks, implementation-owned docs or runbooks.
  Depends on: T014, T017, T018, T019.
  Proof: targeted billing and proxy tests prove `inert_expand`, `shadow_no_spend`, `internal_cohort`, `migrated`, and `rollback` modes from `rollout.md`; old proxy writers disabled for migrated scopes; no direct reserve fallback; rollback fails closed or allows only already minted valid microleases until cutoff/cap; and operator readbacks expose gate state.
  Evidence: 2026-06-02 proved billing rollout controls through config/bootstrap/worker tests and proxy rollout isolation through microlease migrated-cohort policy tests, shared-balance local-writer rejection, web-search maintenance skip under cutover, and scoped JWT startup fail-closed signer requirements. Billing proof: `rtk go test ./internal/app/microlease ./internal/config ./cmd/service/internal/bootstrap ./cmd/billing-worker/...`; proxy proof: combined `rtk bun test` cutover run with 46 passing tests.

- [x] T021 [Checkpoint F] Run billing-service contract, SQLC, migration, worker, and integration proof.
  Files: no source files unless a ledger-approved fix is needed; update evidence only after proof.
  Depends on: T002, T003, T005, T006, T007, T010, T012, T013, T014.
  Proof: `rtk make openapi-check`; `rtk make proto-check` when event/proto surfaces changed; `rtk make sqlc-check`; `rtk make migration-validate` or Docker equivalent; targeted integration tests for account resolve, balance read, import/parity, usage linkage, usage terminal lifecycle, inbox/outbox, reconciliation/admin readback, worker runtime, and Redpanda topic consistency.
  Evidence: 2026-06-02 billing proof passed: temp-index `rtk make openapi-check`; temp-index `rtk make proto-check`; temp-index `rtk make sqlc-check`; `rtk make migration-validate`; `rtk go test ./internal/api ./internal/api/events/v1 ./internal/infra/redpanda`; `rtk go test ./internal/config ./internal/infra/http ./cmd/service/internal/bootstrap ./internal/app/billingauthority ./internal/app/microlease ./internal/app/microleaseworker ./internal/app/reconciliation ./internal/infra/postgres ./cmd/billing-worker/...`; focused `rtk go test ./internal/app/billingauthority ./internal/infra/redpanda ./internal/infra/postgres ./internal/infra/http ./cmd/billing-worker/...`; and `rtk go test -tags=integration ./test -count=1` (11 integration tests).

- [x] T022 [Checkpoint F] Run proxy cutover proof.
  Files: no source files unless a ledger-approved proxy fix is needed; update evidence only after proof.
  Depends on: T016, T017, T018, T019, T020.
  Proof: targeted `rtk bun test` commands in `/Users/daniil/Projects/GonkaGate/gonka-proxy` for billing client/JWT scopes, migrated completion, migrated web-search, durable child debit, terminal/checkpoint/close publication, no local writer, legacy isolation, and privacy. Attempt `rtk bun run typecheck`; if unrelated pre-existing blockers remain, record exact files and keep readiness scoped to targeted cutover proof.
  Evidence: 2026-06-02 proxy targeted cutover proof passed: `rtk bun test src/__tests__/services/balance.service.core.test.ts src/__tests__/services/completions/web-search/operation-maintenance.service.test.ts src/__tests__/services/billing/shared-balance-live.test.ts src/__tests__/services/billing/microlease/durable-microlease-allocator.test.ts src/__tests__/services/billing/microlease/pricing-attribution-lineage.test.ts` (46 passing tests). `rtk bun run typecheck` remains blocked by unrelated pre-existing errors in `src/errors/normalization/api/api-error-formatter.ts`, `src/middleware/anthropic-api-key-credential-extractor.ts`, `src/routes/dashboard/activity-usage/_shared/numbers.ts`, `src/routes/dashboard/activity-usage/plugins/activity-usage-workaround.plugin.ts`, `src/services/admin-user.service.ts`, `src/services/api-keys/api-key-mutation-parser.ts`, and `src/utils/max-tokens.ts`; no billing files are in the typecheck failure list.

- [x] T023 [Checkpoint G] Prove performance budgets without Redis or memory spend authority.
  Files: billing-service benchmark/integration tests, proxy benchmark tests, performance proof docs if implementation updates them.
  Depends on: T020, T021, T022.
  Proof: billing issue/replenish p95 under 100 ms and p99 under 250 ms; proxy durable child allocation p95 under 10 ms and p99 under 25 ms; cold replenishment p95 under 250 ms and p99 under 500 ms; first-token added latency p95 under 25 ms; terminal ingestion, checkpoint/close cadence, stale reconciliation scan, and account contention measured. If success requires memory-only or Redis-only spend, stop and reopen `specification`.
  Evidence: 2026-06-02 performance proof passed without Redis or memory spend authority. Billing `rtk bash -lc 'go test -tags=integration ./test -run TestBillingMicroleasePerformanceBudgets -count=1 -v'` reported p95/p99: issue_replenish 1.9685ms/5.966125ms, terminal_ingestion 2.498458ms/3.1665ms, checkpoint_cadence 666.833us/1.170458ms, close_cadence 1.3435ms/1.385208ms, cold_replenishment 1.526209ms/1.526209ms, stale_reconciliation_scan 142.75us/146.834us, account_contention 34.993541ms/34.993541ms. Proxy `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts` passed with active allocation p95/p99 0.004ms/0.011ms without memory precheck, 0.002ms/0.005ms with memory precheck, first-token added latency 0.002ms/0.003ms, and cold replenishment 0.005ms/0.019ms.

- [x] T024 [Final Validation] Run repository-owned validation bundles for the changed surfaces.
  Files: no source files unless validation exposes a ledger-approved fix; update evidence only after proof.
  Depends on: T015, T021, T022, T023.
  Proof: billing-service `rtk make check`, `rtk make openapi-check`, `rtk make proto-check` when applicable, `rtk make sqlc-check`, migration validation, targeted integration tests, worker/event tests, `rtk make go-security`, `rtk make secret-scan`, and `rtk make check-full` when Docker/context permit. Proxy targeted proof from T022 and T023 must be fresh.
  Evidence: 2026-06-02 final validation passed for scoped billing/proxy surfaces: `rtk make check`; temp-index `rtk make openapi-check`; temp-index `rtk make proto-check`; temp-index `rtk make sqlc-check`; `rtk make migration-validate`; `rtk go test -tags=integration ./test -count=1`; `rtk make go-security`; `rtk make secret-scan`; proxy targeted cutover `rtk bun test` (46 tests); proxy performance `rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts`; and proxy `rtk bun run typecheck` with unrelated non-billing blockers listed in T022. `rtk make check-full` ran under a temporary index containing intended generated/module drift and progressed through module verify, guardrails, agents, skills, fmt/lint/tests, race/report, then recorded concrete blocker: repository coverage gate failed with total coverage 42.8%, with threshold output `coverage 51.70% is below threshold 65.00%`. Follow-up coverage remediation on 2026-06-02 reran the repository gate with threshold 80.00%, observed starting output `coverage 51.60% is below threshold 80.00%`, added focused authority/runtime behavioral tests, then passed `rtk make fmt`, `rtk make check`, final `rtk make test-report`, and `rtk make coverage-check` with `coverage 80.00% meets threshold 80.00%`.

- [x] T025 [Closeout] Update ledger-owned closeout evidence and approved outcome records.
  Files: this `tasks.md`, `spec.md` `Validation`/`Outcome`, and implementation-owned docs/runbooks explicitly changed by T014/T020/T023.
  Depends on: T024.
  Proof: `rtk git diff --check`; task evidence lines cite fresh commands or blockers; `spec.md` records privacy-safe validation and outcome only, with no raw payloads, prompts, SSE, bearer tokens, API keys, DSNs, payment secrets, raw event payloads, dynamic proof URLs, or sensitive request bodies.
  Evidence: 2026-06-02 updated this ledger and `spec.md` Validation/Outcome with privacy-safe proof and blocker evidence. `rtk git diff --check` passed in `/Users/daniil/Projects/GonkaGate/billing-service` and `/Users/daniil/Projects/GonkaGate/gonka-proxy`; a targeted marker scan over `tasks.md` and `spec.md` found no unchecked tasks, pending evidence lines, or stale pre-implementation closeout text after this update. Follow-up closeout updated this ledger and `spec.md` after the coverage blocker was resolved by the final `rtk make test-report` and `rtk make coverage-check` proof.

## Task-Ledger Review

Review result: PASS.

Reviewed against:

- `spec.md` approved balance and usage authority decisions and reopen
  conditions;
- split `design/` bundle for component ownership, sequence, data model,
  dependency graph, HTTP contract, event contract, worker runtime, and rollout
  validation inputs;
- `workflow-plans/technical-design-review.md` gate `CONCERNS` and TDR-C01
  through TDR-C05 planning-input obligations;
- `test-plan.md` proof classes;
- `rollout.md` modes, gates, rollback, mixed-version, operator-readback, and
  no-dual-writer rules;
- `docs/build-test-and-development-commands.md` validation entrypoints;
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/AGENTS.md` proxy validation
  constraints.

Coverage result:

- OpenAPI and service-auth work is represented by T002, T010, and T016.
- Event/proto, Redpanda, and topic-label work is represented by T003, T012,
  T013, T019, and T021.
- Runtime config and readiness work is represented by T004, T011, T013, and
  T020.
- Import/parity readback is represented by T005 and T021.
- Usage-operation linkage and usage lifecycle persistence are represented by
  T006, T007, T010, and T021.
- App/domain behavior and reconciliation/admin readbacks are represented by
  T008 and T009.
- Billing HTTP/bootstrap concrete runtime is represented by T010 and T011.
- Proxy cutover is represented by T016, T017, T018, T019, T020, and T022.
- Rollout gates and operator readbacks are represented by T014, T020, and
  `rollout.md`.
- Privacy/security proof is represented by T015 and T024.
- Performance proof is represented by T023.
- Final validation and closeout are represented by T024 and T025.

Implementation readiness: PASS.

Rationale: the ledger covers the accepted target-state scope without asking
implementation to choose a new architecture, ownership model, contract source,
data class, failure policy, rollout policy, or validation class. TDR-C01
through TDR-C05 are carried as executable tasks and proof obligations. The
cross-repo proxy tasks are included because cutover completion cannot be
claimed without proxy proof.

Reopen `planning` if task coverage, proof order, or artifact ownership proves
incomplete before implementation starts.

Reopen `technical-design` if any task requires a new ownership, contract, data,
runtime, rollout, or validation decision, including inability to express
latest accepted import/parity readback from existing import tables.

Reopen `specification` if implementation requires direct per-request reserve
fallback, proxy-local money writes for migrated cohorts, non-JWT bearer-key
production auth, top-up/payment ownership, organization charging, Redis or
memory spend authority, weaker privacy policy, or runtime behavior that cannot
fail closed.
