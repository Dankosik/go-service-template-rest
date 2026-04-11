# Data, Cache, Privacy, And Secret Handling

## Behavior Change Thesis
When loaded for sensitive data, cache, privacy, secret, config, logging, or telemetry requirements, this file makes the model choose classification, minimization, cache scoping, secret-source, and redaction requirements instead of likely mistake: shared cache keys, secret config files, encryption-as-authorization, or raw diagnostic leakage.

## When To Load
Load this when requirements touch sensitive data, privacy, data minimization, retention, logging/redaction, cache keys, DB privileges, tenant-scoped data, secrets, key management, config source policy, telemetry headers, or CI secret scanning.

## Decision Rubric
- Classify data before choosing storage, cache, logging, telemetry, or retention behavior. Treat personal data, credentials, tokens, API keys, connection strings, health data, financial data, and business secrets as sensitive by default.
- Minimize copies of sensitive data. If storage or caching is required, define purpose, owner, retention/deletion, access scope, and redaction behavior.
- Include tenant, subject, scope, relationship, property-filter, and data-classification dimensions in cache keys when access to the cached value depends on them.
- Authorization, tenant isolation, and secret lookups fail closed unless a safer contract is explicitly proven.
- Store secrets in approved secret stores or environment secret channels, not YAML, code, logs, generated artifacts, cache values, examples, task artifacts, or test snapshots.
- Define secret lifecycle: owner, purpose, consumers, rotation, revocation, expiry when possible, reload window, and audit trail.
- Redact tokens, credentials, DSNs, `Authorization` headers, OTLP headers, and sensitive personal data in logs, traces, problem responses, and tests.

## Imitate
- "Tenant-scoped cache keys include tenant and authorization context, or cache only public fields; cache service failure for an authorization decision causes source-of-truth lookup or deny, never permit." Copy the access dimensions and fail-closed cache behavior.
- "YAML config may contain non-secret defaults only; secret-like keys are rejected at load time and must be supplied through `APP__...` environment secret channels." Copy the repo-specific secret boundary.
- "Parse errors that include secret-like input values are redacted before returning or logging; classifier failure omits the field." Copy conservative redaction.

## Reject
- "Cache by object ID." Object ID alone is wrong when tenant, caller, role, scope, relationship, or property filtering affects access.
- "Encrypt the identifier, so authorization is covered." Encryption may protect confidentiality; it does not prove the caller may access the object.
- "Put example credentials in fixtures." Examples and task artifacts are not secret stores.
- "Log full request/config snapshots for debugging." Debuggability does not justify raw secrets or sensitive personal data.

## Agent Traps
- Do not treat encrypted data as public data. Access control and classification still apply.
- Do not use shared caches for sensitive or tenant-scoped data without key isolation, TTL, invalidation, and stale-data behavior.
- Do not push privacy and redaction entirely to observability work when the security spec owns whether sensitive data may leave the boundary.

## Validation Shape
- Cross-tenant cache tests with same object IDs prove no tenant bleed.
- Property-filtered response tests prove cached data is scoped to auth context or contains only public fields.
- Config tests prove secret-like keys in YAML are rejected and secret-like parse errors redact raw values.
- Redaction tests prove logs, traces, problem responses, and test output omit tokens, DSNs, `Authorization` headers, and sensitive fields.
- Secret rotation or revocation tests prove dependent code stops using old values within the documented reload window when code/config changes are in scope.

## Repo-Local Anchors
- `docs/configuration-source-policy.md` states YAML is for non-secret defaults and environment variables are for secret values.
- `internal/config/load_koanf.go` enforces secret-like key rejection, allowed roots, symlink rejection, write-permission checks, and config size limits outside local mode.
- `internal/config/config_test.go` includes negative tests for secret policy, raw secret-like parse errors, symlink paths, world-writable configs, and outside-root configs.
- `railway.toml` is documented as non-secret deployment policy; Railway variables hold secrets.
- `Makefile` defines `go-security` using `govulncheck` and `gosec`, plus `secrets-scan` using `gitleaks`.
