# Reference Selector

State what unsafe assumption the selected reference will change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| HTTP/config/env/CLI/message/cache/provider input is parsed or normalized before action. | [trust-boundary-and-input-validation-review.md](trust-boundary-and-input-validation-review.md) | Require typed bounded pre-side-effect validation instead of sanitize-later or internal trust. |
| Identity, tenant propagation, object lookup, admin checks, or access failure changes. | [authz-tenant-and-object-access-review.md](authz-tenant-and-object-access-review.md) | Separate authn, authz, tenant, and object ownership. |
| Cookie sessions, credentialed CORS, browser routes, CSRF, or cookie flags change. | [browser-session-cors-and-csrf-review.md](browser-session-cors-and-csrf-review.md) | Trace browser-state attacks instead of treating CORS or server auth alone as sufficient. |
| JWT/header identity, API/session/reset/invite tokens, password reset/storage/hashing changes. | [token-and-credential-flow-review.md](token-and-credential-flow-review.md) | Inspect verification, entropy, replay, storage, and hashing rather than only throttling. |
| SQL/query/filter/template/subprocess/interpreter syntax uses caller input. | [injection-query-and-command-safety.md](injection-query-and-command-safety.md) | Trace attacker data to syntax and choose bind/allowlist/no-shell correction. |
| Caller/tenant/webhook/config data influences outbound URLs, redirects, callbacks, or network targets. | [ssrf-outbound-and-redirect-safety.md](ssrf-outbound-and-redirect-safety.md) | Bound scheme, host, redirects, DNS/IP class, time, and response size instead of merely parsing a URL. |
| Paths, upload names/content, multipart, archives, static/download/temp/config files change. | [path-upload-and-filesystem-safety.md](path-upload-and-filesystem-safety.md) | Constrain access to an owned root and isolated storage instead of lexical cleanup. |
| Secrets, PII, auth headers, DSNs, errors, telemetry, panic/debug/admin endpoints, or deployment policy change. | [secrets-pii-and-telemetry-disclosure.md](secrets-pii-and-telemetry-disclosure.md) | Review concrete disclosure sinks and bounded redaction. |
| Size, pagination/filter complexity, retries, fan-out, queues, file work, OTP/reset/webhook/provider cost, or rate limits change. | [abuse-resistance-and-resource-bounds.md](abuse-resistance-and-resource-bounds.md) | Name the exhausted resource and enforce limits before work. |
