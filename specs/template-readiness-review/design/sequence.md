# Sequence

## Implementation Sequence

1. Update docs first.
   Add the first-production-feature checklist and clarify non-goals. This gives later code/test guardrails an explicit policy to enforce.

2. Clarify sample and stub semantics.
   Strengthen `ping_history` replacement guidance and Redis/Mongo guard-only language. Avoid schema, query, or generated-code churn.

3. Add narrow guardrails.
   Add the app/domain import boundary guardrail and the HTTP route-tree guard. These should enforce existing architecture policy rather than introduce new behavior. While route policy is in scope, canonicalize `Allow` header emission.

4. Fix the concrete startup log drift.
   Update `recordDependencyProbeRejection` to include `err` in the `startup_blocked` log args and add targeted bootstrap coverage.

5. Run scoped validation, then broad validation.
   Start with package/script checks for touched surfaces, then run `go test ./...`. Run OpenAPI or SQLC checks only if their source-of-truth surfaces change unexpectedly.

## Failure And Reopen Points

- If the implementation needs a real auth decision, stop and reopen specification/security design.
- If the implementation needs to rename `ping_history` schema/query/generated surfaces, stop and ask for maintainer approval.
- If the route-tree guard cannot reliably distinguish manual root routes from generated mounts, keep the narrower helper-based guard and document the limitation instead of writing a brittle test.
- If the import boundary guardrail cannot be expressed portably in the existing shell guardrail script, leave it as documentation or add a Go test only after confirming the owning package surface.
