# L1 Billing-Service Current State

Lane question: What billing-service contract, migration, handler, worker, and
config surfaces already exist after the completed microlease work, and what gaps
remain for account resolve, balance read, and usage lifecycle authority?

## Findings

- OpenAPI currently exposes internal microlease routes and operation readback:
  `POST /internal/billing/v1/microleases/issue`,
  `/microleases/readback`, `/microleases/close`, and
  `/operations/readback` in `api/openapi/service.yaml:112-260`.
- The existing OpenAPI does not expose account resolve, balance read, generic
  usage reserve, usage finalize, usage write-off, or usage reversal routes for
  the wider proxy cutover. The schema set around
  `api/openapi/service.yaml:404-680` is microlease/readback specific.
- HTTP generated handlers have a microlease service interface in
  `internal/infra/http/handlers.go:16-29`, but the generated route handlers
  return `503` when `h.microleases == nil`
  (`internal/infra/http/handlers.go:98-160`).
- Current service bootstrap constructs the HTTP router with `Health`, `Ping`,
  and `ReadinessGate` only. It does not provide a concrete `Microleases`
  implementation (`cmd/service/internal/bootstrap/run.go:151-168`).
- No non-test concrete app-level `MicroleaseService` implementation was found.
  Source search found the HTTP interface, tests/fakes, and the repository, but
  not bootstrap wiring from handler to repository/app logic.
- The money core migration creates `billing_accounts`,
  `account_balances`, `idempotency_records`, `operation_outcomes`,
  `usage_operations`, `usage_holds`, terminal outcome, qualified inference,
  top-up/payment, reconciliation, import, and immutable ledger structures
  (`env/migrations/000003_billing_money_core.up.sql:1-562`).
- The microlease migration adds microlease operation kinds/effects, event inbox,
  outbox, admission controls, `spending_microleases`,
  `microlease_child_debits`, `microlease_checkpoints`, and microlease-linked
  reconciliation fields (`env/migrations/000004_billing_microleases.up.sql:1-515`).
- The repository has durable issue, read, close, terminal settlement, checkpoint,
  admission-control, and outbox claim/publish methods in
  `internal/infra/postgres/microlease_repository.go:184-774`.
- The billing worker command exists, but current bootstrap passes
  `disabledRuntimeTasks()` to `microleaseworker.New`, making terminal consumer,
  checkpoint consumer, close consumer, inbox retry, outbox relay, stale
  reconciliation, and admission renewal no-op tasks
  (`cmd/billing-worker/internal/bootstrap/run.go:31-96`).
- Redpanda adapters exist for terminal consume/quarantine and outbox relay
  (`internal/infra/redpanda/consumer.go`, `internal/infra/redpanda/outbox.go`),
  but research did not find live worker bootstrap wiring that connects them to
  the billing worker command.
- Config defaults keep microleases and worker disabled, use fail-closed
  admission while disabled, require service auth/Redpanda/Postgres when enabled,
  and forbid Redis for first microlease runtime target
  (`internal/config/defaults.go:71-89`, `internal/config/validate.go:143-166`,
  `internal/config/validate.go:221-292`).

## Evidence Limits

- This lane inspected source, migrations, generated contracts, and current
  bootstrap wiring only.
- It did not run service startup, integration tests, SQL migrations, or live
  health checks.

## Open Points For Specification

- Define whether the broad cutover is an extension of the microlease API, a
  generic usage lifecycle API, or both.
- Define the concrete billing-service runtime module that backs HTTP account,
  balance, usage, microlease, readback, and reconciliation/admin surfaces.
- Define real worker task wiring and readiness behavior before rollout can be
  specified.
- Define whether top-up/payment tables remain untouched implementation context
  for this scope or need read-only references for balance import parity only.
