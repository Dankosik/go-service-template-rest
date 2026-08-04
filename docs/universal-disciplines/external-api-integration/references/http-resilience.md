# HTTP resilience and resumable synchronization

Use this reference only for an outbound HTTP, transport-failure/rate-limit, or paginated synchronization branch. Start with the provider contract; HTTP semantics constrain a policy but do not replace it.

## Attempt envelope

Represent the operation with an absolute monotonic deadline and compute the remaining budget immediately before each blocking action. Configure the transport so DNS/connect/TLS, request write, response headers, and response-body reads cannot exceed it. Also bound response bytes, decompression expansion, redirect count, and idle connection behavior.

Propagate cancellation to the transport and close or drain response bodies according to the client library's connection-reuse contract. Do not create a fresh full timeout for each retry; that turns a bounded request into an unbounded sequence.

Record attempt number, elapsed and remaining budget, endpoint class, provider request ID, outcome class, retry reason, planned delay, and response size without recording credentials or sensitive bodies.

## Decide retry eligibility

Evaluate these questions together:

1. Did the provider contract define this operation as idempotent, or did a documented idempotency key make this exact intent safe to repeat?
2. Can the client prove the request was not sent, or must it treat the outcome as possibly applied?
3. Does the provider error code or header explicitly make the same operation retryable?
4. Is the same operation identity and immutable payload available for the next attempt?
5. Is enough deadline and retry budget left to make another attempt useful?

RFC 9110 makes safe methods plus `PUT` and `DELETE` idempotent by method semantics, but a provider's resource semantics and preconditions still matter. It advises against automatically retrying a non-idempotent method unless the client knows the request semantics are idempotent or knows the original was not applied. A `POST` with no documented protection therefore starts with zero automatic after-send retries.

Treat status classes as evidence, not a retry table:

- Authentication failures enter the authentication branch; repeating the same rejected credential is not recovery.
- Validation, permission, missing-resource, conflict, and business errors require provider-code interpretation. Some providers deliberately use `404` for unauthorized resources or `409`/`422` for in-progress idempotent work.
- `429` means the caller is rate limited, not that this particular side effect is safe or still useful to retry.
- A side-effecting `5xx`, gateway timeout, connection loss, or truncated success body can be ambiguous if the provider may have committed first.
- A malformed or semantically unknown success response is `incompatible` or `ambiguous`, not automatically success.

## Bound retries and pressure

Set both a maximum attempt count and maximum elapsed retry budget. Exponential backoff with full jitter can draw a delay from `0..min(cap, base * 2^retry_index)`; choose `base`, `cap`, and count from the operation's latency and provider constraints, then test them. Preserve the same idempotency key and intent across attempts.

Parse `Retry-After` as either delay-seconds or HTTP-date. Treat it as a no-earlier-than signal where the provider applies it. If that delay does not fit the operation deadline, stop the synchronous attempt and schedule owned recovery rather than sleeping past cancellation. Provider-specific reset and quota headers can refine this rule only when the pinned contract defines their scope and clock.

Limit concurrency at the provider's documented quota identity: account, credential, tenant, endpoint, region, or another bucket. Share a retry budget so retries cannot consume all capacity needed by first attempts. Queue entries need expiry/deadline and fairness; open-loop sleeping in every caller can preserve a thundering herd.

Measure request rate, in-flight work, queue age, attempt distribution, throttle responses, provider-directed wait, local limiter wait, remaining/reset signals where reliable, retry-budget exhaustion, and quota headroom. Keep provider/account IDs out of metric labels when cardinality or privacy would be unsafe.

## Polling

Poll a documented status/read endpoint, not the original write. Stop on provider-defined terminal states, local deadline/age limits, revocation, or an operator-owned unresolved state. Apply conditional requests, provider poll intervals, jitter, and quota sharing when the contract supports them.

Keep a critical reconciliation poll independent of a short-lived user request. If webhooks normally complete work, polling should focus on overdue or uncertain operations instead of duplicating every callback at high frequency.

## Pagination and restart

Treat cursors and continuation URLs as opaque provider state. Retain the request version, filters, sort, page size, account/environment, cursor, and any snapshot or high-watermark token needed to resume the same traversal.

For each page:

1. Fetch within the synchronization deadline and quota budget.
2. Apply each item idempotently under its provider identity/version.
3. Retain permanent item failures with a reason and explicit retry-after-change or quarantine state.
4. Advance the durable checkpoint only after all accepted page effects and failure records are durable.

A crash before checkpoint advancement replays the page safely. A crash after advancement cannot skip its effects. If one invalid item must not block unrelated items, store its permanent failure and continue only when the business handoff permits partial progress.

When the owning contract is page-atomic, item-level durability is not publication: keep every item effect and the next checkpoint invisible until the whole page can publish atomically. A permanent item failure leaves that page unresolved unless the owner explicitly changes the partial-progress contract; idempotent replay alone does not make already-visible partial effects acceptable.

Offset/page-number pagination over a mutable collection can skip or duplicate records. Prefer a documented stable sort plus cursor/snapshot. If a cursor expires, restart from a safe provider-supported watermark or from an overlapping lookback with deduplication; do not invent cursor structure. Define deletion/tombstone handling and a periodic full or bounded reconciliation when incremental feeds cannot prove removals.

## Failure checks

At minimum, make deterministic tests cover:

- deadline expiry in DNS/connect, request write, headers, and body read;
- cancellation during backoff and I/O;
- connection loss before send and after possible send;
- `429` with valid, absent, malformed, and beyond-deadline delay signals;
- retryable provider error followed by success, and permanent error with zero retry;
- concurrent calls sharing limiter and retry budget;
- crash before and after page checkpoint advancement;
- expired cursor, overlapping restart, duplicate item, deletion, and permanent invalid item.

Use a fake clock and scripted transport when available so timing assertions are executable and fast. Provider sandbox checks supplement these tests; they do not replace the ambiguous network phases a sandbox cannot reproduce reliably.

## Primary sources

- [RFC 9110: HTTP Semantics, method idempotency and Retry-After](https://www.rfc-editor.org/rfc/rfc9110.html)
- [RFC 6585: 429 Too Many Requests](https://www.rfc-editor.org/rfc/rfc6585.html)
- [GitHub REST API best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api)
- [Stripe idempotent requests](https://docs.stripe.com/api/idempotent_requests)
- [Adyen API idempotency](https://docs.adyen.com/development-resources/api-idempotency/)

The provider examples demonstrate why scope, retention, cached-result, regional, and error behavior must be pinned per provider rather than generalized from the presence of an idempotency header.
