# Implementation plan

## Execution context

This plan is for a later implementation session. The current session must not edit production code.

The implementation is one bounded phase across `internal/config` and `cmd/service/internal/bootstrap`. It should preserve public config loading APIs and existing runtime behavior except for the accepted stricter validation and ownership cleanup.

## Phase plan

### Phase 1: accepted review fixes

- Objective: land the accepted config parsing, validation, readability, and bootstrap ownership fixes with focused regression coverage.
- Depends on: approved `spec.md` and `design/`.
- Task ledger link: `tasks.md` IDs `T001` through `T008`.
- Acceptance criteria:
  - integer float conversion rejects non-finite and out-of-range values before converting;
  - sampler arg `NaN`/`Inf` cannot pass config loading and string non-finite inputs are classified as parse errors;
  - Redis and Mongo host-port config rejects nonnumeric, zero, and out-of-range ports;
  - whitespace-only explicit config file paths are rejected before `filepath.Clean`;
  - Redis mode normalization has one source of truth;
  - local environment env-key lookup derives from the namespace mapping helper;
  - `ErrorType(nil)` no longer reports a load failure;
  - dependency-init sentinel and tests are bootstrap-owned, while telemetry labels remain stable.
- Change surface:
  - `internal/config/parse_helpers.go`
  - `internal/config/validate.go`
  - `internal/config/load_koanf.go`
  - `internal/config/snapshot.go`
  - `internal/config/redis.go`
  - `internal/config/errors.go`
  - `internal/config/config_test.go`
  - `cmd/service/internal/bootstrap/*.go`
  - `cmd/service/internal/bootstrap/*_test.go`
- Planned verification:
  - `go test ./internal/config ./cmd/service/internal/bootstrap`
  - inspect `rg -n "ErrDependencyInit" internal/config cmd/service/internal/bootstrap` after edits to confirm ownership moved out of `internal/config`.
- Review / checkpoint:
  - after code edits, re-review only the touched surfaces for error contracts, config validation semantics, and bootstrap boundary integrity.
- Exit criteria:
  - verification passes and no new workflow/process artifacts are needed.

## Cross-phase validation plan

- Add direct tests for `parseInt` and `parseInt64` around `float64(math.MaxInt64)` or the target upper-exclusive boundary, plus `math.NaN()` and `math.Inf(1)`.
- Add a config-loading test for `APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG=NaN` or equivalent non-finite input and assert `ErrParse`.
- Add Redis address tests for nonnumeric, zero, and out-of-range ports when Redis is enabled.
- Add Mongo URI tests for nonnumeric, zero, and out-of-range ports when Mongo is enabled, plus keep existing bare-host default behavior.
- Add a direct same-package test for `loadConfigFile` with a whitespace-only path.
- Add or adjust tests proving Redis mode normalization still returns the expected canonical mode.
- Add a test for `ErrorType(nil)`.
- Update bootstrap tests so dependency initialization errors are checked against the bootstrap-owned sentinel.

## Implementation readiness

- Status: `PASS`.
- Accepted risks:
  - The plan intentionally does not resolve local symlink policy ambiguity; changing that requires a separate security/source-policy decision.
  - Removing `config.ErrDependencyInit` is an internal compatibility change, but current repository evidence shows only bootstrap uses it.
- Proof obligations:
  - run `go test ./internal/config ./cmd/service/internal/bootstrap`;
  - run the `rg` ownership check for `ErrDependencyInit`;
  - ensure error messages for invalid numeric config values still do not include raw secret-like values.

## Blockers / Assumptions

- No blocker for the planned implementation.
- Assumption: ports are numeric `1..65535`; service-name ports are not supported.
- Assumption: `MongoProbeAddress` stays in `internal/config`.

## Handoffs / Reopen conditions

Start the next session from `workflow-plans/implementation-phase-1.md` and `tasks.md`.

Reopen specification/design instead of coding if service-name ports, local symlink behavior, `MongoProbeAddress` ownership, or external use of `config.ErrDependencyInit` becomes a requirement.
