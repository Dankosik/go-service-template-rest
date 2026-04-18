Task: Railway auto migrations
Execution shape: lightweight local
Execution rationale: bounded single-domain delivery change across Railway deployment policy, image contents, and one migration runner; no API/domain behavior change; user asked for end-to-end implementation in one pass.
Current phase: done
Phase status: completed
Session boundary: upfront lightweight-local waiver recorded. Pre-code phases, implementation, and validation were completed locally in this session.
Research mode: local repository analysis plus `exa` retrieval of official Railway docs; no subagent fan-out in this session.
Workflow-plan adequacy challenge: locally reconciled against `workflow-plan-adequacy-challenge` criteria because this session stayed local and did not open challenger lanes.
Spec clarification gate: locally reconciled against `spec-clarification-challenge` criteria; no unresolved approval blockers remain for this bounded change.
Implementation readiness: PASS
Validation status: completed with fresh evidence

Artifacts:
- `spec.md`: approved and closed with validation outcome
- `research/railway-predeploy-and-github-autodeploy.md`: approved
- `design/overview.md`: approved
- `design/component-map.md`: approved
- `design/sequence.md`: approved
- `design/ownership-map.md`: approved
- `design/data-model.md`: not expected; rationale: schema shape does not change, only migration execution ownership does
- `tasks.md`: completed
- `rollout.md`: approved
- `test-plan.md`: not expected; rationale: proof obligations fit in `tasks.md`

Blockers: none

Accepted risks:
- Railway `preDeployCommand` failures block promotion and are not retried by the platform; the safe operator action is to fix the cause and redeploy.
- Railway keeps a 45-second overlap window, so same-deploy schema changes must remain mixed-version compatible. Destructive or contract-only migrations still require staged rollout discipline.

Validation evidence:
- `go test ./cmd/migrate ./internal/infra/postgres`
- `go test ./...`
- `go test -tags=integration ./test -run '^TestPostgresMigrateUpAppliesAndReplaysMigrations$' -count=1`
- live `go run ./cmd/migrate` against temporary Postgres
- `make migration-validate`
- `make guardrails-check`
- `docker build -f build/docker/Dockerfile -t go-service-template-rest:migrate-check .`

Next action: none; task is complete.
Next session: none.
Next session context bundle: default resume order is sufficient because no follow-up session is required.
