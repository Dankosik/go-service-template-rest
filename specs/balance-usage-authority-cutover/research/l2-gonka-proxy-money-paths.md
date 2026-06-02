# L2 Gonka-Proxy Money Paths

Lane question: What are the current `gonka-proxy` balance, usage,
reserve/finalize/write-off, fallback, and local money write paths that must map
to billing-service?

## Findings

- Proxy README already states the intended direction: `billing-service` is
  customer-money truth, proxy is facade/execution coordinator, pricing-service
  is pricing truth, and proxy must not become a second pricing or customer-money
  authority (`/Users/daniil/Projects/GonkaGate/gonka-proxy/README.md:35-104`).
- Prisma still stores user balance in proxy: `User.balanceNgonka` and
  `User.lockedRateUsd` are on `prisma/schema.prisma:164-172`.
- Proxy still stores local balance audit data in `BalanceTransaction`, including
  `type`, `amountNgonka`, USD cost fields, usage pricing basis, rate, and web
  search linkage (`prisma/schema.prisma:847-878`).
- Web search add-on holds reserve local balance transaction IDs, amount in
  ngonka, USD cost fields, rate, state, and release/consume metadata
  (`prisma/schema.prisma:747-777`).
- `BalanceService` still reads local balance and locked rate through
  `BalanceStateService` (`src/services/balance.service.ts:168-195`).
- `BalanceService` blocks local reservations, deductions, additions, and parent
  refunds only when shared-balance cutover is enabled; otherwise it still uses
  local reserve/deduct/add/refund paths (`src/services/balance.service.ts:245-620`).
- `BalanceReservationsService` is process-local memory state keyed by user ID,
  with reserve/release/evaluate logic in
  `src/services/billing/balance/balance-reservations.service.ts:22-153`.
- `BalanceStateService` performs local balance reads, row locks with
  `FOR UPDATE`, balance updates, locked-rate updates, and local
  `BalanceTransaction` creation
  (`src/services/billing/balance/balance-state.service.ts:27-194`).
- Completion runtime still reserves before execution via local balance
  reservation helpers and deducts after success via
  `balance.deductBalanceNgonka` (`src/services/completions/shared/billing.ts:188-319`,
  `src/services/completions/shared/billing.ts:478-590`).
- Public non-streaming, chat non-streaming, public streaming, chat streaming,
  and chat-history completion paths pass `runtime.balance` into reservation and
  finalize helpers, with strict deduction around successful paid work
  (`src/services/completions/public/strategy-non-streaming.ts:251-354`,
  `src/services/completions/chat/strategy-non-streaming.ts:434-605`,
  `src/services/completions/public/strategy-streaming.ts:757-853`,
  `src/services/completions/chat/strategy-streaming.ts:964-992`,
  `src/services/chat-history/completion/streaming-executor.ts:1009-1029`,
  `src/services/chat-history/completion/nonstreaming-executor.ts:289-365`).
- Interrupted streaming settlement can still deduct locally and log local usage
  (`src/services/completions/shared/strategy-runtime/streaming.ts:106-210`).
- Web-search pre-dispatch billing reserves a local cap through
  `attemptReservationNgonka`, blocks when shared-balance cutover is enabled, and
  releases local reservations on completion or abort
  (`src/services/completions/web-search/billing-guards.ts:41-239`).
- Web-search maintenance skips money-touching sweeps when shared-balance cutover
  is enabled, but otherwise uses local compensation/recovery paths
  (`src/services/completions/web-search/operation-maintenance.service.ts:113-177`).
- Proxy's shared-balance bridge posts to
  `/api/v1/usage/reservations`, `/api/v1/usage/finalize-requests`, and
  `/api/v1/account-effects/operator-adjustments`
  (`src/services/billing/shared-balance-live.ts:21-23`,
  `src/services/billing/shared-balance-live.ts:286-420`).
- The shared-balance bridge is configured by
  `BILLING_SHARED_BALANCE_CUTOVER_ENABLED`, `BILLING_SERVICE_URL`, and
  `BILLING_SERVICE_AUTH_KEY` in `src/plugins/billing.ts:16-45`.
- Proxy has a durable microlease allocator and migrated-cohort policy:
  `DurableMicroleaseAllocator` commits child debits through a durable repository
  before success, and the migrated-cohort decision disallows direct reserve
  fallback for `microlease_migrated`
  (`src/services/billing/microlease/durable-microlease-allocator.ts:112-216`,
  `src/services/billing/microlease/migrated-cohort-policy.ts:1-72`).
- Source search found proxy microlease allocator/cohort policy code referenced
  by tests and support files, not by live completion execution paths.
- `UsageService.logUsage` still persists local usage logs and only awaits
  ingestion for successful requests tied to API key spend limits
  (`src/services/usage.service.ts:17-64`).

## Evidence Limits

- This lane inspected local proxy code only.
- No proxy tests, request traces, production logs, database rows, or live
  endpoint calls were used.
- The search did not prove dead code exhaustively; it proved no obvious live
  completion-path import/use of the microlease allocator in the inspected tree.

## Open Points For Specification

- Decide whether proxy completion paths will call billing-service directly for
  usage lifecycle commands, consume billing-issued microleases, or use a combined
  model.
- Decide how migrated paid cohorts are selected and how legacy local
  `balanceNgonka`, local reservations, local deductions, and local
  `BalanceTransaction` writes are disabled for those cohorts.
- Decide how local usage logs and API-key spend-limit counters remain analytics
  only, or become readbacks derived from billing-service authority.
- Decide whether implementation scope includes cross-repo proxy edits or a
  separate proxy handoff.
