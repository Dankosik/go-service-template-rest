# Component Map

## `api/openapi/service.yaml`

Expected changes:

- Add explicit operation-level security decision metadata for current operations.
- Keep current public/system/sample endpoints public only by explicit decision.
- Add real security schemes only in a future auth feature, not in this follow-up.

Stable:

- OpenAPI remains the REST source of truth.
- Generated `internal/api` remains derived output.

## `internal/api/`

Expected changes:

- Regenerate only if OpenAPI vendor extensions or contract details change generated output.
- Do not hand-edit `internal/api/openapi.gen.go`.

Stable:

- `internal/api/README.md` remains the generated-artifact guidance surface.

## `internal/infra/http/`

Expected changes:

- Pass a generated chi `ErrorHandlerFunc` that uses the same sanitized malformed-request Problem path as strict handler request errors.
- Add or adjust tests for generated-error option behavior.
- Redact raw panic values in `Recover`.
- Keep fail-closed CORS behavior.
- Keep `/metrics` as the only documented manual root-router exception unless a future design changes it.

Stable:

- HTTP mapping stays outside `internal/app`.
- Normal API routes continue to flow through generated strict handlers.

## `cmd/service/internal/bootstrap/`

Expected changes:

- Distinguish missing public-ingress declaration from declared false.
- Enforce explicit public-ingress declaration for non-local wildcard binds.
- Keep concrete dependency admission and cleanup in bootstrap.
- Add package-local constants/helper cleanup for dependency startup labels and unused lifecycle/once plumbing.

Stable:

- `cmd/service/main.go` stays thin.
- Do not create a generic dependency manager.

## `internal/config/`

Expected changes:

- Replace exact `http.shutdown_timeout == 30s` validation or document it as intentionally locked. Preferred: relationship/range validation.
- Add aggregate readiness-budget validation for enabled sequential readiness probes.
- Clarify YAML secret placeholder behavior with code/tests/docs aligned.
- Add config key drift protection using tests or package-local constants.
- Clarify that CLI flags are loader controls today, not arbitrary runtime config overrides.

Stable:

- Runtime config source of truth stays in `internal/config`.
- Keep explicit snapshot and validation logic.

## `internal/app/health/`

Expected changes:

- Keep `Probe` as the readiness probe interface owner.
- Add docs/tests only if aggregate readiness behavior changes.

Stable:

- Liveness stays process-only in HTTP handlers.
- Readiness is where dependency/drain/startup-admission checks belong.

## `internal/infra/telemetry/`

Expected changes:

- Redact malformed OTLP header errors so secret-like config values do not appear in returned errors.

Stable:

- Telemetry setup stays in telemetry adapter package.

## `internal/infra/postgres/`

Expected changes:

- None required by default. Do not change schema/sqlc code unless implementation discovers a direct need.

Stable:

- `ping_history` remains sample-only unless a separate data task decides otherwise.
- Do not create generic repository or transaction helpers.

## `internal/domain/`

Expected changes:

- Add `doc.go` or adjust docs. Preferred: add `doc.go` explaining the package is intentionally reserved for shared stable contracts.

Stable:

- Do not add domain types that are not used.

## Docs And Build Surfaces

Expected changes:

- `README.md`: remove stray clone-readiness noise.
- `docs/project-structure-and-module-organization.md`: add endpoint security decision, browser endpoint checklist, runtime dependency checklist, persistence port sketch, liveness/readiness rule, and validation updates.
- `docs/repo-architecture.md`: update extension seams and trust-boundary notes where needed.
- `docs/configuration-source-policy.md`: clarify flags, shutdown/readiness budgets, secret placeholder policy.
- `docs/build-test-and-development-commands.md`: add migration rehearsal and generated-helper drift guidance in feature workflows.
- `test/README.md`: update only if test placement guidance needs a link or consistency repair.
- `Makefile`: change only if help output still hides needed feature validation commands.
