# Receive Standard Webhooks callbacks durably

status: ready

Problem: A service generated from the template can deliver Standard Webhooks,
but it cannot safely acknowledge provider callbacks before their verified
payload is durably available for asynchronous business processing. Adopters
would otherwise rebuild body bounds, signature verification, receipt identity,
duplicate handling, transactionality, and failure responses at each endpoint.

## Scope and accepted boundary

This specification adds one removable ingress capability:

```text
INBOUND_WEBHOOKS=none|standard-webhooks
```

`standard-webhooks` means the Standard Webhooks v1 signing protocol and a
durable PostgreSQL receipt followed by River processing. It requires
`DATABASE=postgres JOBS=postgres`. `none` is the default and leaves no inbound
webhook route, configuration, source, migration, worker registration,
documentation, generated surface, or inbound-only dependency in an initialized
service. The selected value is recorded in `template.lock`.

The existing `WEBHOOKS=none|durable` selector continues to mean outbound
delivery. It is neither renamed nor coupled to inbound receipt. The shared
Standard Webhooks dependency may remain when either direction needs it and is
absent when no selected capability needs it.

V1 supports no Stripe-, GitHub-, or other provider-specific authentication
protocol. If the first adopter cannot send Standard Webhooks v1, Specification
reopens for that exact provider contract instead of adding a protocol registry.

## Wire contract

The selected capability adds `POST /webhooks/{endpoint_id}` to the service
OpenAPI contract. It is intentionally reachable without bearer authentication:
the sender authenticates the exact raw body through Standard Webhooks headers.
It is not a browser API and adds no CORS allowance.

`endpoint_id` identifies one configured trust boundary. It is a non-secret,
case-sensitive ASCII token matching `[A-Za-z0-9_-]{1,64}`. An unknown endpoint
returns `404` before any durable write. Each endpoint has exactly one active
secret and may have one predecessor secret during rotation; secret material is
environment-only, is not reused between endpoints, and never enters generated
configuration, files, responses, logs, traces, or metrics.

The request carries one value for each required header:

- `Webhook-Id`: the sender's stable delivery identity, 1..256 valid UTF-8 bytes
  with no whitespace or control character;
- `Webhook-Timestamp`: signed Unix seconds;
- `Webhook-Signature`: one Standard Webhooks v1 signature list; and
- `Content-Type: application/json`.

The current service-wide `http.max_body_bytes` limit bounds the raw body before
signature verification or JSON decoding; its existing default is 1 MiB. V1
adds no second body-size setting. Verification uses the exact received bytes,
the Standard Webhooks v1 canonical message, constant-time signature comparison,
and the protocol's five-minute past/future timestamp tolerance. Either the
active or predecessor endpoint secret may verify during rotation. There is no
invented environment header: environment binding comes from deploying distinct
endpoint configuration and non-reused secret material per environment.

The receiver verifies before interpreting JSON. A verified body is durably
accepted as opaque JSON bytes; provider-specific parsing and schema validation
run asynchronously. This preserves the signed bytes and prevents a generic
capability from inventing an event envelope that Standard Webhooks does not
define.

## Acceptance and response semantics

One PostgreSQL commit atomically creates the receipt and its River job. No
success response is written before that commit. A `204 No Content` means only
that the exact signed delivery is durably owned for asynchronous processing; it
does not mean that a business effect has completed.

The durable receipt identity is `(endpoint_id, Webhook-Id)`. The receipt is the
authority for ingress acceptance and records a SHA-256 hash of the exact raw
body, signed timestamp, receipt time, and processing outcome. Concurrent or
later reuse has one closed result:

| Existing receipt | Incoming delivery | Result |
| --- | --- | --- |
| absent | valid signature and timestamp | atomically create one receipt and one job; `204` |
| same identity and body hash | valid duplicate | create nothing; `204` |
| same identity, different body hash | conflicting reuse | preserve the first receipt, create nothing; `409` |

The same `Webhook-Id` under another configured endpoint is a distinct identity.
An unknown commit outcome never becomes an assumed success: the sender receives
a retryable failure or loses the response, retries the same identity, and then
observes the duplicate rule above.

Failures use the repository's existing RFC 9457 `Problem` contract and never
echo a submitted header, body, signature, secret, parser error, SQL error, or
provider text.

| Condition | Response and effect |
| --- | --- |
| malformed path, duplicate/invalid non-authentication header, or wrong content type | `400`; no write |
| missing, malformed, stale, future, or non-matching authentication material | sanitized `400`; no write |
| unknown endpoint | `404`; no write |
| body exceeds the configured limit | `413`; verification, parsing, and writes do not run |
| accepted identity conflicts with another body | `409`; original receipt remains authoritative |
| configured rate limit rejects the request | `429` with required `Retry-After`; no write |
| PostgreSQL is unavailable, the transaction fails, or service capacity sheds the request | `503` with required `Retry-After`; no success is implied |
| request budget expires before a response can be committed | `504`; sender retries the same identity |
| unexpected internal failure | sanitized `500`; no success is implied |

`431` remains the existing listener/middleware answer for excessive headers.
The fixed success body is empty because Standard Webhooks defines signing, not
a provider-specific response document.

## Asynchronous processing and truth

Business code receives only a verified durable delivery equivalent to:

```go
type VerifiedDelivery struct {
	EndpointID string
	DeliveryID string
	SignedAt   time.Time
	Body       json.RawMessage
	ReceivedAt time.Time
}
```

