# Secrets And Disclosure

## When To Load

Load this when a secret, credential, personal field, or internal detail can
reach a response, a log, a trace, a metric label, a config file, or the
repository.

## Behavior Change Thesis

Without this file, the finding is "do not log secrets" and the fix is a
redaction helper. This repository already draws the boundaries; the defects that
survive are the ones that cross them — a secret-shaped key added to YAML, an
upstream cause wrapped into a client-visible error, and a caller-controlled
value promoted to a metric label, which is a disclosure sink and an unbounded
series at the same time.

## Decision Rubric

- `internal/config/secret_policy.go` rejects secret-like keys — `password`,
  `token`, `secret`, `authorization`, `dsn` — in a config file at load time.
  YAML holds non-secret defaults and `APP__...` environment variables are the
  secret channel, per `docs/configuration-source-policy.md`. `railway.toml` is
  deployment policy, not a secret store.
- Sanitized failure is a type here, not a habit: `oidcjwt.Error` carries a
  closed `Kind` and no parser, token, key, endpoint, or provider text. A change
  that wraps a cause into it — or into a `problem` detail crossing the API
  boundary — reopens the leak the type exists to close.
- Metric labels stay low-cardinality and caller-independent: route template,
  method, status, error class. A tenant or user identifier as a label discloses
  it to every metrics reader and makes the series unbounded under an attacker
  who controls the value; correlation belongs in access-controlled logs.
- Request identifiers are correlation handles rather than secrets, and security
  events still need stable sanitized metadata. Silence is not redaction.
- Classify data before choosing its storage, cache, log, or telemetry sink, and
  treat an unknown classification as sensitive until it is decided.
- Redact at the point the value is produced. A wrapped error, a span attribute,
  or a metric label is already published by the time a later redactor runs.

## Reject

- A live-looking credential in a fixture, example, or doc: `make secret-scan`
  runs gitleaks against the change with `.gitleaks.toml` and
  `.gitleaks.baseline.json`, so a plausible value either fails the gate or
  teaches the next reader to reuse it. Obviously fake placeholders do neither.
- Treating a scanner as proof of a privacy or access rule: `make go-security`
  runs `govulncheck` and `gosec`, which cover dependency and pattern classes and
  say nothing about who may read a field.

## Validation Shape

Assert the raw value is absent from the error, the log, the response, and the
telemetry, rather than asserting a redactor was called. `make secret-scan`
covers the change against its base ref and `make secret-scan-history` covers
full history; run the scan after touching config, docs, CI, deployment,
examples, or fixtures.
