# Component Map

Status: review-ready
Date: 2026-06-02

## Billing-Service Contract And Transport

`api/openapi/service.yaml`

- Add the source-of-truth routes and schemas for account resolve, balance
  read, usage reserve/finalize/write-off/reversal, operation readback,
  reconciliation readback, admin ledger/readback, and any microlease schema
  extension needed by those resources.
- Preserve existing microlease issue/readback/close routes as the parent
  spend-authority surface for migrated proxy paid admission.
- Align every protected route with `ServiceBearerAuth` and `x-route-scopes`.
  `/internal/billing/v1/operations/readback` must require
  `billing.operations.read`.
- Do not add `/api/v1/usage/*` compatibility routes as a long-lived provider
  contract. Proxy adapters must move to `/internal/billing/v1`.

`internal/api/`

- Remains generated from OpenAPI.
- Must not be hand-edited or treated as contract authority.

`internal/infra/http/`

- Expand strict handlers from `MicroleaseService` only to a
  billing-authority handler surface that can route to account, balance, usage,
  microlease, operation, reconciliation, and admin app services.
- Keep transport validation, route contract IDs, body/path identifier rules,
  Problem mapping, service-auth scope checks, low-cardinality route labels, and
  privacy rejection at the edge.
- Handler-level `503` remains valid only when the route is disabled, runtime
  admission fails, or a required concrete service is intentionally absent. It
  must not be the enabled production behavior for migrated routes.

## Billing-Service App Layer

`internal/app/billingauthority` or equivalent new package

- Owns account resolve, balance read, generic usage lifecycle, operation
  readback, reconciliation readback, and admin readback use-case orchestration.
- Uses app-owned ports for account/balance/usage/microlease/reconciliation
  storage and runtime health.
- Does not import `internal/infra/http`, concrete Postgres packages, Redpanda
  adapters, or config loaders.
- Enforces account state, import/parity state, microlease authority mode,
  idempotency intent, fail-closed policy, privacy-safe metadata, and retryable
  versus terminal outcome classification.

`internal/app/microlease`

- Remains the app-owned decision package for issuing/replenishing microleases,
  active exposure, strict/fail-closed gates, terminal settlement decisions,
  close/expiry rules, rollout gate policy, and support-safe metadata checks.
- Must be consumed by the broader billing-authority app service instead of
  being bypassed by a second usage reserve path.

`internal/app/reconciliation`

- Extends from microlease-only repair decisions to account, usage operation,
  child debit, inbox, outbox, import/parity, and stale terminal decisions.
- Does not own money mutation directly. It chooses repair/reconciliation cases
  and calls the same app/repository commands that normal terminal paths use.

`internal/domain/money`

- Remains the shared exact USD atom vocabulary and parsing/formatting boundary.
- Do not introduce floating point or proxy `ngonka` customer-money arithmetic in
  new billing app code.

## Billing-Service Persistence

`env/migrations/`

- Existing money-core and microlease migrations already provide most source of
  truth state: `billing_accounts`, `account_balances`,
  `idempotency_records`, `operation_outcomes`, `usage_operations`,
  `usage_holds`, `usage_terminal_outcomes`, `ledger_entries`,
  `legacy_import_batches`, `legacy_balance_imports`,
  `reconciliation_cases`, `billing_event_inbox`, `billing_outbox`,
  `billing_admission_controls`, `spending_microleases`,
  `microlease_child_debits`, and `microlease_checkpoints`.
- Add migrations only for missing readback/import/runtime fields that planning
  and implementation prove cannot be derived from existing tables.
- Top-up/payment tables remain untouched except as explicitly excluded balance
  readback fields.

`internal/infra/postgres/queries/` and `internal/infra/postgres/sqlcgen/`

- Add SQLC query sources for account resolve, balance read, active exposure,
  usage lifecycle, operation lookup, reconciliation/admin readbacks, import
  parity readbacks, and worker claim/update loops.
- Keep SQLC generated code derived and regenerate through repository-owned
  targets when those sources change.

`internal/infra/postgres/`

