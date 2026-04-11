# Async And Distributed Security Examples

## When To Load
Load this when requirements touch queues, workers, outbox/inbox, webhooks, callbacks, background retries, cross-service calls, third-party APIs, token propagation, message replay, compensations, or distributed workflow security.

## Selected Controls
- Define message authenticity: producer identity, signature or MAC choice, signing key ownership, key rotation, canonical payload, and verification point.
- Define replay controls: event ID, nonce or sequence, timestamp tolerance, dedup store, TTL, and consumer-side idempotency.
- Prohibit raw end-user bearer token propagation through async payloads unless a specific short-lived, scoped, encrypted, and auditable exception is approved.
- Use token exchange or internal service credentials for worker-to-service calls, with audience and scope limited to the target service/action.
- Require step-level authorization for who may trigger, retry, compensate, cancel, or resume each workflow step.
- For third-party API consumption, require TLS, response validation, redirect allowlist, timeout, resource limits, and lower-trust treatment of returned data.
- For webhooks, require endpoint allowlists or ownership verification, callback signatures where possible, bounded retries, and no raw provider response relay to untrusted clients.

## Rejected Controls
- Reject relying on eventual consistency to fix access control after a side effect already ran.
- Reject trusting a queue solely because it is internal. Queue access, producer identity, and message integrity still matter.
- Reject infinite retries, unbounded dead-letter growth, or retry loops that bypass authorization after the first attempt.
- Reject async payloads that carry secrets, full bearer tokens, or unnecessary sensitive data.
- Reject blind redirect following or unlimited response processing from third-party APIs.

## Fail-Closed Examples
- Worker cannot verify message signature, timestamp, or dedup state: acknowledge according to poison-message policy or quarantine, but do not run the side effect.
- Token exchange fails for a downstream call: stop the step and mark retryable/non-retryable according to policy; do not reuse the caller's raw token.
- Compensation actor lacks permission: deny compensation and emit a security event rather than applying a reversal.
- Third-party API returns an unexpected redirect, content type, size, or schema: reject the response and do not persist or forward it.

## Testable Requirements
- Given duplicate messages with the same event ID, only one side effect commits and later deliveries are deduped.
- Given a stale timestamp, invalid signature, wrong audience, wrong tenant, or mutated payload, the consumer rejects before side effects.
- Given a downstream auth failure during a retry, the worker does not bypass policy or swap to a broader credential.
- Given a third-party API response containing SQL or script-like payloads, downstream storage and rendering paths treat it as untrusted input.
- Given a webhook URL pointing to loopback, metadata service, private network, or a redirect to those targets, the registration or callback fails closed.

## Repo-Local Anchors
- This template currently has HTTP, config, PostgreSQL, telemetry, and health/ping surfaces, but no committed queue or worker runtime. Async security requirements should mark queue/worker assumptions explicitly until such infrastructure exists.
- `cmd/service/internal/bootstrap` and `internal/infra/http` own startup/shutdown and HTTP boundaries; new workers should define equivalent lifecycle and security boundary ownership.

## Exa Source Links
- OWASP Threat Modeling Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Threat_Modeling_Cheat_Sheet.html
- OWASP REST Security Cheat Sheet for JWT, access control, and out-of-order API execution: https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html
- OWASP API10:2023 Unsafe Consumption of APIs: https://owasp.org/API-Security/editions/2023/en/0xaa-unsafe-consumption-of-apis/
- OWASP API7:2023 Server Side Request Forgery: https://owasp.org/API-Security/editions/2023/en/0xa7-server-side-request-forgery/
- OWASP OAuth2 Cheat Sheet for token audience, scope, and sender-constrained token guidance: https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html
- Go `net/http` documentation for client timeouts, transports, and redirects: https://pkg.go.dev/net/http
