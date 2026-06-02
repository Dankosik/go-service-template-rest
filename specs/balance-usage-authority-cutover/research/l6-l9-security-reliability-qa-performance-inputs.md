# L6-L9 Security, Reliability, QA, And Performance Inputs

L6 question: What trust-boundary, service-auth, tenant/account attribution,
privacy, and abuse controls affect scope and validation?

L7 question: What timeout, fail-closed, worker lifecycle, readiness, rollback,
and degraded-mode behavior must be specified?

L8 question: What unit, integration, contract, worker, replay, privacy,
performance, and cross-repo proof obligations must be carried into
specification and planning?

L9 question: What hot-path budgets and benchmark proof are required for proxy
paid-usage cutover without unbacked memory or Redis spend authority?

## Security And Privacy Findings

- Billing-service service auth is JWT/JWKS based with issuer, audience, RS256,
  and route scopes (`internal/infra/http/service_auth.go:76-137`).
- Proxy shared-balance bridge sends an `Authorization: Bearer <authKey>` header
  from `BILLING_SERVICE_AUTH_KEY` plus target account headers
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/utils/http/internal-json-service.ts:18-34`,
  `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/shared-balance-live.ts:422-460`).
  Specification must close the auth-model mismatch instead of assuming
  compatibility.
- Billing-service currently recognizes `billing.microleases.read` and
  `billing.microleases.write` constants, while OpenAPI also declares
  `billing.operations.read` for operation readback
  (`internal/infra/http/service_auth.go:25-27`,
  `api/openapi/service.yaml:228-260`).
- Microlease HTTP validation rejects unsafe metadata and requires caller context,
  account scope, pricing identity, idempotency, deadline, and trace identity
  (`internal/infra/http/handlers.go:176-317`).
- Proxy microlease child-debit helper rejects metadata containing prompt,
  completion, SSE, bearer, key, DSN, payment secret, raw provider payload, or
  request body tokens
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/microlease/durable-microlease-allocator.ts:252-278`).
- Existing billing docs require no raw prompts, completions, SSE chunks, API
  keys, DSNs, bearer tokens, payment secrets, or raw provider payloads in billing
  logs or durable records (`docs/PRD.md`, `docs/critical-billing-context.md`).

## Reliability And Rollout Findings

- Billing-service config requires fail-closed admission while microlease runtime
  is disabled and validates config so enabled microlease runtime requires
  Postgres, service auth, Redpanda, and Redis disabled
  (`internal/config/validate.go:143-166`, `internal/config/validate.go:221-239`).
- Defaults keep microlease HTTP and worker disabled and set transaction/time
  budgets for issue, terminal, stale age, reconciliation SLA, and admission
  renewal (`internal/config/defaults.go:71-89`,
  `env/config/default.yaml:57-92`).
- HTTP generated handlers return `503` when the microlease runtime is absent,
  giving a fail-closed behavior but also proving current service bootstrap is
  not ready for runtime calls (`internal/infra/http/handlers.go:98-160`,
  `cmd/service/internal/bootstrap/run.go:151-168`).
- Billing worker command exits if `microlease.worker_enabled` is false and
  currently uses no-op tasks if enabled
  (`cmd/billing-worker/internal/bootstrap/run.go:31-96`).
- Proxy web-search local billing guard throws service unavailable when
  shared-balance cutover is enabled, so specification must define replacement
  billing authority before enabling that flag for web-search cohorts
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/completions/web-search/billing-guards.ts:41-60`).
- Proxy migrated-cohort policy explicitly disallows direct reserve fallback for
  `microlease_migrated` and fails closed when capacity is missing, cutoff is
  reached, or local backlog is critical
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/services/billing/microlease/migrated-cohort-policy.ts:24-72`).

## QA And Proof Findings

- `docs/build-test-and-development-commands.md` identifies repository proof
  surfaces for later phases: `make check`, OpenAPI check, SQLC check,
  migration validation, integration tests, guardrail checks, and generated drift
  checks. None were run in this research phase.
- Prior completed microlease task evidence says the previous packet validated
  billing microlease integration/performance and proxy allocator performance,
  but this broader cutover still needs fresh proof against new surfaces.
- Required future proof must cover both repos if proxy edits are approved:
  billing-service OpenAPI generation, SQLC generation, migrations, repository
  transactions, HTTP handler status mapping, service auth, worker replay,
  outbox/inbox, privacy-safe logs, and proxy completion/web-search behavior.
- Negative-path proof is approval-critical: stale/ambiguous operations,
  idempotency replay, payload conflicts, missing auth scopes, missing billing
  runtime, unavailable pricing, missing account, suspended account, no
  microlease capacity, cutoff reached, and worker outage.

## Performance Findings

- Existing billing config budgets include max issue transaction duration of
  100ms and max terminal transaction duration of 250ms by default
  (`internal/config/defaults.go:86-87`).
- `docs/microlease-performance-proof.md` records prior proof commands and
  budgets: billing issue/replenish p95 under 100ms/p99 under 250ms; proxy durable
  child allocation p95 under 10ms/p99 under 25ms; cold replenishment p95 under
  250ms/p99 under 500ms; first-token added latency p95 under 25ms.
- The same document states memory is a proxy cache/precheck only and every
  successful allocation must still commit through the durable repository and
  return a terminal obligation before execution can proceed
  (`docs/microlease-performance-proof.md:17-23`).
- Proxy's current hot path still uses local in-process reservation before
  external execution and local DB deduction after execution. The cutover spec
  must define the replacement budget and prove no memory or Redis path can mint
  spend authority.

## Must-Decide-Now Inputs For Specification

- Auth: scoped JWT service auth, proxy token minting, route scopes, account
  binding, and whether legacy `BILLING_SERVICE_AUTH_KEY` is removed or replaced.
- Fail-closed behavior: exact proxy response class and retryability for
  billing-service unavailable, no account, suspended/manual-review account,
  missing capacity, stale pricing, stale microlease, and worker backlog.
- Readiness: what service and worker readiness checks must pass before cohorts
  move to billing authority.
- Rollback: whether migrated cohorts can roll back to local proxy money writes.
  Current constraints say direct reserve fallback is out of scope for migrated
  paid cohorts, so any rollback must avoid dual-writing money authority.
- Validation: future plans must include fresh local command evidence and, if
  live secrets are unavailable, clearly separate config/negative-path proof from
  happy-path live proof.
