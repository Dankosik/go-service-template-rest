# Protected HTTP Contract Design

Status: review-ready design context
Trigger: new protected microlease issue/replenish/readback/close commands.

Runtime authority remains `api/openapi/service.yaml` after implementation
planning. This file defines design constraints only.

## Contract Principles

- Protected routes require real service authentication and route scopes.
- All money-affecting commands require idempotency key, request fingerprint,
  caller context, deadline, account scope, and trace correlation.
- Identifiers for account, microlease, child, operation, and checkpoint stay in
  request bodies, not path variables, until route-path redaction is explicitly
  implemented and reviewed.
- A timeout after possible acceptance is ambiguous. Callers retry/read back with
  the same operation identity.
- Same idempotency key plus same fingerprint returns the stored outcome.
- Same idempotency key plus changed fingerprint returns conflict.
- No route accepts raw prompts, completions, SSE chunks, bearer tokens, API
  keys, DSNs, payment secrets, raw event payloads, dynamic proof URLs, or
  sensitive request bodies.

## Routes To Design In OpenAPI

Names are provisional and must be normalized in `api/openapi/service.yaml`.

| Capability | Provisional route | Money effect |
| --- | --- | --- |
| Issue or replenish microlease | `POST /internal/billing/v1/microleases/issue` | May reserve balance and mint capacity. |
| Read microlease or operation state | `POST /internal/billing/v1/microleases/readback` | No money mutation. |
| Close or cancel microlease with proof | `POST /internal/billing/v1/microleases/close` | May release proven unallocated capacity or open reconciliation. |
| Read reconciliation/operation state | `POST /internal/billing/v1/operations/readback` | No money mutation. |

The design does not add direct terminal finalize/write-off protected HTTP for
the target microlease path. Terminal facts arrive through event inbox. If
planning needs a synchronous terminal mutation path, reopen specification.

## Common Request Fields

Every command body carries:

- `routeContractId`;
- `contractVersion`;
- `idempotencyKey`;
- `requestFingerprint`;
- `callerContext`;
- `accountScopeKey`;
- `representedUserContext` or service attribution context when applicable;
- `deadlineAtEpochMs`;
- `traceRequestId`;
- safe operation metadata.

Issue/replenish additionally carries:

- requested cap inputs and max requested amount in USD atoms or decimal string;
- `maxChildCapUsdAtoms`;
- use class and risk class;
- pricing snapshot identity, fingerprint, policy version, decision time,
  selector/use-class context, and contract metadata;
- proxy allocator owner ID;
- current owner/fence when replenishing;
- requested TTL and cutoff class;
- local backlog summary, terminal lag class, and strict-mode reason if any.

Close additionally carries:

- microlease ID;
- owner and generation/fence;
- close/checkpoint sequence;
- high-water mark;
- allocated child count and cap sum;
- terminal submitted/published/accepted counts;
- unresolved child count and cap sum;
- local remaining capacity;
- close fingerprint.

## Success Result Codes

Issue/replenish:

- `issued`;
- `replenished`;
- `reduced_issued` when billing lowers cap to preserve active exposure gates;
- `readback`;
- `strict_required`;
- `rejected`;
- `payload_conflict`;
- `manual_review`;
- `reconcile_required`;
- `not_ready`.

Readback:

- `found`;
- `not_found`;
- `reconcile_required`;
- `not_ready`.

Close:

- `closed_released`;
- `closed_unresolved_reserved`;
- `duplicate_stored_outcome`;
- `checkpoint_recorded`;
- `rejected`;
- `payload_conflict`;
- `state_conflict`;
- `manual_review`;
- `reconcile_required`;
- `not_ready`.

## Problem And HTTP Status Mapping

| Condition | HTTP | Result / Problem class |
| --- | --- | --- |
| Missing/invalid service auth | 401 | authenticated caller missing. |
| Authenticated but wrong route scope/account binding | 403 | `caller_scope_mismatch` or `account_scope_mismatch`. |
| Schema/framing/unsupported enum | 400 | `schema_contract_mismatch` or `unsupported_request_enum`. |
| Same idempotency key changed fingerprint | 409 | `payload_conflict`. |
| Stale owner/fence, state conflict, duplicate child proof conflict | 409 | `state_conflict`. |
| Insufficient available balance or safety floor | 422 | `insufficient_funds` / safe rejection code. |
| Stale/missing pricing or unsupported use class | 422 | `stale_pricing_snapshot`, `pricing_unavailable`, `unsupported_use_class`. |
| Manual review or suspended account | 423 or 409 | `manual_review` / `rejected` per OpenAPI convention. |
| Admission throttle | 429 | `throttle`. |
| Dependency or admission control not ready | 503 | `not_ready`, fail closed. |

Problem details must be bounded and support-safe.

## Stored Outcome Shape

Issue success stores and returns:

- `microleaseId`;
- account scope;
- proxy owner;
- generation/fence;
- issued cap;
- remaining cap at issuance;
- debit cutoff and expiry;
- pricing snapshot identity/fingerprint;
- policy versions;
- strict/fail-closed reason when reduced or denied;
- replay identity.

Close success stores and returns:

- microlease ID;
- close state;
- released amount;
- unresolved reserved amount;
- terminal/child summary;
- reconciliation case ID when opened;
- replay identity.

## Security And Privacy

- No bearer tokens, API keys, plaintext secrets, raw prompts, raw completions,
  SSE chunks, DSNs, raw events, raw provider payloads, or dynamic proof URLs in
  request/response bodies, Problems, logs, audit, inbox/outbox, or traces.
- Metrics labels may include route, result class, failure class, strict reason,
  and lag bucket only.
- Account scope and operation IDs may appear in structured logs only where the
  repository privacy policy allows support-safe identifiers; never in metrics
  labels.

## Compatibility Notes

The current `gonka-proxy` TypeBox internal-money billing contract has reserve,
finalize, and write-off route IDs. Those are compatibility inputs, not the
target authority for new billing-service runtime routes. Planning must treat
target microlease HTTP as billing-service OpenAPI work and separately task proxy
bridge/disablement obligations.
