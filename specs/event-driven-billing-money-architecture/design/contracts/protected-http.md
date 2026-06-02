# Protected HTTP Contract Design

Status: repaired review-ready technical design for billing-issued spending leases
Authority note: design-only. `api/openapi/service.yaml` is the future runtime
source of truth.
Consumes: `../overview.md`, `../sequence.md`, `../ownership-map.md`

## Contract Boundary

The target protected HTTP surface covers spending-lease commands and readback.
Per-request child debit allocation is proxy-local durable state, not a billing
HTTP call. Terminal finalize/write-off mutation is carried by Redpanda terminal
facts after proxy durable terminal submission.

Current target routes:

| Route | Purpose | Side effect | Required scope |
| --- | --- | --- | --- |
| `POST /internal/billing/v1/spending-leases/issue` | Issue initial lease or replenish capacity for an account/owner. | May reserve USD exposure, create/update lease, ledger effect, stored outcome, and outbox facts. | `billing.lease.issue` |
| `POST /internal/billing/v1/spending-leases/readback` | Read durable lease state after ambiguous timeout, expiry, or settlement lag. | None. | `billing.lease.read` |
| `POST /internal/billing/v1/spending-leases/close` | Close/cancel a lease when proxy or operator supplies durable proof that unused capacity is safe to release. | May release reserved capacity or open reconciliation. | `billing.lease.close` |
| `POST /internal/billing/v1/operations/readback` | Read operation, child debit, terminal, or stored outcome state. | None. | `billing.operation.read` |
| `POST /internal/billing/v1/accounts/balance-readback` | Read primary balance state by account scope for internal callers/support paths. | None. | `billing.account.balance.read` |
| `POST /internal/billing/v1/reconciliation/redrive` | Authorized repair/redrive command referencing existing billing durable state. | May create reconciliation command/outcome, not direct silent ledger edits. | `billing.reconciliation.redrive` |

Identifiers are passed in request bodies, not path variables, because current
HTTP access logging records raw `path`. If a future route places account, lease,
debit, or operation identifiers in the path, implementation must first redact
raw paths or prove route-template-only logging for protected money routes.

## Security

OpenAPI must not leave these routes under global `security: []`.

Required:

- OpenAPI security scheme such as `InternalServiceBearerAuth` or equivalent
  service-principal auth.
- Route-scope checks in HTTP middleware before handler side effects.
- Service principal identity, caller scope key, represented subject evidence,
  deadline, contract version, request ID, and trace context.
- Fail-closed startup/readiness when protected-route auth config is missing.
- 401 for missing/invalid authentication.
- 403 for authenticated caller lacking route scope or account binding.
- No browser CORS exception for these routes.

Accepted caller principals for current scope:

- `svc:gonka-proxy` for lease issue/replenish/readback/close and operation or
  balance readback needed by proxy.
- billing operator/admin principal for close/cancel and reconciliation redrive.
- billing-service worker principal only for internal repair/readback paths
  where explicitly scoped.

## Common Headers

Use existing proxy shared internal-money header evidence where compatible:

- `x-gonkagate-contract-version`
- `x-gonkagate-deadline-at-epoch-ms`
- `x-request-id`
- `authorization`
- trace propagation headers

OpenAPI must also define an idempotency header or body field for
money-affecting POSTs. Lease issue/replenish and close/cancel commands reject
missing idempotency.

## Lease Issue/Replenish Request

Required semantic fields:

- `accountScopeKey`
- `proxyLeaseOwnerId`
- requested `spendingLeaseId` when retrying/readback or `null` for first issue
  when billing generates it;
- requested amount in USD decimal string and/or atoms after canonical parsing;
- requested use class and lease policy version;
- requested expiry and child debit cutoff;
- `idempotencyKey`
- `operationFingerprint`
- pricing snapshot identity/fingerprint, timestamp/expiry, policy versions,
  reserve ceiling, and USD-compatible amount basis;
- represented user context and safe API-key/session attribution evidence;
- authorization/policy evidence, including whether spend-limit check was
  required;
- safe correlation IDs.

Forbidden:

- raw prompt, completion, SSE, provider payload, bearer token, API key, DSN,
  payment secret, raw webhook body, or dynamic untrusted evidence URL content;
- non-USD customer-money amount;
- pricing-service lookup while holding the money transaction.

## Lease Issue/Replenish Response

Success returns the durable stored outcome:

