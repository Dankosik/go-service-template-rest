# Runtime Sequence And Failure Design

Status: review-ready
Date: 2026-06-02

## Account Import And Parity Before Migration

1. An import/backfill tool or worker reads an approved proxy balance snapshot
   outside the paid request path.
2. Billing creates or verifies `billing_accounts` using
   `account_scope_key=user:<proxy_user_id>`.
3. Billing creates `account_balances` and posts an immutable
   `migration_import` ledger entry for the derived USD atom amount.
4. Billing records `legacy_import_batches` and `legacy_balance_imports` with
   proxy source snapshot fingerprint, derived USD atoms, import fingerprint,
   and parity status.
5. Shadow parity compares billing settled/reserved/available and active
   exposure against proxy legacy balance/import evidence.
6. Migrated paid mode can be enabled only for accounts whose billing account,
   balance, import, parity, and reconciliation state pass the rollout gates.

Failure points:

- Missing proxy snapshot or non-canonical amount: import fails; account remains
  unmigrated.
- Parity mismatch: open or update `legacy_import_mismatch` reconciliation;
  account resolve returns `import_required` or `reconcile_required` for paid
  admission.
- Suspended/manual-review/closed account: paid admission fails closed.
- Import retry with same fingerprint returns stored outcome; changed fingerprint
  creates conflict or supersedes only through an explicit import batch state,
  not request-path mutation.

Side effects:

- Import is the only place this cutover writes initial customer balance.
- Account resolve and balance read do not create credit or mutate balance.

## Account Resolve

1. `gonka-proxy` calls
   `POST /internal/billing/v1/accounts/resolve` with scoped service JWT,
   contract version, deadline, trace/correlation ID, caller context, and
   represented user context.
2. `internal/infra/http` validates service auth, required
   `billing.accounts.resolve` scope, body shape, safe metadata, and deadline.
3. The account app service derives `account_scope_key=user:<subjectId>` and
   reads `billing_accounts`, `account_balances`, latest import/parity state,
   active reconciliation flags, and runtime/admission gate state.
4. The app returns canonical account ID, account scope key, state, import state,
   safe balance-read eligibility metadata, retryability, and failure class.
5. HTTP maps the app result to a bounded response or Problem without exposing
   bearer tokens, request bodies, prompts, completions, SSE chunks, secrets, or
   raw provider payloads.

Failure points:

- Missing/invalid JWT or route scope: `401`/`403`; app is not called.
- Represented user mismatch or unsupported account shape: fail closed.
- Account not found, not imported, suspended, manual review, active
  reconciliation, dependency not ready, or stale runtime gate: fail closed with
  explicit safe result code.
- Timeout after possible read acceptance: caller retries with the same trace and
  represented user context; no new money operation is minted because resolve is
  read-only.

## Balance Read

1. `gonka-proxy`, support tooling, or admin tooling calls the protected balance
   read route for one account scope.
2. HTTP validates service JWT and either `billing.balances.read` or
   `billing.admin.read`, depending on the route.
3. The app reads the account, current `account_balances`, active
   `usage_holds`, active `spending_microleases`, unresolved
   `microlease_child_debits`, relevant `reconciliation_cases`, latest import
   status, and worker/admission state.
4. The response returns settled, reserved, available, pending/import exposure,
   active usage hold exposure, active microlease exposure, balance version,
   stale operation flags, stale microlease flags, terminal lag flags,
   reconciliation backlog flags, manual-review flags, and safe correlation
   identifiers.

Failure points:

- Unknown account, import/parity not ready, suspended/manual-review account, or
  reconciliation-required state returns fail-closed readback status for paid
  admission.
- Worker/admission critical lag is included in the response and blocks migrated
  paid admission.
- Expired microleases remain reserved exposure until terminal, close, release,
  write-off, reversal, or reconciliation proof exists.

Side effects:

- None. Balance read does not repair or release money.

## Migrated Paid Admission

1. Proxy authenticates the public caller and obtains pricing/API-key
   attribution evidence. Pricing evidence must include immutable USD-compatible
   snapshot identity, fingerprint, policy version, decision time, selector or
   use-class context, and contract metadata.
2. Proxy resolves the billing account or uses a fresh accepted resolve result.
3. Proxy reads or obtains a billing-issued microlease through
   `/internal/billing/v1/microleases/issue`. Billing locks the account balance,
   validates account/import/admission/pricing state, reserves parent microlease
   exposure in Postgres, stores idempotency/outcome, and returns a replay-stable
   grant.
4. Before external execution, proxy commits a durable child debit and terminal
   obligation in proxy durable storage against the valid microlease grant.
5. Proxy admits external execution only if migrated cohort policy says:
   durable microlease capacity exists, debit cutoff has not passed, local
   backlog is not critical, old proxy money writer is disabled, and direct
   reserve fallback is disabled.
6. Optional memory or Redis prechecks can deny or shape traffic only over
   already reserved authority. They cannot authorize execution without the
   durable child debit and terminal obligation.

Failure points:

- Microlease issue denial, stale pricing, account not ready, no capacity,
  stale admission control, worker lag, missing service auth, or timeout without
  successful same-identity readback: proxy fails paid admission closed.
- Child debit conflict or missing terminal obligation: proxy fails before
  external execution.
- Direct reserve fallback enabled for migrated cohort: proxy fails admission
  closed and must not call local reserve/deduct.
- Possible accepted timeout on microlease issue: proxy retries or reads back
  with the same idempotency key/fingerprint; it must not create a new money
  operation.

Side effects:

- Billing side effect before execution is parent microlease reserve only.
- Proxy side effect before execution is child debit plus terminal obligation.
- No proxy local `balanceNgonka` or `BalanceTransaction` money write occurs for
  migrated cohorts.

