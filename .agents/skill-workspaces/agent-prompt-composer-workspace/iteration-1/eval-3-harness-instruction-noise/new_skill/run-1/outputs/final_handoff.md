**Objective**
Investigate and fix a regression in config validation where an empty `DATABASE_URL` may now be accepted when Postgres is enabled, possibly because a default value masks it.

**User Intent And Context**
The user wants a focused bug investigation and fix in the config validation path, not a broad refactor or app rewrite. The likely surface is `internal/config`, and there is probably already a nearby test case in `config_test.go`.

**Confirmed Signals And Exact Identifiers**
- `DATABASE_URL`
- `postgres.enabled`
- `postgres.dsn`
- `internal/config`
- `config_test.go`
- “keep exact error messages stable if tests already assert them”
- “Validation should be focused, not just `run everything`”

**Relevant Repository Context**
- Config loading and validation live in `internal/config/`.
- Current Postgres validation in [`internal/config/validate.go`](/Users/daniil/Projects/Opensource/go-service-template-rest/internal/config/validate.go) rejects empty `postgres.dsn` with `ErrSecretPolicy` when `postgres.enabled=true`, and rejects unparsable DSNs with `ErrValidate`.
- Existing required-if-enabled coverage is in [`internal/config/config_test.go`](/Users/daniil/Projects/Opensource/go-service-template-rest/internal/config/config_test.go), including `postgres_enabled_without_dsn_rejected` and `postgres_enabled_with_dsn_allowed`.
- `DATABASE_URL` is likely mapped into config through the config loading layer, so the bug may be in how defaults or env aliases interact with validation rather than in the validator alone. That is an inference, not yet confirmed.

**Inspect First**
1. Check the `internal/config` load path to see how `DATABASE_URL` becomes `postgres.dsn`.
2. Review the Postgres validation branch in `validate.go`.
3. Inspect nearby tests in `config_test.go` for an empty-string or default-masking case.
4. Confirm whether the expected failure should remain `ErrSecretPolicy` or change to `ErrValidate`; preserve existing message text if tests already rely on it.

**Requested Change / Problem Statement**
Fix the config validation regression so that enabling Postgres with an empty DSN does not pass because of defaults or env masking. The result should keep validation narrow and targeted to the config layer.

**Constraints / Preferences / Non-goals**
- Do not turn this into a full application rewrite.
- Do not broaden scope beyond config validation.
- Keep existing error messages stable where tests assert exact text.
- Prefer the smallest fix that explains and prevents the regression.
- If the bug is already covered by a test, update the minimal assertion needed rather than rewriting the test suite.

**Acceptance Criteria / Expected Outcome**
- A focused fix in `internal/config` prevents empty Postgres DSNs from being accepted when Postgres is enabled.
- A nearby test in `config_test.go` covers the regression, ideally with the exact failing case around `DATABASE_URL` or its mapped config key.
- Existing error-type behavior remains stable unless the current contract clearly requires a change.
- Validation remains narrowly scoped, with no unrelated refactor.