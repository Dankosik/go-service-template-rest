# Implementation Plan

## Execution Context

This plan is for a future implementation session. It consumes the decisions in `spec.md` and the technical context in `design/`.

The implementation is a single lightweight local phase because the change is bounded to `internal/config`, one YAML baseline file, one policy doc, and package tests. No OpenAPI, database, generated code, or rollout choreography is expected.

## Phase Plan

### Phase 1: Config Review Fixes

- Objective: land the package-local fixes from the review while preserving existing valid config behavior.
- Depends on: approved handoff artifacts in this directory.
- Task ledger: `tasks.md` T001 through T008.
- Change surface:
  - `internal/config/load_koanf.go`
  - `internal/config/validate.go`
  - `internal/config/defaults.go` or a new package-local schema helper file under `internal/config`
  - `internal/config/config_test.go`
  - `env/config/default.yaml`
  - `docs/configuration-source-policy.md`
- Acceptance criteria:
  - Empty `APP__...` values are explicit final overrides.
  - Required empty values fail parse/validation instead of falling back to defaults.
  - Mongo probe normalization rejects empty and malformed bracket hosts.
  - Strict-mode known-key validation is based on typed config keys, not `defaultValues()`.
  - `env/config/default.yaml` includes `otlp_traces_endpoint`.
  - File-policy mode naming is behavior-preserving if implemented.
  - `ErrorType` fallback remains unchanged.
- Planned verification:
  - `go test ./internal/config`
  - targeted `go test ./internal/config -run 'TestMongoURI|TestMongoProbeAddress|TestKnownConfigKeys|TestEnv|TestLoadDefaults'`
  - `go test ./cmd/service/internal/bootstrap`
  - recommended final `go test ./...`
- Exit criteria: tests pass and no changes outside the stated surfaces are needed.

## Cross-Phase Validation Plan

Use package tests to prove all behavior changes. Use a broader test run because empty env semantics can affect bootstrap tests that depend on config reset helpers.

No manual deployment validation is expected.

## Implementation Readiness

Status: `PASS`

Rationale: all planning-critical decisions are recorded. The only policy-visible decision, empty env values as explicit overrides, is selected in `spec.md` and tied to a docs update and test obligations.

## Blockers / Assumptions

- Assumes empty `APP__...` means explicit override, not unset.
- Assumes deriving known keys from `Config` tags is acceptable in production package code.
- Assumes `ErrorType` metric-label behavior should remain stable.

## Handoffs / Reopen Conditions

Start the future implementation from `tasks.md`.

Reopen design before coding if the implementation must touch bootstrap lifecycle, telemetry metric label contracts, runtime Redis/Mongo adapters, or non-local file security behavior beyond behavior-preserving naming.
