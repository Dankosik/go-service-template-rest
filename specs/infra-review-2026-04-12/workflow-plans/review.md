# Review Phase Plan: internal/infra

Phase: review
Status: complete

## Local Orchestration

Order:
1. Run workflow-plan adequacy challenge against this workflow-control pair.
2. Repair workflow-control artifacts only if the challenger reports a blocking routing gap.
3. Run read-only review fan-out in parallel.
4. Fan in results, compare overlapping claims, discard style-only nits, and produce final findings.

Parallelizable work:
- Idiomatic Go lane can run independently.
- Simplification/readability lane can run independently.
- Design/maintainability lane can run independently.
- Chi transport lane can run independently.
- Postgres data-access lane can run independently.

## Lanes

- `workflow-adequacy`: role `challenger-agent`; owned question: are `workflow-plan.md` and this review phase plan sufficient for this agent-backed review task; skill `workflow-plan-adequacy-challenge`.
- `idiomatic-go`: role `quality-agent`; owned question: Go semantics, stdlib contracts, error/context/nil/resource idioms in `internal/infra`; skill `go-idiomatic-review`.
- `readability-simplification`: role `quality-agent`; owned question: maintainable local control flow, naming, helper economics, source-of-truth extraction, and readable test shape in `internal/infra`; skill `go-language-simplifier-review`.
- `design-maintainability`: role `architecture-agent`; owned question: infra boundary ownership, dependency direction, generated-source boundaries, source-of-truth drift, and accidental complexity; skill `go-design-review`.
- `chi-transport`: role `api-agent`; owned question: chi router ownership, middleware order/scope, fallback policy, route labels, and generated/manual route integration in `internal/infra/http`; skill `go-chi-review`.
- `postgres-data-access`: role `data-agent`; owned question: pgx/sqlc resource safety, transaction boundaries, context propagation, and query discipline in `internal/infra/postgres`; skill `go-db-cache-review`.

## Completion Marker

The review phase is complete when:
- workflow adequacy challenge has no unreconciled blocking finding,
- required subagent lanes have returned or an explicit user redirect cancels them,
- orchestrator has reconciled duplicate or conflicting claims,
- final response contains review findings first, then open questions/residual risks and validation notes.

## Stop Rule

Do not implement fixes in this session. Do not create specification, design, implementation plan, or task-ledger artifacts unless the user asks for a follow-up fix task.

## Local Blockers

- None currently.

## Adequacy Challenge Status

Status: complete
Resolution: no blocking adequacy gaps in initial pass or second pass after adding chi and Postgres data-access lanes; proceed with planned read-only review fan-out.

## Completion

Completion marker: satisfied.
Session boundary reached: yes.
Next action: final response only; no code edits in this session.
Validation evidence: `go vet ./internal/infra/...` passed; `go test -count=1 ./internal/infra/...` passed with 110 tests across 4 packages.
