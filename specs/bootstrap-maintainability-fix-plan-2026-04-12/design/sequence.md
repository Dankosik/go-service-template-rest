# Sequence

## Implementation Sequence

1. Add `startup_rejections_total{reason}` in `internal/infra/telemetry`.
2. Add metric tests for the new series, normalizer, and zero-value no-op behavior.
3. Update bootstrap rejection sites:
   - config load failure increments config failure and startup rejection;
   - policy violation increments startup rejection;
   - dependency init/probe rejection increments startup rejection;
   - HTTP startup rejection increments startup rejection.
4. Update bootstrap tests that currently assert non-config reasons on `config_validation_failures_total`.
5. Add config readiness predicate helpers for Postgres and Mongo.
6. Replace Postgres/Mongo direct feature-flag checks in config validation and bootstrap with the new predicates.
7. Convert `serveHTTPRuntime` to a named argument struct and update call sites.
8. Convert `recordDependencyProbeRejection` to take `startupDependencyProbeLabels`.
9. Rename the explicit declaration bool parser and egress exception validation helper.
10. Run focused validation.

## Runtime Sequence After Implementation

### Config Failure

1. Bootstrap calls `config.LoadDetailedWithContext`.
2. On failure, bootstrap derives `errorType := config.ErrorType(err)`.
3. Bootstrap records config failure metrics with `IncConfigValidationFailure(errorType)`.
4. Bootstrap records startup rejection with `IncStartupRejection("config_" + errorType)` through the chosen mapping.
5. Startup outcome remains `rejected`.
6. The returned error remains wrapped with the config error type and root cause.

### Non-Config Startup Rejection

1. Bootstrap detects policy violation, dependency initialization failure, dependency probe rejection, or HTTP startup failure.
2. The rejection helper records span attributes and logs as before.
3. The helper increments `IncStartupRejection(policy_violation|dependency_init|startup_error)`.
4. The helper increments startup outcome `rejected`.
5. The helper does not increment `IncConfigValidationFailure`.

### Readiness Predicate Use

1. `internal/config.validateReadinessProbeBudgets` uses `cfg.PostgresReadinessProbeRequired()`, `cfg.RedisReadinessProbeRequired()`, and `cfg.MongoReadinessProbeRequired()`.
2. Bootstrap uses the same predicates to decide which probes join `health.New`.
3. The probe set and aggregate readiness budget validation remain aligned.

## Failure Points

- If startup rejection reason normalization maps a planned reason to `other`, tests must fail.
- If non-config rejections still emit `config_validation_failures_total`, tests must fail.
- If readiness predicate helpers drift from feature-flag semantics, config tests must fail.
- If the HTTP runtime args struct changes behavior, existing startup server tests must fail.