## Generic Usage Reserve Convergence

1. A caller submits a usage reserve command with account scope, stable
   idempotency key, request fingerprint, usage operation identity, pricing
   lineage, max authorized USD exposure, and represented user context.
2. For `svc:gonka-proxy` migrated cohorts, the command must include
   `authorityMode=billing_microlease_with_proxy_child_debit` or the exact
   generated enum chosen in OpenAPI, plus microlease ID, child debit identity,
   child cap, owner/fence/generation, and request basis fingerprint.
3. The app validates the referenced billing microlease and records or reads back
   the `usage_operations` and operation outcome identity without reserving
   account balance a second time.
4. For non-migrated/future authority modes, the route must either be disabled
   or explicitly rejected until a later approved spec allows a direct
   account-balance `usage_holds` reserve path.

Failure points:

- Missing child debit lineage for migrated proxy reserve: fail closed.
- Missing idempotency or changed fingerprint: stored conflict.
- Stale or denied microlease authority: fail closed, no direct balance reserve.

Side effects:

- Migrated reserve stores operation/readback lineage against already reserved
  microlease authority. It does not create a second customer-money hold.

## Terminal Finalize, Write-Off, And Reversal

1. Proxy submits terminal evidence through Redpanda as the normal migrated path
   after execution, or through synchronous HTTP for retry/repair paths that need
   immediate readback.
2. The terminal input references the prior usage operation when one exists, the
   microlease ID, child debit identity, terminal fingerprint, canonical final
   charge or write-off amount in USD atoms, pricing lineage, qualified
   inference or execution evidence ID, and safe terminal metadata.
3. The app validates idempotency, child cap, parent microlease cap, account
   state, terminal kind, terminal fingerprint, and privacy-safe metadata.
4. The Postgres repository locks the child debit and affected balance rows,
   records qualified inference evidence when present, posts ledger effects,
   updates child debit, usage terminal outcome, operation outcome, microlease
   settlement totals, and outbox facts in one local transaction.
5. HTTP returns replay-stable outcome; worker commits Redpanda offset only after
   the store applies or quarantines the event.

Failure points:

- Terminal total exceeds child cap: open or update reconciliation, store safe
  outcome/conflict, do not overcharge customer.
- Parent cap exceeded or unresolved child lineage: open reconciliation and
  preserve reserved exposure until repair.
- Missing inference evidence where required: write-off or
  `reconcile_required` according to operation kind and evidence class.
- Duplicate same fingerprint: return stored outcome.
- Same key or event identity with changed fingerprint: conflict/quarantine.
- DB commit succeeds but HTTP response fails: stored outcome is authoritative;
  caller must read back.
- DB commit succeeds but outbox publish later fails: outbox retry owns forward
  recovery; the ledger effect is already authoritative.

Side effects:

- Final charge is capped by child debit and parent microlease authority.
- Write-off and reversal are explicit ledger/reconciliation effects, never
  silent balance edits.

## Operation And Admin Readback

1. Caller submits operation readback by billing operation ID, usage operation
   ID, idempotency key, microlease ID, child debit ID, terminal outcome ID, or
   reconciliation case link as allowed by the generated contract.
2. HTTP enforces `billing.operations.read`, `billing.usage.read`,
   `billing.microleases.read`, `billing.reconciliation.read`, or
   `billing.admin.read` according to route.
3. App validates account binding and reads idempotency, operation outcome,
   usage/microlease/child debit, ledger, reconciliation, and import state.
4. Response returns operation kind, state, stored outcome or conflict,
   retryability, account scope, safe pricing/execution lineage, failure class,
   reconciliation link, and safe ledger identifiers.

Failure points:

- Account binding mismatch: `403` or fail-closed problem, no data leak.
- Unknown operation: safe not-found result, not a new operation.
- Ambiguous timeout: caller retries or reads back by same operation identity;
  new operation IDs are prohibited.

## Worker Runtime

1. `cmd/billing-worker` loads config and exits cleanly when worker runtime is
   disabled.
2. When enabled, bootstrap validates Postgres, service auth, Redpanda,
   microlease config, admission-control staleness, and Redis-not-authority
   policy.
3. Bootstrap wires concrete tasks for terminal consumer, checkpoint consumer,
   close consumer, inbox retry, outbox relay, stale reconciliation, and
   admission-control renewal.
4. Worker sets readiness only after dependency probes pass.
5. Each task uses bounded concurrency, context cancellation, safe telemetry
   labels, idempotent store operations, and explicit retry/backoff.
6. Shutdown cancels tasks, waits inside configured grace, marks readiness false,
   and leaves uncommitted events unacknowledged for retry.

Failure points:

- Enabled worker with no-op task wiring: not production ready; readiness must
  fail for migrated paid cohorts.
- Redpanda unavailable or consumer lag critical: admission controls move to
  strict or fail-closed and proxy admission fails closed.
- Inbox conflict or quarantine: reconciliation readback exposes the issue and
  admission gates block affected scopes.

## Rollback And Failback

1. Before migrated mode, shadow/no-spend and internal cohort modes prove parity
   while old proxy writers are still controlled.
2. Migrated mode requires old proxy money writers disabled for the account
   scope and no direct reserve fallback.
3. Rollback cannot re-enable proxy-local money writes for already migrated
   scopes as a silent failback.
4. Rollback either fails paid admission closed or allows only already minted,
   valid microleases until debit cutoff/cap while old proxy writers remain
   disabled.

Failure points:

- Need to restore proxy-local writes for migrated accounts: reopen
  specification.
- Missing operator readbacks for lag, stale exposure, import parity, or
  reconciliation: rollout cannot proceed.
