# Internal Infra Review Workflow Plan

## Task

Review `/Users/daniil/Projects/Opensource/go-service-template-rest/internal/infra` for Go idiomaticity, maintainability, readability, and boundary/design drift. The user explicitly requested subagents.

## Execution Shape

- Shape: lightweight local, agent-backed review.
- Current phase: implementation.
- Review phase-collapse waiver: accepted for the original review session because that task was read-only review of an existing directory, not implementation or behavior design.
- Research mode: fan-out with read-only subagent lanes plus orchestrator synthesis.

## Implementation Follow-Up

- Shape: lightweight local implementation from accepted review findings.
- Implementation-readiness status: WAIVED for this narrow follow-up because the findings are already concrete, local, and do not require new product, architecture, or API decisions.
- Scope: apply all accepted review findings in `internal/infra`, `env/migrations`, and tests; no broad refactor.
- Proof path: targeted package tests after each block, then scoped `go test`, `go test -race`, `go vet`, and migration/query drift checks as applicable.
- Status: completed.

## Artifact Status

- `workflow-plan.md`: completed for this review session.
- `workflow-plans/review.md`: completed for this review session.
- `spec.md`: not expected; review-only request, no new product or behavior decision.
- `design/`: not expected; no implementation design requested.
- `plan.md`: not expected; no implementation requested.
- `tasks.md`: not expected; no implementation requested.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.

## Lanes

- Workflow adequacy challenge: `challenger-agent`, skill `workflow-plan-adequacy-challenge`, checks only the sufficiency of these review-control artifacts.
- Idiomatic Go review: `quality-agent`, skill `go-idiomatic-review`, owns language-level correctness and Go-standard-library idiom issues in `internal/infra`.
- Readability and simplification review: `quality-agent`, skill `go-language-simplifier-review`, owns local reasoning load, helper economics, naming, and control-flow clarity in `internal/infra`.
- Boundary and maintainability design review: `architecture-agent`, skill `go-design-review`, owns package boundary, dependency direction, source-of-truth, and accidental-complexity findings in `internal/infra`.
- Chi transport review: `api-agent`, skill `go-chi-review`, owns chi middleware order, route fallback policy, generated-route integration, and route-label semantics in `internal/infra/http`.
- DB/cache review: `data-agent`, skill `go-db-cache-review`, owns SQL access discipline, transaction boundaries, context/resource safety, and repository data-access maintainability in `internal/infra/postgres`.

## Adequacy Challenge

- Status: passed.
- Result: no blocking workflow-control gaps found.
- Evidence boundary: the read-only challenger inspected only this master workflow plan, `workflow-plans/review.md`, and the `workflow-plan-adequacy-challenge` skill; it did not perform code review.

## Stop Rule

Stop after the orchestrator has compared subagent outputs with local code evidence, produced prioritized findings with file/line references, and recorded any validation commands run. Do not edit code as part of this review.

## Review Result

- Status: completed.
- Code edits: none.
- Findings were reconciled from idiomatic Go, simplification, design, chi, and DB/cache lanes plus local review.
- Fresh validation evidence:
  - `go test ./internal/infra/...` passed.
  - `go test -race ./internal/infra/...` passed.
  - `go vet ./internal/infra/...` passed.
  - Targeted tests for telemetry, config OTLP env, Postgres validation, and HTTP policy/route labels passed.

## Implementation Result

- Status: completed.
- Accepted findings fixed in HTTP middleware/router/problem handling, Postgres repository/pool error handling, telemetry OTLP endpoint parsing, and ping history migration coverage.
- Fresh validation evidence:
  - `go test ./internal/infra/... -count=1` passed.
  - `go test -race ./internal/infra/... -count=1` passed.
  - `go vet ./internal/infra/...` passed.
  - `go test -tags=integration ./test -run 'PingHistory' -count=1 -v` passed.
  - `go test ./... -count=1` passed.
  - `go vet ./...` passed.

## Blockers And Risks

- Blockers: none known.
- Accepted risk: generated `internal/infra/postgres/sqlcgen` code may be reviewed only for generated-surface concerns; primary source-of-truth for SQL access remains the query/migration/generator inputs.

## Next Session

No follow-up phase is planned unless the user asks for fixes. If fixes are requested, start a new implementation-framed session from the accepted findings.
