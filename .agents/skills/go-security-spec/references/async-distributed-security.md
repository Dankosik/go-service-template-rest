# Async And Distributed Security

## Behavior Change Thesis
When loaded for queues, workers, callbacks, cross-service calls, or third-party APIs, this file makes the model choose authenticity, replay, scoped credential, and step-authorization requirements instead of likely mistake: trusting internal queues, propagating raw bearer tokens, or relying on eventual consistency after unsafe side effects.

## When To Load
Load this when requirements touch queues, workers, outbox/inbox, webhooks, callbacks, background retries, cross-service calls, third-party APIs, token propagation, message replay, compensations, or distributed workflow security.

## Decision Rubric
- Define message authenticity: producer identity, signature or MAC choice, signing key owner, rotation behavior, canonical payload, and verification point.
- Define replay controls: event ID, nonce or sequence, timestamp tolerance, dedup store, TTL, poison-message handling, and consumer-side idempotency.
- Prohibit raw end-user bearer-token propagation through async payloads unless a short-lived, scoped, encrypted, audited exception is explicitly approved.
- Prefer token exchange or internal service credentials for worker-to-service calls, scoped to target audience and action.
- Require step-level authorization for who may trigger, retry, compensate, cancel, or resume each workflow step.
- Treat third-party API responses as untrusted input: TLS, redirect allowlist, timeout, response-size limit, media-type check, schema validation, and no raw provider response relay to untrusted clients.

## Imitate
- "Worker verifies producer signature, timestamp skew, event ID, tenant binding, and dedup state before side effects; unverifiable messages are quarantined or acknowledged according to poison-message policy without mutation." Copy the verify-before-side-effect order.
- "Retry workers use a target-service credential with audience and scope limited to the action; downstream auth failure does not fall back to a broader credential or caller token." Copy credential purpose and fail-closed retry behavior.
- "Compensation requires permission for the compensation action, not just permission for the original action." Copy step-level authorization.

## Reject
- "The queue is internal, so messages are trusted." Queue access, producer identity, payload integrity, and tenant binding still matter.
- "Eventually consistent auth will fix it later." Access control after an irreversible side effect is not a security control.
- "Put the user's bearer token in the job payload." This widens token lifetime, audience, replay, and disclosure risk.
- "Retry until it works." Infinite retries can bypass policy, amplify abuse, or grow dead letters without control.

## Agent Traps
- Do not let distributed-design terms such as outbox, saga, or choreography imply security. They describe delivery shape, not authenticity or authorization.
- Do not forget webhook registration security: endpoint ownership, callback signature support, SSRF target policy, and retry bounds.
- Do not persist full third-party responses unless data classification, minimization, and sanitization are decided.

## Validation Shape
- Duplicate-message tests prove one committed side effect per event ID.
- Mutated payload, invalid signature, stale timestamp, wrong audience, wrong tenant, unknown producer, and missing dedup state fail before side effects.
- Retry tests prove downstream auth failures do not bypass policy or swap to broader credentials.
- Third-party and webhook tests cover unexpected redirect, schema, media type, response size, SSRF target, and provider-error sanitization.

## Repo-Local Anchors
- This template currently has HTTP, config, PostgreSQL, telemetry, and health/ping surfaces, but no committed queue or worker runtime. Async security requirements should mark queue/worker assumptions explicitly until such infrastructure exists.
- `cmd/service/internal/bootstrap` and `internal/infra/http` own startup/shutdown and HTTP boundaries; new workers should define equivalent lifecycle and security boundary ownership.
