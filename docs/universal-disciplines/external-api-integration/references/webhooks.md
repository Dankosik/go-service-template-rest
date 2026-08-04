# Secure and recoverable webhooks

Use this reference only when the provider pushes callbacks or webhooks. A webhook is an untrusted, retried notification channel; durable provider state or a documented event log remains the recovery source.

## Authenticate before effect

Capture the exact raw request bytes before JSON/form parsing or middleware mutation. Enforce HTTPS, method/content-type/body-size limits, and the provider's endpoint/environment binding. Verify with the official library when available; otherwise implement the exact documented canonicalization, algorithm, key encoding, and constant-time comparison.

Verify the signature or documented authentication before durable acceptance or any business effect. Bind verification to the correct endpoint secret/key generation and environment. Network allowlists or Basic Authentication can add a layer but do not substitute for message-integrity verification when the provider signs events.

Apply the provider's signed timestamp and tolerance contract when it has one, with a synchronized clock. If the signature scheme has no signed timestamp, do not invent timestamp protection: use the provider delivery/event identity for deduplication, retain it for at least the provider's replay/redelivery window, and retrieve provider state when freshness matters.

During signing-secret rotation, accept exactly the provider-documented old/new overlap and record which generation verified the event. Remove the old verifier only after the overlap and delivery backlog are clear.

## Accept durably, then process

Keep the synchronous handler small:

1. Bound and capture the raw envelope.
2. Authenticate it and apply replay-window checks.
3. Atomically retain the raw-body hash, provider event/delivery identity, headers needed for audit, received time, version/type, and processing state.
4. Return the provider-defined success response.

If durable retention fails, return the provider-defined failure response so its delivery policy can recover. After an authenticated duplicate is already durable, return the same success class without repeating the business effect. Semantic validation and business processing happen asynchronously; they cannot extend provider acknowledgement latency.

Hand message transport and consumer mechanics to `reliable-messaging` with the retained event identity, ordering key if documented, durable-acceptance point, acknowledgement rule, and poison-event policy.

## Deduplicate and order by contract

Prefer a provider delivery/event ID as the receipt identity. If the provider documents that distinct events can express the same semantic change, add the documented semantic key, such as object identity plus event type/version. Enforce uniqueness atomically in the storage design handed to `postgres-schema-design`.

Assume duplicate and out-of-order delivery unless the pinned contract guarantees otherwise. A receipt timestamp is not provider order. Use a provider sequence, object version, or event creation position only with documented comparison semantics. Make stale events a no-op, and resolve gaps or conflicts by fetching current provider state or running reconciliation rather than guessing intermediate transitions.

Do not let one unrecognized event create a retry storm. Durably quarantine unknown type/version/schema evidence, emit an incompatibility signal, and choose the acknowledgement behavior from the provider's retry and retention contract. Reconciliation must still be able to recover the affected resource.

## Recover missed delivery

Track accepted, verified, rejected, duplicate, queued, processed, failed, and quarantined states. Record acknowledgement latency, delivery-to-receipt lag, processing lag, signature/timestamp failures, duplicates, stale/out-of-order events, unknown schemas, retry count where exposed, and oldest unprocessed event.

Run reconciliation over overdue local operations and provider resources/events so a lost webhook, expired provider retry window, disabled endpoint, or local queue failure cannot permanently lose state. A production replay is an external side effect requiring separate authorization, a bounded event set, deduplication proof, and post-replay reconciliation.

## Failure checks

Use provider fixtures or generated test signatures to prove:

- valid raw bytes pass while a one-byte mutation, wrong secret, wrong environment, or wrong algorithm fails;
- valid old and future timestamps follow the provider tolerance, including clock-skew handling;
- a captured signed delivery cannot repeat its business effect;
- the same event delivered concurrently is retained once and acknowledged safely;
- out-of-order and stale events cannot regress state, and a sequence gap triggers lookup/reconciliation;
- durable-store failure returns the provider-defined retry response;
- slow or failed business processing does not extend synchronous acknowledgement;
- unknown event type/version is retained, alerted, and recoverable without fabricated interpretation;
- dropped callbacks converge through polling or reconciliation.

## Primary sources

- [Stripe webhook delivery, ordering, deduplication, signatures, and replay protection](https://docs.stripe.com/webhooks)
- [Adyen webhook handling and durable acknowledgement](https://docs.adyen.com/development-resources/webhooks/handle-webhook-events)
- [Adyen webhook security and HMAC verification](https://docs.adyen.com/development-resources/webhooks/secure-webhooks)
- [Slack request signing and timestamp replay checks](https://docs.slack.dev/authentication/verifying-requests-from-slack/)

These providers differ in identity fields, signing envelopes, timestamp behavior, ordering data, retry windows, and acknowledgement contracts. Preserve those differences instead of importing one provider's guarantees into another integration.
