# Make the template a trustworthy implementation reference
status: done
Completion: Every accepted audit finding is implemented, current repository
proof passes, and the reviewed candidate is ready for publication.
Blocked stop: Record the exact failed proof and reopen only its owning task.
Global constraints: Preserve production API behavior and unrelated repository
state. Do not hand-edit generated Go. A native managed-worktree implementation
Worker is unavailable in this Codex App task, so the root owns implementation,
review, and acceptance in the dedicated branch.

- [x] T1: Harden and simplify the existing production owners without changing
  supported behavior.
  - Source: `spec.md` Behavior and contract delta; `design/overview.md` Runtime
    decisions.
  - Owner/surface/resources: `internal/infra/postgres`, `internal/config`,
    `cmd/service/internal/bootstrap`, affected tests and configuration docs.
  - Depends on: none
  - Proof: focused PostgreSQL, config, and bootstrap tests pass; regressions
    prove `PGO_ENABLED` acceptance and fail-closed public ingress.

- [x] T2: Add the isolated generated-contract-first article reference service
  and include its generated artifacts in drift validation.
  - Source: `spec.md` reference-slice contract; `design/overview.md` Reference
    feature flow and ownership.
  - Owner/surface/resources: `examples/reference-service`, generated drift
    script, Makefile OpenAPI checks, structure and API guidance docs.
  - Depends on: T1
  - Proof: generation is stable; article use-case and full HTTP-path tests pass;
    OpenAPI lint, schema validation, generated package tests, and runtime
    validation cases pass.

- [x] T3: Review and prove the complete candidate for the authorized GitHub
  workflow.
  - Source: `spec.md` Success criteria and proof expectations.
  - Owner/surface/resources: full branch diff and repository-native checks.
  - Depends on: T1, T2
  - Proof: `git diff --check`, focused tests, race-relevant tests,
    `make openapi-check`, `make lint`, and `make check` pass on the final tree.
