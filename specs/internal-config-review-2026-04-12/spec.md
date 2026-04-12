# internal/config accepted review fixes

## Context

The review of `internal/config` on 2026-04-12 used read-only subagents for Go idiomaticity, simplification/readability, and design maintainability, then reconciled their output against repository evidence.

The accepted issues are bounded but worth fixing before they spread:

- integer parsing accepts float inputs whose upper bounds are unsafe near 64-bit limits;
- `NaN` can pass the OTel sampler argument range check;
- `net.SplitHostPort` is used as if it validates numeric TCP ports;
- `loadConfigFile` cleans a path before checking whether the trimmed path is empty;
- Redis mode normalization has two owners;
- the environment key for `app.env` is hardcoded instead of derived from the namespace mapping rule;
- `ErrorType(nil)` reports `"load"`;
- `internal/config` owns the bootstrap-only dependency initialization sentinel.

## Scope / Non-goals

In scope:

- Fix the accepted `internal/config` parsing and validation findings.
- Keep config errors inspectable with `errors.Is`.
- Move bootstrap dependency initialization error ownership out of `internal/config` and into `cmd/service/internal/bootstrap`.
- Add focused regression tests for each behavior change.
- Preserve existing user-facing config keys, precedence, default values, secret-source policy, and `LoadDetailed` report shape.

Out of scope:

- Do not implement in this session.
- Do not move `MongoProbeAddress` out of `internal/config`; `docs/configuration-source-policy.md` explicitly says it belongs to the guard-only config path for now.
- Do not change local/non-local symlink policy without a separate security/source-policy decision.
- Do not replace the manual snapshot builder or validation file with a reflection/generic framework.
- Do not add a `test-plan.md` or `rollout.md`; there is no migration or rollout choreography.

## Constraints

- `go.mod` reports Go `1.26.1`; local verification ran on `darwin/arm64`.
- Error messages must continue avoiding raw config values, especially secret-like values.
- Runtime telemetry labels such as `"dependency_init"` may stay stable even after the sentinel moves to bootstrap.
- Numeric TCP port validation should be deterministic across operating systems.
- Future implementation must not touch the unrelated deleted files under `specs/template-readiness-review`.

## Decisions

- Reject non-finite floats in `parseFloat64`; string inputs such as `NaN` and `Inf` should fail as `ErrParse` before config values reach range validation.
- Preserve support for integer values delivered as float inputs, but make conversion safe:
  - reject non-finite floats;
  - reject fractional floats;
  - reject values below the target minimum or at/above the first unrepresentable upper bound before converting to `int` or `int64`.
- Validate Redis and Mongo host-port inputs with numeric TCP port rules. Use `strconv.ParseUint` and accept only `1..65535`; do not use `net.LookupPort`, because service-name acceptance and OS service databases would make config validation less deterministic.
- Keep bare Mongo host handling stable: `normalizeMongoProbeAddress` should still add the default Mongo port for a bare host or IPv6 host without a port.
- Fix `loadConfigFile` by trimming first, rejecting the empty string, then applying `filepath.Clean`.
- Add one production helper for namespace env names, and use it for the `app.env` lookup used by local/non-local file policy.
- Add one Redis mode normalization owner and use it from both `buildSnapshot` and `RedisConfig.ModeValue`.
- Make `ErrorType(nil)` return a no-error value, preferably the empty string, and cover it in tests.
- Move dependency-init error ownership to bootstrap. `internal/config` should keep only config load/parse/validate/strict/secret sentinels and classifications.

## Open Questions / Assumptions

- Assumption: Redis and Mongo config ports should be numeric only. If service-name ports such as `localhost:http` are desired, reopen this spec before implementation.
- Assumption: moving the dependency-init sentinel to an unexported bootstrap sentinel is acceptable because all current call sites are in `cmd/service/internal/bootstrap` tests or implementation.
- Deferred: local symlink behavior currently appears stricter than the docs' non-local symlink wording. This is intentionally not part of the implementation plan because changing it needs a security/source-policy decision.

## Plan Summary / Link

Implementation should follow `plan.md` and the executable ledger in `tasks.md`. The next session starts from `workflow-plans/implementation-phase-1.md`.

## Validation

Baseline evidence before planning:

- `go test ./internal/config` passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap` passed.
- `go env GOOS GOARCH` returned `darwin arm64`.

Expected post-implementation proof:

- `go test -count=1 ./internal/config ./cmd/service/internal/bootstrap`.
- `rg -n "ErrDependencyInit" internal/config cmd/service/internal/bootstrap` returns no matches.
- Focused new tests for float bounds, non-finite floats with parse-error classification, numeric TCP ports, empty config path handling, Redis mode normalization, namespace env-name mapping, `ErrorType(nil)`, and bootstrap-owned dependency init errors.

## Outcome

Implemented in `implementation-phase-1`; all task-ledger items `T001` through `T008` are complete.
