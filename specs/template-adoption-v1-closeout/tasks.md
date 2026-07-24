# Adoption v1 closeout tasks

Status: implemented

## Global constraints

- Agent workflow files and behavior are immutable for this outcome.
- Preserve unrelated user files and work only in the isolated implementation
  worktree.
- Use existing dependencies, generators, and repository gates.
- Do not push, deploy, or mutate external systems.

## T1 — Mandatory workflow and removable reference example — complete

- Depends on: none
- Outcome: initialization has no workflow profile and preserves all workflow
  paths byte-for-byte; service OpenAPI gates remain complete with or without
  the reference example.
- Owner/surface/resources: `scripts/init-module.sh`,
  `scripts/ci/template-init-check.sh`, `scripts/ci/generated-drift-check.sh`,
  `Makefile`, initialization/user docs; temporary local checkouts only.
- Proof: TD-1, TD-2; `make template-init-check`.
- Stop condition: no `AGENT_WORKFLOW` runtime/profile reference remains and the
  temporary deletion proof passes.

## T2 — Bounded, diagnosable migrations — complete

- Depends on: T1
- Outcome: typed overall/statement/lock budgets reach their owning layers,
  cancellation cannot report success, dirty version failures carry safe manual
  recovery guidance, and automatic force remains absent.
- Owner/surface/resources: `internal/config`, `cmd/migrate`,
  `internal/infra/postgresmigrate`, migration integration tests, Railway
  deployment docs; one disposable local PostgreSQL container for integration.
- Proof: TD-3 through TD-6; focused unit tests and
  `REQUIRE_DOCKER=1 go test -tags=integration ./test -run
  'TestPostgresMigrate'`.
- Stop condition: invalid inputs fail before connection, timeout/cancellation
  and dirty diagnostics are executable, and cleanup assertions remain green.

## T3 — CI-applied PostgreSQL POST vertical slice — complete

- Depends on: T2
- Outcome: a temporary initialized service proves the complete OpenAPI ->
  transport -> domain -> hand-written repository -> sqlc -> PostgreSQL flow,
  including negative request paths and transaction finality.
- Owner/surface/resources: `scripts/ci/fixtures/postgres-post-feature.patch`,
  `scripts/ci/template-init-check.sh`, `Makefile`, `.github/workflows/ci.yml`;
  generated files and a disposable PostgreSQL container exist only under the
  temporary checkout.
- Proof: TD-7, TD-8;
  `REQUIRE_DOCKER=1 TEMPLATE_POSTGRES_PROOF=1 make template-init-check`.
- Stop condition: dedicated CI invocation is required, the fixture leaves no
  base demo domain, and the real database assertions pass.

## T4 — Closeout — complete

- Depends on: T3
- Outcome: canonical/generated drift, focused tests, broad repository gates
  justified by the cross-cutting change, and final diff review support the
  completion claim.
- Owner/surface/resources: current worktree; no external writes.
- Proof: `make check`, required Docker integration from T2/T3, targeted
  OpenAPI/template checks, `git diff --check`, and final status/diff review.
- Stop condition: all mapped claims have current evidence or a named,
  honestly reported environment gap.
