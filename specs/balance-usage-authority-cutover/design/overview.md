# Technical Design Overview

Status: review-ready
Date: 2026-06-02
Owner: orchestrator

## Design Goal

This bundle turns the approved `spec.md` into reviewable technical context for
the balance and usage authority cutover. It keeps `billing-service` as the USD
customer-money authority for migrated `gonka-proxy` paid cohorts, preserves the
approved durable microlease admission model, and avoids any direct per-request
account-balance reserve fallback for migrated proxy cohorts.

The design consumes:

- `specs/balance-usage-authority-cutover/spec.md`;
- completed research under `specs/balance-usage-authority-cutover/research/`;
- the completed microlease packet under
  `specs/event-driven-billing-money-performance-microleases/`;
- current billing-service OpenAPI, Postgres, service-auth, worker, and
  Redpanda surfaces;
- read-only `gonka-proxy` contract and money-path evidence.

## Chosen Approach

Billing-service exposes the canonical `/internal/billing/v1` contract through
`api/openapi/service.yaml`, generated `internal/api` bindings,
`internal/infra/http` adapters, transport-agnostic app services, and
Postgres-backed repositories. The existing microlease routes remain the parent
spend-authority surface for migrated proxy paid admission. New account,
balance, generic usage, operation, reconciliation, and admin readback surfaces
are added as billing-owned resources that converge on the same account,
microlease, usage operation, idempotency, ledger, stored-outcome, inbox/outbox,
and reconciliation state.

For migrated proxy paid cohorts, the customer-money hold is the billing-issued
microlease reserve in billing Postgres. The per-request authorization proof is
the proxy durable child debit and terminal obligation committed before external
execution. Generic usage reserve/finalize/write-off/reversal APIs must bind to
that lineage and stored outcome model; they must not create a hidden path that
reserves directly from account balance after microlease denial.

The service request path stays synchronous only where a caller needs an
admission, readback, or terminal command answer. Durable terminal, checkpoint,
close, outbox, inbox retry, stale repair, and admission-control renewal work is
owned by the explicit `cmd/billing-worker` runtime. No durable background loop
is hidden inside HTTP handlers.

## Selected Live Forks

### Account Creation And Import

Selected: migrated paid accounts are created or imported before migrated paid
traffic is enabled. Request-path account resolve is read-only and fail-closed.

Rejected: idempotent account bootstrap inside paid request-path resolve. It
would hide import/parity failures behind admission and make it harder to prove
when proxy `balanceNgonka` became historical for the account.

### Migrated Usage Reserve Semantics

Selected: `billing_microlease_with_proxy_child_debit` is the only accepted
authority mode for migrated proxy paid reserve semantics. A usage reservation
record may be created or read back for a migrated child debit only when the
request references a valid billing microlease and durable proxy child-debit
identity. It consumes already reserved parent microlease exposure rather than
locking account balance again.

Rejected: direct `usage_holds` reserve from account balance as a fallback after
missing, stale, or denied microlease authority. That is an explicit
specification reopen condition.

### Terminal Command Convergence

Selected: synchronous HTTP terminal commands and asynchronous Redpanda terminal
events both call the same app-level terminal settlement method and store the
same idempotency, child-debit, usage-operation, ledger, and operation-outcome
state. Redpanda is the normal migrated terminal path; synchronous HTTP terminal
commands are allowed for retry/readback repair and future clients, not as a
separate money model.

Rejected: event-only settlement with no operation readback. The proxy and
operators need a deterministic readback path after ambiguous timeouts and
worker lag.

### Proxy Execution Route

Selected: planning must include cross-repo `gonka-proxy` tasks and proof when
cross-repo writes are authorized. If they are not authorized, planning must stop
with a precise proxy implementation handoff and must not claim cutover
completion. Billing-service-only implementation can prove provider readiness,
but it cannot prove the authority cutover.

Rejected: declaring success after billing-service endpoints exist while live
proxy completion or web-search paths still mutate local money state for
migrated cohorts.

## Artifact Index

Core design artifacts:

- `design/overview.md`: review-ready entrypoint and artifact index.
- `design/component-map.md`: review-ready package, adapter, runtime, and
  cross-repo surface map.
- `design/sequence.md`: review-ready account, balance, admission, terminal,
  worker, readback, and fail-closed flows.
- `design/ownership-map.md`: review-ready source-of-truth, dependency,
  generated-code, auth, worker, and non-owner decisions.

Conditional design artifacts:

- `design/data-model.md`: review-ready. Triggered by account import/parity,
  usage operation/hold/readback, microlease child debit, inbox/outbox, and
  reconciliation state.
- `design/dependency-graph.md`: review-ready. Triggered by new app services,
  generated contract flow, Postgres repositories, worker adapters, and
  cross-repo proxy adapter coupling.
- `design/contracts/http-api.md`: review-ready. Triggered by new internal
  account, balance, usage, operation, reconciliation, admin, and auth-scope
  contract surfaces.
- `design/contracts/events.md`: review-ready. Triggered by Redpanda terminal,
  checkpoint, close, and billing-fact event ownership.
- `design/worker-runtime.md`: review-ready. Triggered by the existing no-op
  worker bootstrap and required Redpanda/inbox/outbox/reconciliation runtime
  ownership.
- `design/rollout-validation-inputs.md`: review-ready. Triggered by mixed-mode
  proxy cutover, failback constraints, cross-repo proof, and layered validation.

Later conditional artifacts:

- `test-plan.md`: expected later. Validation is too broad for `tasks.md` alone,
  but this technical-design phase is not allowed to write it. Inputs are in
  `design/rollout-validation-inputs.md`.
- `rollout.md`: expected later. Proxy cutover, mixed-mode gating, failback, and
  operator readbacks require a rollout artifact, but this technical-design
  phase is not allowed to write it. Inputs are in
  `design/rollout-validation-inputs.md`.

Not expected in this phase:

- code, migrations, generated files, tests, `tasks.md`, `test-plan.md`,
  `rollout.md`, technical-design-review output, implementation handoff, or
  validation execution.

## Review Readiness

Technical design review may start from this bundle. No planning-critical
decision is intentionally left for `tasks.md` or implementation. Review should
fail and reopen specification if it finds a need for direct per-request reserve
fallback, proxy-local money writes for migrated cohorts, non-JWT bearer-key
production auth, top-up/payment ownership, organization charging, Redis or
memory spend authority, weaker privacy policy, or a runtime shape that cannot
fail closed.
