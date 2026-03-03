# 50 Security Observability DevOps

## Security
- Trust boundary: external config input (files/env) is untrusted until validated.
- Secret policy: reject secret-like populated keys from YAML in all environments.
- Non-local file hardening:
  - absolute path requirement,
  - allowed roots enforcement,
  - symlink rejection,
  - no group/other writable files,
  - file size limit.

## Observability
- Startup config lifecycle metrics:
  - `config_load_duration_seconds{stage,result}`
  - `config_validation_failures_total{reason}`
  - `config_unknown_key_warnings_total`
  - `config_startup_outcome_total{outcome}`
- Telemetry bootstrap metric:
  - `telemetry_init_failure_total{reason}`
- No key-translation telemetry metrics.

## DevOps
- CI gates for touched scope: `fmt`, `test`, `vet` (and wider gates when required by changed files).
- Docs-drift policy: behavior/contract changes must update docs/spec in same change.
