# Make the template's examples safe and easy to extend
status: done
Completion: Every accepted follow-up finding is implemented, current local
proof passes, and the reviewed candidate is ready for the authorized GitHub
workflow.
Blocked stop: Record the exact failed proof or external merge condition and
reopen only its owning task.
Global constraints: Preserve public behavior and generated authorities, add no
dependency or speculative abstraction, and leave unrelated repository state
untouched. A native managed-worktree Worker is unavailable in this Codex App
task, so the root owns implementation, review, and acceptance in the dedicated
branch.

- [x] T1: Make migration discovery, full-chain rehearsal, and runner cleanup
  fail honestly.
  - Source: `spec.md` Behavior and ownership; `spec.md` Proof obligations.
  - Owner/surface/resources: `cmd/migrate`, `internal/infra/postgres`, focused
    unit tests, and the Docker-gated PostgreSQL integration fixture.
  - Depends on: none
  - Proof: focused tests distinguish absent and invalid paths, assert
    up/down/up rehearsal and joined close errors, and the multi-migration
    integration case reaches an earlier down migration.

- [x] T2: Close the mutable HTTP client seam and remove residual generic
  bootstrap/config behavior.
  - Source: `spec.md` Behavior and ownership;
    `../template-quality-hardening/design/overview.md` PostgreSQL startup.
  - Owner/surface/resources: `internal/infra/httpclient`, `internal/config`,
    `cmd/service/internal/bootstrap`, and their focused tests.
  - Depends on: none
  - Proof: package tests exercise `Client.Do`, direct whitespace-path rejection,
    and concrete PostgreSQL startup/rejection behavior.

- [x] T3: Split mixed-responsibility implementation and test files without
  changing package boundaries or behavior.
  - Source: `spec.md` Scope and non-goals; `spec.md` Behavior and ownership.
  - Owner/surface/resources: existing files in `internal/config`,
    `cmd/service/internal/bootstrap`, and `internal/infra/postgres`; no
    generated files.
  - Depends on: T1, T2
  - Proof: affected package tests pass, formatting is stable, and the final
    diff contains only same-package ownership moves plus accepted fixes.

- [x] T4: Review and prove the complete candidate for the authorized GitHub
  workflow.
  - Source: `spec.md` Proof obligations; repository `AGENTS.md` Validation
    Matrix.
  - Owner/surface/resources: full branch diff and repository-native checks.
  - Depends on: T1, T2, T3
  - Proof: focused tests, Docker-gated migration integration, and
    `BASE_REF=origin/main make pr-check` pass on the final candidate.
