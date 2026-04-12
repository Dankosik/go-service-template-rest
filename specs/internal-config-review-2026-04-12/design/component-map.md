# Component map

## `internal/config/parse_helpers.go`

Change:

- Add finite-float checks for `parseFloat64`, `intFromFloat64`, and `int64FromFloat64`.
- Replace `float64(math.MaxInt)` / `float64(math.MaxInt64)` inclusive upper-bound checks with conversion-safe bounds.

Stable:

- Keep existing mixed numeric input support.
- Keep error messages value-free.

## `internal/config/validate.go`

Change:

- Add deterministic numeric TCP port validation for `redis.addr`.
- Reuse the same validation inside `normalizeMongoProbeAddress` when the Mongo host already has a port.
- Keep Mongo bare-host defaulting to `27017`.
- Optionally keep a defensive non-finite sampler check even though `parseFloat64` should reject string `NaN`/`Inf` first.

Stable:

- Keep `MongoProbeAddress` exported from `internal/config`.
- Keep sampler `NaN`/`Inf` from string config classified as `ErrParse` through `readFloat64Into`; direct defensive validation may still return `ErrValidate` if a typed non-finite value reaches `validateSampler`.

## `internal/config/load_koanf.go`

Change:

- Trim `path`, reject empty, then clean it.
- Replace the hardcoded `APP__APP__ENV` lookup with a helper-derived env name for the `app.env` config key.

Stable:

- Keep config source precedence and file hardening policy.
- Do not change local/non-local symlink behavior in this task.

## `internal/config/snapshot.go` and `internal/config/redis.go`

Change:

- Introduce one unexported Redis mode normalization helper and use it from snapshot construction and `RedisConfig.ModeValue`.

Stable:

- Keep stored `Redis.Mode` normalized to current behavior unless a test proves a better compatibility path.
- Keep `StoreMode` and `RedisReadinessProbeRequired` behavior unchanged.

## `internal/config/errors.go`

Change:

- Add an explicit nil branch to `ErrorType`.
- Remove dependency-init classification from config after bootstrap owns that sentinel.

Stable:

- Keep config load/parse/validate/strict/secret error classification order.
- Keep unknown non-nil errors classified as `"load"` unless implementation uncovers a stronger local convention.

## `cmd/service/internal/bootstrap`

Change:

- Add a bootstrap-owned dependency initialization sentinel, likely unexported because bootstrap tests are same-package.
- Replace all `config.ErrDependencyInit` uses in bootstrap implementation and tests with the bootstrap-owned sentinel.

Stable:

- Keep telemetry/log labels as `"dependency_init"`.
- Keep existing retry and startup admission behavior.

## Tests

Change:

- Add or update tests in `internal/config/config_test.go`.
- Add or update tests in `cmd/service/internal/bootstrap/*_test.go` for the sentinel ownership move.

Stable:

- Keep tests package-local; no external test package conversion is required.