Each configured endpoint must have exactly one feature-owned decoder and
handler at startup. The capability owns cryptographic verification, receipt
deduplication, durable dispatch, and transport outcomes. The feature adapter
owns the provider JSON schema, event type/version, semantic identity, domain
validation, state transitions, idempotent effects, and any provider lookup or
reconciliation.

River may invoke the same receipt job more than once. A successful handler
marks the receipt handled; a retryable handler failure remains retryable under
the existing jobs policy. Malformed JSON, an unknown event type or version, or
a provider schema the endpoint decoder rejects moves the receipt to quarantine
without invoking a business effect. Exhausted processing remains a durable
failed outcome. Neither quarantine nor terminal failure turns the earlier
`204` into business success, and neither causes an automatic outbound call.

Receipt time does not order provider events. The capability promises no
cross-delivery ordering, exactly-once business effect, or gap detection.
Provider sequence/version rules and the authoritative reconciliation read, when
one exists, belong to the feature adapter. An accepted receipt is not a second
writer for business state.

The raw body is retained while processing can retry and while a receipt is
quarantined or failed. After successful handling, payload bytes are erased while
the minimal identity, body hash, timestamps, and terminal outcome remain so an
old duplicate cannot repeat the effect. V1 has no automatic deletion of that
deduplication evidence; a legal erasure or finite retention requirement reopens
Specification before release for that adopter.

## Runtime and operator behavior

- Selecting the capability with missing PostgreSQL, jobs, endpoint, secret, or
  decoder/handler ownership fails initialization or startup; partial enablement
  is unsupported.
- HTTP readiness requires the durable PostgreSQL acceptance path. It does not
  require a worker to be polling at that instant: a committed job may wait in
  the durable backlog.
- The shared HTTP server owns admission, timeout, drain, and shutdown of an
  in-flight receipt. The existing jobs worker owns claims, retries, recovery,
  telemetry, drain, and shutdown after acceptance. No webhook-specific process
  or retry engine is added.
- Metrics use bounded outcomes such as `accepted`, `duplicate`, `rejected`,
  `quarantined`, `retrying`, `handled`, and `failed`; endpoint IDs, delivery IDs,
  event types, headers, and body fields are not metric labels. Operator logs use
  request and internal receipt correlation only and contain no raw body or
  authentication material.
- Existing global body, concurrency, timeout, and optional rate-limit bounds
  apply before durable work. V1 adds no per-endpoint quota; reopen when measured
  sender traffic or a contractual fairness limit shows the shared bounds are
  insufficient.

## Deliberately unchanged and non-goals

- Outbound webhook delivery, its wire/retry contract, and `WEBHOOKS` profile are
  unchanged.
- The capability is not a generic inbox, broker consumer, provider registry,
  universal event schema, synchronous business handler, or exactly-once effect
  framework.
- It adds no Stripe/GitHub adapters, polling fallback, automatic reconciliation,
  ordering engine, operator HTTP API, dashboard, replay UI, legal hold, or
  provider-specific response body.
- It does not introduce a capability-manifest generator. The existing
  initializer/profile ownership remains authoritative for this task.
- Deployment still owns public ingress routing and TLS termination, secret
  delivery and rotation timing, database capacity, worker capacity, and
  provider endpoint registration. Local proof cannot establish those facts.

## Success criteria and proof expectations

1. A selected profile exposes the documented operation and fails closed when a
   required dependency or endpoint binding is absent; an unselected profile has
   no inbound-webhook residue and leaves current behavior unchanged.
2. Every `204` is preceded by one durable PostgreSQL commit containing the
   receipt and job, including across process loss and ambiguous responses.
3. One-byte body/signature/endpoint changes, timestamp boundary violations,
   oversized bodies, and unavailable storage are rejected before a write.
4. Concurrent identical deliveries create one receipt/job and both succeed;
   conflicting identity reuse preserves the first body hash and returns `409`.
5. Process loss after durable acceptance but before handling does not lose the
   job; retries reuse one receipt identity and never invoke an unverified body.
6. Unknown schemas quarantine the retained evidence, while a supported handler
   can reach handled without leaking provider-controlled data through errors or
   telemetry.
7. OpenAPI/runtime contract checks cover every reachable status, public
   signature-auth rationale, required `Retry-After`, and the empty `204` body.

Transactionality and concurrent uniqueness require real PostgreSQL proof.
Focused negative tests must also show absence of receipt/job/business effects,
not merely the returned status. Template initialization must prove both the
selected and physically removed trees. Test Design owns the final scenario
matrix and proving levels.

## Risks, assumptions, and reopen conditions

- The bounded assumption is that the first adopter can send Standard Webhooks
  v1 with JSON payloads. A named provider requiring another signature,
  timestamp, response, replay, or payload contract reopens Specification for
  that provider rather than widening this generic profile.
- Standard Webhooks v1 and the current repository dependency define the
  canonicalization and five-minute tolerance. Refresh their compatibility in
  Technical Design if the pinned dependency or protocol changes.
- The template has no accepted legal retention/erasure window, provider
  reconciliation API, per-sender quota, or event-ordering promise. Any such
  adopter requirement reopens its named behavioral rule before implementation
  or rollout.
- If raw payload classification forbids retention until terminal processing,
  durable asynchronous receipt is blocked for that provider and Specification
  must choose a permitted encrypted/filtered representation or reject the
  capability for that adopter.

Ready movement is to System / Integration Design, then Go Code / Ownership
Design. Technical Design must select the schema, transaction seam, OpenAPI raw
body path, worker registration, configuration ownership, payload erasure path,
and exact initializer removal set without changing the observable rules above.
