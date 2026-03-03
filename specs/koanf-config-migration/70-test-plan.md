# 70 Test Plan

## Scope
- Unit/integration checks for config load pipeline, strictness, required-if-enabled rules, and source hardening.

## Core Suites
1. Precedence determinism
- defaults < file < overlays < `APP__...` env.
- repeated loads produce identical snapshots.

2. Namespace-only behavior
- flat non-canonical keys do not change effective config.

3. Strictness and unknown keys
- strict mode rejects unknown canonical keys.
- permissive mode reports unknown-key warnings.

4. Required-if-enabled and parsing
- postgres/mongo required secret checks.
- parse classifier for malformed durations/YAML.

5. File hardening and secret policy
- non-local restrictions (allowed roots, symlink, permissions).
- secret-like values in YAML are rejected.

## Mandatory Local Evidence
- `go test ./...`
- `go vet ./...`
