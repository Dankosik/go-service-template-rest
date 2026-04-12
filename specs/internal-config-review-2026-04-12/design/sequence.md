# Sequence

## Config load path after fixes

1. `LoadDetailedWithContext` creates the load context as today.
2. `loadKoanf` loads defaults, files, and env as today.
3. `loadConfigFile` trims the requested path and rejects empty input before `filepath.Clean`.
4. `isLocalEnvironmentHint` reads the env name derived from the namespace mapping for `app.env`.
5. `buildSnapshot` reads values into the typed `Config` snapshot.
6. Numeric parse helpers reject fractional, non-finite, and out-of-range integer float inputs before converting.
7. Float parse helpers reject non-finite values before validation, so string `NaN`/`Inf` inputs return the existing parse-error path.
8. `validateConfig` keeps existing validation order, with Redis and Mongo host-port checks requiring numeric TCP ports.
9. `LoadDetailedWithContext` returns the same `Config` and `LoadReport` shapes as today.

## Bootstrap dependency error path after fixes

1. Bootstrap code detects dependency init or network-policy failures as today.
2. Bootstrap wraps those failures with its own dependency initialization sentinel.
3. Logs, metrics, and spans continue using `"dependency_init"`.
4. `internal/config.ErrorType` remains responsible only for config-load errors and is not used for dependency-init classification.

## Failure and proof points

- `APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG=NaN` must fail config loading as a parse error.
- `redis.addr=localhost:notaport`, `localhost:0`, and `localhost:65536` must fail when Redis is enabled.
- `mongodb://localhost:notaport/app`, `mongodb://localhost:0/app`, and `mongodb://localhost:65536/app` must fail parseability validation when Mongo is enabled.
- Bare Mongo hosts and IPv6 hosts without ports should keep defaulting to `27017`.
- Whitespace-only explicit config file paths should fail as empty path, not fall through to `"."` policy checks.
- Bootstrap dependency errors must remain inspectable with `errors.Is(err, <bootstrap dependency sentinel>)`.
