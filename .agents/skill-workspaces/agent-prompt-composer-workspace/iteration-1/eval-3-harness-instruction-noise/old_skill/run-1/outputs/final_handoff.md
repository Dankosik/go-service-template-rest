Objective  
Investigate and fix the config validation path so `DATABASE_URL` cannot be silently accepted when Postgres is enabled, while keeping the existing error strings stable.

User Intent And Context  
The user believes a recent cleanup changed config validation behavior. The suspected symptom is that an empty `DATABASE_URL` is now accepted when Postgres is enabled, possibly because a default is masking the missing value. This should be handled as a focused config-validation bugfix, not a broad app rewrite.

Confirmed Signals And Exact Identifiers  
- `DATABASE_URL`
- `internal/config`
- `config_test.go`
- `postgres.enabled`
- `postgres.dsn`
- `ErrValidate`
- `ErrSecretPolicy`
- “keep exact error messages stable if tests already assert them”
- “validation should be focused, not just `run everything`”

Relevant Repository Context  
- Config loading and validation live in `internal/config/`.
- Postgres validation currently happens in `internal/config/validate.go`.
- Existing tests for required-if-enabled behavior are already present in `internal/config/config_test.go`.
- `validatePostgres` currently rejects empty `postgres.dsn` only when `cfg.Enabled` is true, and parses the DSN when enabled.
- The repo treats `internal/config/` as the source of truth for building one validated runtime config snapshot.

Inspect First  
- `internal/config/validate.go`
- `internal/config/config_test.go`
- `internal/config/defaults.go`
- `internal/config/config.go` if the load/validation ordering matters

Requested Change / Problem Statement  
Find why an empty Postgres connection value is getting through after the last cleanup, determine whether the issue is in defaults, env mapping, or validation ordering, and fix the config validation path so enabling Postgres still requires a real DSN value.

Constraints / Preferences / Non-goals  
- Keep the change narrowly scoped to config validation.
- Preserve existing error messages where tests already depend on them.
- Do not turn this into a broader application refactor.
- Prefer a targeted test update or new regression case beside the existing config tests.

Acceptance Criteria / Expected Outcome  
- Empty Postgres DSN input is rejected when Postgres is enabled.
- The fix does not break existing `postgres_enabled_without_dsn_rejected` / `postgres_enabled_with_dsn_allowed` behavior.
- Error classification and message text remain stable unless a test explicitly needs an update.
- There is a regression test covering the empty-value case.

Validation / Verification  
- Run focused tests in `internal/config`, especially `config_test.go`.
- If you change validation logic, verify the specific Postgres config cases rather than running the full suite.

Assumptions / Open Questions  
- The user’s `DATABASE_URL` mention likely maps to the repo’s `postgres.dsn` configuration path, but confirm the exact env mapping before changing behavior.
- It is unclear whether the bug is caused by a default value, env precedence, or validation ordering; inspect the load/validate path before patching.