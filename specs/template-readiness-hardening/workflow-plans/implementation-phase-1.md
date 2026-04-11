# Template Readiness Hardening Implementation Phase 1

## Phase Control

- Phase: implementation-phase-1.
- Status: complete.
- Entry condition: implementation readiness is PASS in `workflow-plan.md` and `plan.md`.
- Work source: `tasks.md`.
- Stop rule: implement only the approved hardening tasks; do not create new workflow/process artifacts.

## Scope

- Rename or otherwise align the OpenAPI security-decision guard test with `make openapi-runtime-contract-check`.
- Document the protected endpoint convention in `internal/api/README.md`.
- Bound the `ping_history` sample list limit and add targeted tests.
- Move Redis mode/readiness policy into a narrow config-owned API and consume it from bootstrap.
- Apply the supporting README, CONTRIBUTING, project-structure, architecture, command-doc, and test README placement/discoverability updates marked planned in `research/coverage-audit.md`.

## Out Of Scope

- No generic auth framework.
- No new OpenAPI protected endpoint.
- No Redis adapter implementation.
- No startup failure recorder extraction.
- No README-wide onboarding rewrite beyond the targeted placement and migration-validation pointers.
- No log-label taxonomy change or degraded dependency logging helper extraction.

## Completion Marker

- All tasks in `tasks.md` complete.
- Required targeted tests pass or blocked commands are recorded with reason.
- `workflow-plan.md` and `tasks.md` are updated by the implementation session to reflect actual proof.

## Implementation Closeout

- Completion marker met: yes.
- Tasks complete: T001-T018.
- Required proof passed:
  - `make openapi-runtime-contract-check`
  - `go test ./internal/infra/http -count=1`
  - `go test ./internal/infra/postgres -count=1`
  - `go test ./internal/config ./cmd/service/internal/bootstrap -count=1`
- Recommended proof also passed:
  - `make openapi-check`
  - `make check`
- Documentation review: completed against `research/coverage-audit.md`; each planned documentation point is covered by the updated README, CONTRIBUTING, command docs, architecture baseline, project-structure guide, API README, or test README.
