# Reference Selector

Load the rubric whose symptom would materially change the security decision.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Trust boundaries, attacker paths, or boundary ownership are implicit. | [trust-boundary-threat-modeling.md](trust-boundary-threat-modeling.md) | Choose named boundary, attacker-path, enforcement, and proof requirements instead of generic "use auth" or "validate input" advice. |
| Identity, authorization, tenant, object, property, or admin rules are in scope. | [authentication-authorization-tenant-isolation.md](authentication-authorization-tenant-isolation.md) | Choose caller/subject/tenant-bound access rules instead of role-only checks, untrusted headers, or `subject_id == path_id` shortcuts. |
| JSON/input parsing, SQL or interpreter use, outbound URLs, webhooks, or sanitized errors are in scope. | [input-output-injection-and-ssrf-controls.md](input-output-injection-and-ssrf-controls.md) | Choose strict parser, allowlist, SSRF dial, and sanitized output requirements instead of denylist validation, late validation, or raw error relay. |
| REST/OpenAPI status codes, CORS, problem responses, method policy, request limits, retry/idempotency, or browser-visible headers are in scope. | [api-facing-security-semantics.md](api-facing-security-semantics.md) | Choose contract-visible fail-closed semantics instead of ad hoc status codes, `200` error bodies, permissive CORS, or retry ambiguity. |
| Queues, workers, outbox/inbox, webhooks, callbacks, cross-service calls, third-party APIs, or retries are in scope. | [async-distributed-security.md](async-distributed-security.md) | Choose authenticity, replay, scoped credential, and step-authorization rules instead of trusting internal queues or propagating raw bearer tokens. |
| Sensitive data, privacy, cache keys, DB access, secrets, config source policy, logging, redaction, or telemetry fields are in scope. | [data-cache-and-secret-handling.md](data-cache-and-secret-handling.md) | Choose classification, minimization, cache scoping, secret-source, and redaction requirements instead of shared caches, secret config, or log leakage. |
| Rate limits, expensive operations, bulk work, provider-cost triggers, repeated attempts, or resource exhaustion are in scope. | [resource-abuse-and-cost-controls.md](resource-abuse-and-cost-controls.md) | Choose principal/tenant-scoped budgets and cheap pre-side-effect gates instead of one global rate limit or reliability-only overload wording. |
| Security decisions need proof obligations, test matrices, abuse-path checks, or scanner-vs-test separation. | [security-negative-path-verification.md](security-negative-path-verification.md) | Choose concrete negative-path and no-side-effect proof obligations instead of "covered by integration tests" or scanner-only confidence. |

Use repository API, transport, config, `SECURITY.md`, and CI authorities. Record absent identity, gateway/mesh, secret, cache/queue, third-party, or enforcement guarantees as assumptions or blockers.
