# Secrets, PII, And Telemetry Disclosure Review

## Behavior Change Thesis
When loaded for symptom "sensitive data can leave through errors, config, logs, traces, metrics, docs, or deployment files," this file makes the model name the exact disclosure sink and redaction/minimization fix instead of likely mistake broad "do not log secrets" advice.

## When To Load
Load this when changed Go code handles credentials, tokens, DSNs, auth headers, cookies, reset tokens, PII, config loading, error payloads, logs, traces, metrics labels, panic recovery, debug endpoints, telemetry exporters, docs, examples, tests, or deployment policy files.

If the primary issue is token generation, token verification, password hashing, or account-recovery lifecycle, load `token-and-credential-flow-review.md`; use this file for where the sensitive value is exposed or retained.

## Decision Rubric
- Name the sink: client response, startup log, request log, trace attribute, metric label, panic log, config file, CI output, docs, examples, or deployment policy.
- Name who can read it: caller, unauthorized operator, broader telemetry readers, CI logs, repository readers, or public clients.
- Treat raw stack traces, SQL text, DSNs, auth headers, cookies, tokens, raw upstream bodies, raw request bodies, and high-risk PII as unsafe in broad sinks.
- Use low-cardinality metric labels: method, route pattern, status code, stage, and error class. Avoid raw paths, queries, IDs, tenant IDs, emails, tokens, and error strings.
- Keep secrets in environment or managed secret stores, not YAML or deployment policy files.
- Preserve repo policy rejecting secret-like YAML keys such as `dsn`, `password`, `token`, `secret`, `authorization`, and `otlp_headers`.
- Use obviously fake placeholders in tests and docs; do not add live-looking credentials.

## Imitate
```text
[high] [go-security-review] internal/config/validate.go:126
Issue: Axis: Secrets And Error Disclosure; the new DSN parse error wraps the raw `postgres.dsn` value into the returned error.
Impact: A malformed env value can expose credentials to startup logs and CI output.
Suggested fix: Return the parse failure with the config key name only and assert the raw DSN substring is absent in tests.
Reference: config source and error disclosure policy.
```

Copy this shape when a secret value is included in an error path.

```text
[medium] [go-security-review] internal/infra/http/middleware.go:151
Issue: Axis: Telemetry Disclosure; access logs now record `r.URL.RawQuery`, which can include tokens and email addresses from client requests.
Impact: Any log reader can recover sensitive query data.
Suggested fix: Keep the low-cardinality route label and status, or log a redacted allowlist of query fields.
Reference: HTTP telemetry redaction boundary.
```

Copy this shape when the sink is operational telemetry rather than a client response.

```text
[medium] [go-security-review] internal/infra/telemetry/metrics.go:82
Issue: Axis: Telemetry Disclosure; the new metric label uses `tenant_id` directly.
Impact: Tenant identifiers leak into a broad metrics sink and create unbounded cardinality under attacker-controlled tenants.
Suggested fix: Use route, status, and error-class labels only; put tenant correlation in access-controlled logs if required.
Reference: metric label contract.
```

Copy this shape when the same field is both sensitive and high-cardinality.

## Reject
```text
Issue: Do not log secrets.
```

Reject because it does not name the value, sink, reader, or safe replacement.

```text
Suggested fix: Hash the tenant ID and keep it as a metric label.
```

Reject when the metric still has unbounded or policy-sensitive cardinality.

## Agent Traps
- Do not call request IDs sensitive by default; they are usually safe correlation handles, not authorization inputs.
- Do not replace every log with silence; security events still need stable, sanitized metadata.
- Do not assume test credentials are harmless if they look real enough to trigger scanners or operator reuse.
- Do not let redaction happen after the value is already attached to a trace, metric, or wrapped error.
- Do not duplicate token lifecycle findings here; use the token reference when entropy, verification, hashing, expiry, or replay is primary.

## Validation Shape
- Add tests that raw secret-like values are absent from errors, logs, traces, and client responses.
- Add metrics tests that labels use route templates and bounded error classes rather than raw paths or identifiers.
- Run `make secrets-scan` after touching config, docs, CI, deployment, examples, or fixtures.
- Run `make go-security` when telemetry or config code changed.

## Repo-Local Anchors
- `docs/configuration-source-policy.md` requires non-secret YAML and secret values from `APP__...` environment variables.
- `internal/config/load_koanf.go` rejects secret-like YAML keys and hardens non-local file config.
- `railway.toml` is non-secret deployment policy; Railway variables/secrets hold secrets.
- `Makefile` provides `make secrets-scan` with gitleaks redaction.
- `internal/infra/http/problem.go` uses sanitized problem details and request IDs.
- `internal/infra/telemetry/metrics.go` uses low-cardinality HTTP labels.
