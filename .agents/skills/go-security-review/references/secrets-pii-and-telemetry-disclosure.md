# Secrets, PII, And Telemetry Disclosure Review

## When To Load
Load this when changed Go code handles credentials, tokens, DSNs, auth headers, PII, config loading, error payloads, logs, traces, metrics labels, panic recovery, debug endpoints, telemetry exporters, or deployment policy files.

## Attacker Preconditions
- The attacker can cause an error, validation failure, panic, auth failure, malformed config, upstream failure, or unusual request that includes sensitive data.
- The sensitive value can appear in a client response, log event, trace attribute, metric label, config file, repository file, CI output, or deployment policy.
- The attacker or an unauthorized operator can read that output, or the output is retained in a broader telemetry system.

## Review Signals
- Client-visible errors include raw stack traces, SQL text, DSNs, auth headers, tokens, config values, raw upstream bodies, or filesystem internals.
- Logs or traces include `Authorization`, cookies, reset tokens, API keys, DSNs, OTLP headers, raw request bodies, or high-risk PII.
- Metrics labels use user-controlled IDs, paths, emails, tenant IDs, tokens, or unbounded error strings.
- Secret-like config keys are newly allowed in YAML or committed deployment files.
- `railway.toml`, examples, tests, or docs include real-looking secrets instead of placeholders.
- Panic recovery logs raw values whose type may carry secrets.

## Bad Finding Examples
- "Do not log secrets."
- "This leaks PII."
- "Redact telemetry."

These need the exact output sink, sensitive field, reader, and correction.

## Good Finding Examples
- "[high] [go-security-review] internal/config/validate.go:126 Axis: Secrets And Error Disclosure; the new DSN parse error wraps the raw `postgres.dsn` value into the returned error. A malformed env value can expose credentials to startup logs and CI output. Return the parse failure with the config key name only and assert the raw DSN substring is absent in tests."
- "[medium] [go-security-review] internal/infra/http/middleware.go:151 Axis: Telemetry Disclosure; access logs now record `r.URL.RawQuery`, which can include tokens and email addresses from client requests. Any log reader can recover sensitive query data. Keep the low-cardinality route label and status, or log a redacted allowlist of query fields."
- "[medium] [go-security-review] internal/infra/telemetry/metrics.go:82 Axis: Telemetry Disclosure; the new metric label uses `tenant_id` directly. This leaks tenant identifiers and creates unbounded label cardinality under attacker-controlled tenants. Use route/status/error-class labels only and put tenant correlation in access-controlled logs if required."

## Smallest Safe Correction
- Return sanitized problem details to clients; include a request or correlation ID instead of raw internals.
- Log security-relevant events with stable metadata, but mask or omit secrets, tokens, DSNs, auth headers, raw bodies, and high-risk PII.
- Use low-cardinality metric labels: method, route pattern, status code, stage, and error class. Avoid raw paths, query strings, user IDs, tenant IDs, and error text.
- Keep secrets in environment or managed secret stores, not YAML or deployment policy files.
- Preserve repo policy rejecting secret-like config keys in YAML: `dsn`, `password`, `token`, `secret`, `authorization`, and `otlp_headers`.
- Use placeholders in tests and docs unless the value is clearly fake and cannot be mistaken for a live credential.

## Validation Ideas
- Add tests that assert raw secret-like values are absent from errors and logs.
- Add metrics tests that assert labels use route templates rather than raw paths or identifiers.
- Run `make secrets-scan` after touching config, docs, CI, deployment, or examples.
- Run `make go-security` when telemetry or config code changed.

## Repo-Local Anchors
- `docs/configuration-source-policy.md` requires non-secret YAML and secret values from `APP__...` environment variables.
- `internal/config/load_koanf.go` rejects secret-like YAML keys and hardens non-local file config.
- `railway.toml` is documented as non-secret deployment policy only; Railway variables/secrets hold secrets.
- `Makefile` provides `make secrets-scan` with gitleaks redaction.
- `internal/infra/http/problem.go` uses sanitized problem details and request IDs.
- `internal/infra/telemetry/metrics.go` uses low-cardinality HTTP labels.

## Exa Source Links
- OWASP Logging Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
- OWASP Secrets Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- OWASP Authorization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html
- Go `net/http` package docs: https://pkg.go.dev/net/http
