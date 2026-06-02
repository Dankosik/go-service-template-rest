# HTTP Contract Design

Status: review-ready
Date: 2026-06-02

## Contract Authority

Runtime authority is `api/openapi/service.yaml`. This design file records the
required shape before OpenAPI authoring; it does not replace the OpenAPI file.

All new internal routes:

- live under `/internal/billing/v1`;
- use `ServiceBearerAuth` scoped JWT/JWKS auth;
- require contract version, deadline, trace/correlation ID, caller principal,
  caller scope, represented user/account context where applicable, and safe
  operation metadata where accepted;
- reject non-USD customer-money inputs;
- reject missing idempotency key/fingerprint for money commands;
- return bounded Problem responses and/or bounded success/failure envelopes
  without raw request bodies or secrets.

## Route Set

Account and balance:

- `POST /internal/billing/v1/accounts/resolve`
  - Scope: `billing.accounts.resolve`.
  - Purpose: map represented proxy user to `account_scope_key`, billing account
    ID, account state, import state, migration state, and paid-readiness flags.
  - Side effects: none.
- `GET /internal/billing/v1/accounts/{accountScopeKey}/balance`
  - Scope: `billing.balances.read`; admin variant also accepts
    `billing.admin.read`.
  - Purpose: authoritative balance, active exposure, import/parity,
    stale/ambiguous, manual-review, and worker/admission readback.
  - Side effects: none.

Usage:

- `POST /internal/billing/v1/usage/reservations`
  - Scope: `billing.usage.write`.
  - Purpose: create or read back a usage reservation lineage record.
  - Migrated proxy authority mode: requires microlease ID, child debit identity,
    owner/fence/generation, child cap, request basis fingerprint, pricing
    lineage, and stable idempotency.
  - Rejected for migrated proxy if it lacks child-debit lineage or attempts
    direct account-balance reserve fallback.
- `POST /internal/billing/v1/usage/finalizations`
  - Scope: `billing.usage.write`.
  - Purpose: terminal final charge within child/parent authority cap.
- `POST /internal/billing/v1/usage/write-offs`
  - Scope: `billing.usage.write`.
  - Purpose: explicit write-off or release when terminal evidence is missing,
    unsafe, over cap, or external execution cannot be charged.
- `POST /internal/billing/v1/usage/reversals`
  - Scope: `billing.usage.write`.
  - Purpose: explicit compensation against a prior committed effect.
- `POST /internal/billing/v1/usage/readback`
  - Scope: `billing.usage.read`.
  - Purpose: account-bound usage operation readback by safe usage identity.

Microleases:

- Keep existing `/internal/billing/v1/microleases/issue`,
  `/readback`, and `/close`.
- Extend schemas only as needed to bind account resolve/import state,
  active-exposure readback, or operation readback identity.
- Keep issue/close as `billing.microleases.write`; readback as
  `billing.microleases.read`.

Operations:

- `POST /internal/billing/v1/operations/readback`
  - Scope: `billing.operations.read`.
  - Purpose: stored outcome and ambiguous outcome readback by billing operation
    ID, usage operation ID, microlease ID, child debit ID, terminal outcome ID,
    idempotency key, or reconciliation link according to generated schemas.
  - Design repair: middleware route-scope mapping must align with existing
    OpenAPI `x-route-scopes: [billing.operations.read]`, not
    `billing.microleases.read`.

Reconciliation and admin readbacks:

- `GET /internal/billing/v1/reconciliation/cases`
  - Scope: `billing.reconciliation.read`.
  - Purpose: list support-safe stale/ambiguous/conflict/import cases with
    filters by account, reason, state, and severity.
- `GET /internal/billing/v1/admin/accounts/{accountScopeKey}/ledger`
  - Scope: `billing.admin.read`.
  - Purpose: support-safe ledger and balance-version history.
- `GET /internal/billing/v1/admin/accounts/{accountScopeKey}/exposure`
  - Scope: `billing.admin.read`.
  - Purpose: active holds, microleases, child debits, terminal lag, outbox,
    inbox, and admission-control readback.

Admin mutation routes are not in scope.

## Request Identity And Idempotency

Money-affecting POST requests require:

- `routeContractId`;
- `contractVersion`;
- `idempotencyKey`;
- immutable `requestFingerprint`;
- `accountScopeKey`;
- operation identity such as `usageOperationId`, `microleaseId`,
  `debitAuthorizationId`, or `terminalOutcomeId` as applicable;
- pricing snapshot identity and fingerprint for reserve/finalize paths;
- `callerContext`;
- represented user/account context;
- `deadlineAtEpochMs`;
- `traceRequestId`;
- support-safe metadata only.

Replay behavior:

- same idempotency key plus same fingerprint returns stored outcome;
- same key plus changed fingerprint returns conflict;
- timeout after possible acceptance requires same-identity retry or operation
  readback;
- request ID is never a settlement lookup key.

## Status And Result Mapping

HTTP status classes:

- `200`: accepted/stored success, duplicate replay, stored failure readback, or
  operation readback represented in the response envelope.
- `400`: malformed transport, schema contract mismatch, unsupported enum,
  invalid body/path identity.
- `401`: missing or invalid service JWT.
- `403`: valid service JWT without required scope or account binding mismatch.
- `409`: idempotency/fingerprint conflict.
- `422`: business rejection such as non-USD, stale pricing, missing lineage,
  unsupported account state, over cap, or unsafe terminal evidence.
- `423`: locked/manual-review/reconciliation-required when route contract uses
  lock semantics.
- `429`: throttled admission control.
- `503`: dependency/runtime not ready, worker/admission critical state,
  billing-service disabled, or ambiguous outcome requiring same-identity
  readback.

Proxy maps internal failures to existing OpenAI-compatible errors, but must not
mask billing failure as successful unpaid or locally charged usage.

## Privacy And Metadata

Contracts must reject or omit:

- raw prompts;
- raw completions;
- SSE chunks;
- bearer tokens;
- API keys;
- DSNs;
- payment secrets;
- raw provider payloads;
- raw event payloads;
- dynamic proof URLs;
- sensitive request bodies.

Safe identifiers are allowed only when they are bounded and needed for
correlation, readback, or audit: account scope key, operation IDs, microlease
ID, child debit ID, idempotency key digest or exact key where policy permits,
pricing snapshot IDs, safe execution references, reconciliation case IDs, and
ledger/effect IDs.

## Proxy Compatibility Decision

The existing proxy shared-balance bridge to `/api/v1/usage/*` is not a
provider contract for billing-service. Later proxy work must replace/adapt that
bridge to these `/internal/billing/v1` routes and scoped JWT auth. Billing
should not expose a long-lived compatibility clone of proxy's old paths.
