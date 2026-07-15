---
name: go-security-review
description: "Use when changed Go crosses a trust boundary or affects authn, authz, tenant or object isolation, browser sessions, CORS/CSRF, tokens, credentials, injection, SSRF, paths, secrets, or abuse controls; Own security-policy enforcement defects; Skip when policy is unset or the primary issue is business meaning, chi wiring, or observability privacy."
---

# Go Security Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Trigger, Scope, And Boundary

Review changed Go trust boundaries, untrusted inputs, authentication, authorization, tenant/object access, browser sessions, CORS/CSRF, tokens/credentials, injection, SSRF, filesystem/uploads, secrets/PII/telemetry, async identity, and resource abuse for concrete exploit or security-contract risk.

Stay read-only and fail-path-first. Do not invent policy when no approved contract exists, call internal traffic trusted by default, turn generic hardening into findings, or redesign identity/API/data/reliability/rollout when local correction cannot own the decision.

## Security Invariants

1. Inputs, internal messages, cache values, provider data, paths, URLs, and identifiers are untrusted until an explicit validated boundary proves otherwise.
2. Authentication, authorization, tenant binding, and object ownership are separate fail-closed checks performed before sensitive reads or side effects.
3. Browser cookie flows pair narrow credentialed CORS with CSRF defenses and hardened cookie/session lifecycle; CORS never substitutes for authorization.
4. Tokens and credentials use cryptographic entropy, complete verification, bounded lifetime/replay, safe storage, strong password hashing, and non-enumerating responses.
5. Interpreter, query, command, URL, redirect, DNS/IP, path, archive, upload, and file boundaries use parameterization/allowlists/root constraints and bounded I/O.
6. Secrets, auth material, PII, stack/SQL/internal topology, and user-controlled high-cardinality values never leak through errors, logs, traces, metrics, panic/debug surfaces, or config.
7. Expensive paths have pre-work limits, deadlines, concurrency/queue bounds, retry/replay controls, and tenant-safe resource accounting.

## Symptom-Driven Reference Selector

State what unsafe assumption the selected reference will change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| HTTP/config/env/CLI/message/cache/provider input is parsed or normalized before action. | [trust-boundary-and-input-validation-review.md](references/trust-boundary-and-input-validation-review.md) | Require typed bounded pre-side-effect validation instead of sanitize-later or internal trust. |
| Identity, tenant propagation, object lookup, admin checks, or access failure changes. | [authz-tenant-and-object-access-review.md](references/authz-tenant-and-object-access-review.md) | Separate authn, authz, tenant, and object ownership. |
| Cookie sessions, credentialed CORS, browser routes, CSRF, or cookie flags change. | [browser-session-cors-and-csrf-review.md](references/browser-session-cors-and-csrf-review.md) | Trace browser-state attacks instead of treating CORS or server auth alone as sufficient. |
| JWT/header identity, API/session/reset/invite tokens, password reset/storage/hashing changes. | [token-and-credential-flow-review.md](references/token-and-credential-flow-review.md) | Inspect verification, entropy, replay, storage, and hashing rather than only throttling. |
| SQL/query/filter/template/subprocess/interpreter syntax uses caller input. | [injection-query-and-command-safety.md](references/injection-query-and-command-safety.md) | Trace attacker data to syntax and choose bind/allowlist/no-shell correction. |
| Caller/tenant/webhook/config data influences outbound URLs, redirects, callbacks, or network targets. | [ssrf-outbound-and-redirect-safety.md](references/ssrf-outbound-and-redirect-safety.md) | Bound scheme, host, redirects, DNS/IP class, time, and response size instead of merely parsing a URL. |
| Paths, upload names/content, multipart, archives, static/download/temp/config files change. | [path-upload-and-filesystem-safety.md](references/path-upload-and-filesystem-safety.md) | Constrain access to an owned root and isolated storage instead of lexical cleanup. |
| Secrets, PII, auth headers, DSNs, errors, telemetry, panic/debug/admin endpoints, or deployment policy change. | [secrets-pii-and-telemetry-disclosure.md](references/secrets-pii-and-telemetry-disclosure.md) | Review concrete disclosure sinks and bounded redaction. |
| Size, pagination/filter complexity, retries, fan-out, queues, file work, OTP/reset/webhook/provider cost, or rate limits change. | [abuse-resistance-and-resource-bounds.md](references/abuse-resistance-and-resource-bounds.md) | Name the exhausted resource and enforce limits before work. |

## Evidence And Domain Finding Rules

Trace attacker-controlled input through validation, identity/tenant context, authorization, query/URL/path construction, side effects, storage, errors, telemetry, async replay, and resource consumption. Each finding adds the security axis, violated control/unsafe assumption, realistic attacker preconditions, affected trust boundary/data asset, exploit impact, and focused negative proof.

Start `Issue` with `Axis:` when it clarifies the risk. `critical` requires a confirmed exploitable high-impact vulnerability; `high` requires strong evidence of a significant security-contract breach. No finding is preferable to generic hardening without an exploit or contract path.

## Success, Escalation, And Stop Conditions

Success means findings are exploit-oriented, fail-closed, evidence-anchored, merge-risk ordered, and bounded to the changed surface with explicit handoffs and proof.

Stop rather than invent policy: unresolved or changed identity, tenant, object, session, credential, disclosure, or abuse controls go to `go-security-spec`; public errors/statuses to `go-api-contract-spec`; data-isolation architecture to `go-data-architecture-spec`; durable trust/replay to `go-distributed-spec`; and reliability/rollout policy to `go-reliability-spec`. Hand performance, test, DB/cache, concurrency, and operability depth to their primary reviewers once the security consequence is stated.