- `spendingLeaseId`
- `accountScopeKey`
- `proxyLeaseOwnerId`
- `spendingLeaseGeneration` / `leaseFence`
- issued or replenished amount;
- total issued amount and billing remaining reserved amount;
- debit cutoff and lease expiry;
- stored outcome identity;
- hold ledger reference or settlement effect reference;
- pricing snapshot identity and policy versions;
- safe balance version/readback reference.

Stored business rejection returns a replayable failure body with a safe reason
code and stored outcome reference when one exists. Lease issue/replenish never
returns `not_ready` as a success that permits new paid admission.

## Lease Readback

`POST /internal/billing/v1/spending-leases/readback` accepts one of:

- `spendingLeaseId` plus account scope;
- stored outcome identity;
- idempotency key plus account/kind when the proxy only has retry basis.

Readback statuses:

- `issued`
- `active`
- `closing`
- `closed`
- `expired`
- `revoked`
- `reconcile_required`
- `manual_review`
- `conflict`
- `not_found`

Readback cannot create a money effect.

## Lease Close/Cancel

Close/cancel request fields:

- `spendingLeaseId`
- `accountScopeKey`
- `proxyLeaseOwnerId`
- `spendingLeaseGeneration` / `leaseFence`
- close/cancel kind;
- monotonic checkpoint sequence or close proof identity;
- allocated child cap sum, terminal submitted sum, and proxy-reported local
  remaining amount;
- close proof fingerprint;
- idempotency key and operation fingerprint;
- safe caller/operator evidence.

Billing may release only capacity it can prove is not allocated to open child
debits and not already settled. Incomplete proof opens or updates
reconciliation and keeps disputed exposure reserved.

## Operation Readback

`POST /internal/billing/v1/operations/readback` accepts one operation identity:

- `usageOperationId`;
- `debitAuthorizationId` plus `spendingLeaseId`;
- `storedOutcomeId`;
- idempotency key plus account/kind when the proxy only has retry basis.

Readback statuses:

- `lease_issued`
- `lease_closed`
- `child_debit_seen`
- `finalized`
- `written_off`
- `reconcile_required`
- `manual_review`
- `conflict`
- `not_ready` only for async terminal or close outcomes durably accepted but
  not yet settled
- `not_found`

Readback cannot create a money effect.

## Balance Readback

`POST /internal/billing/v1/accounts/balance-readback` accepts
`accountScopeKey` plus caller/account binding evidence.

Returns primary Postgres balance:

- settled, reserved, available, and pending USD amounts;
- exact atom values for internal callers;
- active lease exposure summary;
- balance version and last ledger reference;
- account state.

Readback from cache or replica requires a later staleness design. Initial target
uses primary Postgres for correctness.

## Problem And Status Mapping

| Condition | HTTP status | Safe problem code |
| --- | --- | --- |
| malformed request, missing idempotency, invalid deadline, invalid decimal | 400 | `invalid_request` |
| unauthenticated service principal | 401 | `unauthenticated` |
| missing scope or caller/account mismatch | 403 | `forbidden` or `account_scope_mismatch` |
| unsupported route/method | 404 or 405 | existing route policy |
| insufficient funds for requested lease capacity | 402 | `insufficient_funds` |
| stale pricing, unsupported currency, account not spendable, stale policy evidence, unsupported use class | 422 | specific safe business code |
| same idempotency key changed fingerprint | 409 | `idempotency_conflict` |
| lease/debit/readback conflict | 409 | `lease_conflict` or `debit_conflict` |
| account or lease contention timeout, account-scoped admission throttle, or admitted overload | 429 | `account_contention_timeout`, `admission_backpressure`, or `overloaded` |
| dependency/readiness failure, missing or expired admission-control lease, global fail-closed state before command acceptance | 503 | `billing_not_ready` or `admission_closed` |
| internal timeout after possible acceptance | 504 to caller only when no stored outcome can be returned; proxy must retry/readback same identity | `ambiguous_outcome` |

Problem responses include `type`, `title`, `status`, `request_id`, and safe
machine code. They must not include raw request body, event payload, pricing
payload, prompt/completion text, tokens, secrets, or DSNs.

## Compatibility

The current proxy TypeBox internal-money contract is source evidence for names
and field intent. It is not the target authority.

Any rollout bridge must adapt to the target OpenAPI semantics:

- same account scope;
- same lease/debit/usage operation lineage;
- same idempotency key and fingerprint semantics;
- same stored outcome/readback semantics;
- no direct per-request reserve fallback;
- no long-lived proxy-local balance writer for migrated cohorts.
