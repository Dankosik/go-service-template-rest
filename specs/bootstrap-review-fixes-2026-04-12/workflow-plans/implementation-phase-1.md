# Implementation Phase 1

Phase: `implementation-phase-1`
Status: `complete`

Purpose:
- Implement the approved bootstrap review fixes from `../spec.md`, `../design/`, `../plan.md`, and `../tasks.md`.

Allowed writes:
- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`
- `cmd/service/internal/bootstrap/network_policy_parsing.go`
- `cmd/service/internal/bootstrap/startup_bootstrap.go`
- `cmd/service/internal/bootstrap/startup_common.go` only if needed for structured policy log fields
- `cmd/service/internal/bootstrap/startup_common_additional_test.go` only if policy log fields are wired
- `docs/configuration-source-policy.md`
- `docs/repo-architecture.md` only if a cross-link or egress policy note is needed
- `docs/project-structure-and-module-organization.md` only if a cross-link or egress policy note is needed
- `env/.env.example`
- Existing workflow/progress artifacts in `specs/bootstrap-review-fixes-2026-04-12/`

Disallowed writes:
- Redis/Mongo adapter packages.
- `internal/config/**` unless implementation is reopened to technical design first.
- `internal/infra/telemetry/**` unless implementation is reopened to technical design first.
- API contracts, generated code, migrations, or unrelated docs.
- New workflow/process artifacts.

Execution order:
1. Complete T001-T004 for dependency startup cleanup.
2. Complete T005-T006 for network policy documentation.
3. Complete T007 for network policy label ownership.
4. Complete T008 verification.

Stop / reopen rule:
- If `NETWORK_*` needs to become typed config, stop and reopen technical design.
- If metric API changes are needed, stop and reopen technical design.
- If Redis/Mongo adapter semantics appear necessary, stop and reopen specification or technical design.

Exit criteria:
- T001-T008 checked according to real completed work.
- Verification commands pass or failures are documented with blockers.
- Master `workflow-plan.md` and this file are updated with implementation status.

Implementation status:
- T001-T008 complete:
  - `go test ./cmd/service/internal/bootstrap`: passed, 90 tests.
  - `go test ./internal/config ./cmd/service/internal/bootstrap`: passed, 189 tests.
  - `go vet ./cmd/service/internal/bootstrap`: passed, no issues.
  - `gofmt -l` on touched Go files: passed, no output.
  - `rg networkPolicyErrorLabels cmd/service/internal/bootstrap`: production use remains in `startup_bootstrap.go` plus parser/tests.

Blocker:
- None for this implementation phase.
