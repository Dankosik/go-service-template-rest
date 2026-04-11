# Data, Cache, Privacy, And Secret Handling Examples

## When To Load
Load this when requirements touch sensitive data, privacy, data minimization, retention, logging/redaction, cache keys, DB privileges, tenant-scoped data, secrets, key management, config source policy, telemetry headers, or CI secret scanning.

## Selected Controls
- Classify data before choosing storage, cache, logging, or telemetry rules. Treat personal data, credentials, tokens, API keys, connection strings, health data, financial data, and business secrets as sensitive by default.
- Minimize storage and cache copies of sensitive data. Avoid storing sensitive data when not required and define deletion/retention when it is required.
- Keep tenant, subject, scope, and data-classification dimensions in cache keys for tenant-scoped or permission-scoped data.
- Define cache fail-open versus fail-closed behavior. Authorization, tenant isolation, and secret lookups should fail closed unless a safer contract is explicitly proven.
- Store secrets in approved secret stores or environment secret channels, not YAML, code, logs, generated artifacts, cache values, or test snapshots.
- Require secret lifecycle requirements: owner, purpose, consumers, rotation, revocation, expiry when possible, and audit trail.
- Redact tokens, credentials, DSNs, `Authorization` headers, OTLP headers, and sensitive personal data in logs, traces, errors, and test output.
- Use repository-native security gates where applicable: `govulncheck`, `gosec`, `gitleaks`, and container scanning.

## Rejected Controls
- Reject cache keys that include only object ID when data access also depends on tenant, caller, role, scope, relationship, or property filtering.
- Reject shared caches for sensitive or tenant-scoped data without explicit key isolation, TTL, and invalidation rules.
- Reject secrets in config YAML, `railway.toml`, examples, fixtures, logs, or task artifacts.
- Reject reversible encryption for passwords or custom cryptography.
- Reject logging full request bodies, SQL text with sensitive values, config snapshots, or outbound headers by default.
- Reject treating encrypted data as authorization. Encrypted identifiers still require access control.

## Fail-Closed Examples
- Cache miss or cache service failure for an authorization decision causes a policy lookup or deny, not permit.
- Tenant-scoped cache key missing tenant binding denies cache use and forces a scoped source-of-truth read.
- Secret manager or required environment secret unavailable at startup blocks the dependent feature instead of falling back to insecure defaults.
- Redaction classifier failure causes logs to omit the field rather than logging the raw value.
- Data-classification unknown means no shared cache and no telemetry value export until classification is resolved.

## Testable Requirements
- Given two tenants with the same object ID, cache lookups never return the other tenant's value.
- Given role/property-filtered responses, cached values are scoped to the authorization context or only store public fields.
- Given secret-like keys in YAML, config loading rejects them according to repo policy.
- Given parse or validation errors that include secret-like input values, returned errors and logs do not contain raw secrets.
- Given a deleted, revoked, or rotated secret, dependent code stops using the old value within the documented rotation or reload window.
- Given CI or local security checks, `make go-security` and `make secrets-scan` remain the expected proof path when code or config changes are in scope.

## Repo-Local Anchors
- `docs/configuration-source-policy.md` states YAML is for non-secret defaults and environment variables are for secret values.
- `internal/config/load_koanf.go` enforces secret-like key rejection, allowed roots, symlink rejection, write-permission checks, and config size limits outside local mode.
- `internal/config/config_test.go` includes negative tests for secret policy, raw secret-like parse errors, symlink paths, world-writable configs, and outside-root configs.
- `railway.toml` is documented as non-secret deployment policy; Railway variables hold secrets.
- `Makefile` defines `go-security` using `govulncheck` and `gosec`, plus `secrets-scan` using `gitleaks`.

## Exa Source Links
- OWASP Secrets Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- OWASP Logging Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
- OWASP Cryptographic Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html
- OWASP Developer Guide, Protect Data Everywhere: https://devguide.owasp.org/en/04-design/02-web-app-checklist/08-protect-data/
- Go avoiding SQL injection risk: https://go.dev/doc/database/sql-injection
- Go executing transactions: https://go.dev/doc/database/execute-transactions