- Add concrete repositories that map generated SQLC rows to app-owned types.
- Keep one short local transaction per money command.
- Lock `account_balances` or the specific operation/child debit rows where the
  invariant requires single-writer behavior.
- Do not call pricing, proxy, Redpanda, JWKS, or any external service while a
  money transaction is open.

## Billing-Service Worker And Events

`cmd/billing-worker/internal/bootstrap/`

- Replace `disabledRuntimeTasks()` with concrete task construction when the
  worker feature is enabled.
- Keep the command disabled when configured disabled. Enabled-but-no-op is not
  production ready.

`internal/app/microleaseworker`

- Keep the seven required roles:
  `terminal_consumer`, `checkpoint_consumer`, `close_consumer`, `inbox_retry`,
  `outbox_relay`, `stale_reconciliation`, and
  `admission_control_renewal`.
- Add task implementations through app ports and infra adapters; do not move
  business rules into the worker loop.

`internal/infra/redpanda`

- Keep terminal consumer and outbox relay adapter ownership here.
- Add checkpoint and close consumers with the same envelope, producer
  identity, fingerprint, quarantine, retry, and offset-commit discipline.
- Add concrete Redpanda client adapters only at the infra layer.

## Bootstrap And Config

`internal/config/`

- Add a broader balance/usage authority runtime gate, exact key names to be
  chosen during implementation, that defaults disabled and requires Postgres
  and service auth when enabled.
- Migrated paid authority additionally requires microlease runtime, Redpanda,
  worker runtime, admission controls, and Redis disabled as money authority.
- Preserve existing fail-closed microlease defaults.

`cmd/service/internal/bootstrap/`

- Wire concrete app services and repositories into `httpx.NewRouter` when the
  broader authority runtime is enabled and dependency admission passes.
- Keep readiness false for migrated paid cohorts until Postgres, service auth,
  Redpanda-dependent runtime gates, and startup admission are valid.

## Gonka-Proxy Read-Only Cutover Surfaces

These files are design evidence and later planning inputs only. This phase does
not edit `gonka-proxy`.

`src/services/billing/shared-balance-live.ts`

- Replace or adapt the old bearer-key `/api/v1/usage/*` bridge to the
  billing-service `/internal/billing/v1` contract.
- Remove `BILLING_SERVICE_AUTH_KEY` as production auth for migrated money
  authority and use scoped service JWT minting.

`src/services/billing/microlease/durable-microlease-allocator.ts`

- Remains the proxy durable child debit and terminal obligation owner before
  external paid execution.
- Memory cache remains deny/precheck only; repository commit plus terminal
  obligation is required before spend authority is returned.

`src/services/billing/microlease/migrated-cohort-policy.ts`

- Remains the migrated cohort fail-closed policy seam.
- `microlease_migrated` must keep `directReserveFallbackAllowed=false`.

`src/services/completions/shared/billing.ts`
and `src/services/completions/web-search/billing-guards.ts`

- Must stop using local reservation/deduction/write paths for migrated cohorts.
- Completion and web-search paid admission must fail closed when billing
  microlease/child-debit authority, readback, or worker health is missing.

`BalanceService`, `BalanceStateService`, `BalanceReservationsService`,
`BalanceTransaction`, and `User.balanceNgonka`

- Remain legacy cohort behavior, historical display, analytics, and import
  parity evidence only for migrated account scopes.
- They must not be live migrated money writers.

## Intentional Non-Touches

- Public OpenAI-compatible `/v1*` route ownership stays in `gonka-proxy`.
- Pricing catalog/source ownership stays in `pricing-service`; billing stores
  immutable pricing lineage only.
- API-key lifecycle and spend policy object ownership stay in
  `api-key-service`; callers still perform final spend/account/usage checks
  when API-key-service returns `spend_limit_check_required`.
- Top-up lifecycle, payment evidence, PSP details, refunds, and payment-service
  integration stay out of scope.
- Redis and process memory stay cache, limiter, projection, or deny/precheck
  surfaces only. They cannot mint customer-money authority.
