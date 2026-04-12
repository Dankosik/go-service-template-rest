# Sequence

## Implementation Sequence

1. Add `internal/observability/otelconfig`.
   - Define OTel sampler/protocol constants and defaults.
   - Add pure validation or normalization helpers only if they reduce duplication at call sites.
   - Add focused package tests.
   - Update repository architecture/structure docs to name the new package boundary.

2. Update `internal/config`.
   - Replace string literals in OTel defaults and validation with `otelconfig`.
   - Add or preserve validation so resource identity fields consumed by telemetry do not depend on telemetry fallback defaults.
   - Preserve existing validation error wrapping with `ErrValidate`.
   - Keep config tests green and update expected values only if constants change the exact source expression, not behavior.

3. Update `internal/infra/telemetry` tracing.
   - Remove `resource.WithFromEnv()`.
   - Remove fallback defaults for resource identity fields that `internal/config` owns.
   - Use `otelconfig` for sampler/protocol vocabulary.
   - Reject non-finite sampler ratios before SDK sampler construction.
   - Add a resource-construction test or equivalent regression test proving OTEL env resource attributes are not implicitly applied.
   - Add NaN and infinity sampler tests.

4. Update `internal/infra/telemetry` metrics and bootstrap call sites.
   - Add intent-named methods for ready/blocked startup dependency status.
   - Update bootstrap call sites.
   - Move telemetry init failure reason strings into telemetry constants and update bootstrap classification.
   - Keep metric output labels identical.

5. Update `internal/infra/http` server.
   - Add uninitialized-server error and receiver guard.
   - Add nil receiver and zero-value tests.
   - Preserve `ErrNilListener` for initialized `Serve(nil)`.

6. Update `internal/infra/postgres` fixture.
   - Delete the unused transaction helper and transaction-only tests/fakes.
   - Remove now-unused imports, fields, and interfaces.
   - Keep `Create` and `ListRecent` tests plus SQLC integration behavior.

7. HTTP route metadata cleanup.
   - Merge manual root route reason metadata into the route declaration and update tests to derive the lookup from that table.
   - Do not change route behavior.

8. Run validation.
   - Use `test-plan.md` for focused and aggregate commands.

## Failure Points

- Import cycle after adding `otelconfig`: reopen design and adjust package placement; do not create `common`.
- OTel env behavior is required by product/ops policy: reopen specification and config-source policy rather than silently keeping `resource.WithFromEnv()`.
- Postgres transaction fixture is used outside tests: stop and reassess scope before deletion.
- Metric labels change unexpectedly: treat as regression and preserve existing labels.
