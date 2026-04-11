# Review Phase 1 Workflow Plan

## Phase Control

- Phase: review-phase-1.
- Phase status: complete.
- Research mode: fan-out plus local repository inspection.
- Completion marker: workflow adequacy challenge reconciled, required review lanes returned or are explicitly superseded, required security lane returned or is explicitly superseded with a narrow rationale, local evidence inspected, validation status recorded, and final template-readiness review report delivered.
- Stop rule: do not implement code changes, create implementation plans, or create task-local design artifacts in this phase.
- Next action: none in this review phase; final report is delivered in chat.

## Local Inspection Surfaces

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `docs/project-structure-and-module-organization.md`
- `README.md`
- `go.mod`
- `Makefile`
- `cmd/service/internal/bootstrap/`
- `internal/app/`
- `internal/domain/`
- `internal/infra/http/`
- `internal/infra/postgres/`
- `internal/infra/telemetry/`
- `internal/config/`
- `api/openapi/service.yaml`
- `internal/api/README.md`
- `env/migrations/`
- `test/`

## Lanes

- `workflow-adequacy`: `challenger-agent`; skill `workflow-plan-adequacy-challenge`; asks whether `workflow-plan.md` and this phase plan are sufficient for this review task and fan-out shape.
- `architecture`: `architecture-agent`; skill `go-design-review`; owns package boundaries, ownership seams, extension paths, and whether new business-code placement is obvious.
- `quality`: `quality-agent`; skill `go-language-simplifier-review`; owns duplicated helpers, scattered policy, naming, junk-drawer helper risk, and same-package source-of-truth opportunities.
- `http-api`: `api-agent`; skill `go-chi-review`; owns chi/router/OpenAPI boundaries, generated API boundary clarity, and endpoint-addition path.
- `data`: `data-agent`; skill `go-db-cache-review`; owns Postgres/sqlc/migration/repository boundaries and persistence-extension clarity.
- `reliability`: `reliability-agent`; skill `go-reliability-review`; owns bootstrap, config, startup, shutdown, readiness, and dependency-admission guidance as template conventions.
- `qa`: `qa-agent`; skill `go-qa-review`; owns test layout, validation commands, integration-test guidance, and future feature-test placement.
- `docs-make-local`: orchestrator local lane; skill `no-skill`; owns README/project-structure docs, Make target discoverability, and whether documented generated-code/test commands match template extension paths.
- `security`: `security-agent`; skill `go-security-review`; decision: required for completion; rationale: local inspection and architecture lane flagged auth/CORS/trust-boundary extension concerns around OpenAPI `security: []`, unused bearer auth, fail-closed CORS preflight, and minimal security headers.

## Fan-In And Synthesis

- Compare subagent claims against direct repository evidence.
- Treat all subagent findings as advisory; final severity and recommendations belong to the orchestrator.
- Reconcile disagreements explicitly in the final report when material.
- Keep findings repository-grounded and avoid generic Go-template checklist advice.
- Fan-in status: architecture, quality, HTTP/API, data, reliability, QA, and security lanes completed; no unresolved lane blocker remains.

## Validation

- Local inspection used `rg`, `find`, `sed`, `nl`, `go list`, and targeted file reads across the requested paths.
- `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/postgres ./internal/infra/telemetry ./internal/app/...`: passed.
- `make check`, `make openapi-check`, `make sqlc-check`, `make test-integration`, and `make migration-validate`: not run; no full baseline, OpenAPI drift, sqlc drift, or live migration correctness claim is made.

## Workflow Plan Adequacy Challenge

- Status: complete.
- Blocking findings: master handoff routing could skip fan-out; security lane was recorded as run but completion allowed skip.
- Resolution: master routing now starts before fan-out/fan-in, the security lane is required for completion, and stale Russian-report wording was replaced with neutral final-report wording.
