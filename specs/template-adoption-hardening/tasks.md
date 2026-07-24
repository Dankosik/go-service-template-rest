# Make derived services safe and small by default
status: implementation candidate
Completion: Every adoption finding is implemented, the committed candidate
passes mapped proof, and the GitHub PR is merged only after required checks pass.
Blocked stop: Preserve the last accepted candidate, record the exact failed
gate or missing external authority, and do not merge.
Global constraints: Start from `origin/main` commit `d737781`; preserve the
production health API and generated-source authorities; add no runtime
dependency; never expose metrics on the application listener. The current Codex
session has no native managed-worktree Worker, so root owns sequential
implementation and acceptance on `codex/template-adoption-hardening`.

- [x] T1: Derived repositories receive complete identity and an explicit
  database profile while every repository retains the complete agent workflow.
  - Source: `spec.md` Derived-service profiles; `design/overview.md`.
  - Owner/surface/resources: `scripts/init-module.sh`,
    `scripts/ci/template-init-check.sh`, profile transformations, README,
    Makefile, environment examples, exact temporary directories.
  - Depends on: none
  - Proof: TD-1 and TD-6; `make template-init-check`.

- [x] T2: Application and Prometheus diagnostics listeners are separate,
  explicit public application ingress works, and lifecycle remains bounded with
  the smaller wait path.
  - Source: `spec.md` Runtime behavior; `design/overview.md` Runtime and
    lifecycle; TD-2, TD-3, TD-5.
  - Owner/surface/resources: `internal/infra/http`,
    `internal/infra/telemetry`, `internal/config`,
    `cmd/service/internal/bootstrap`, loopback ephemeral ports, affected docs.
  - Depends on: T1
  - Proof: focused config/HTTP/bootstrap tests, repeated lifecycle test,
    `go test -race ./cmd/service/internal/bootstrap ./internal/infra/http`.

- [x] T3: Unknown configuration always fails and configuration documentation
  has one accurate extension path.
  - Source: `spec.md` strict configuration; TD-4.
  - Owner/surface/resources: `internal/config`, service/migrator config callers,
    affected docs and tests.
  - Depends on: T2
  - Proof: focused config and bootstrap argument tests.

- [x] T4: Migration code is isolated from the service graph, the first-feature
  guide is maintained against real owners, and stale optional-capability
  promises are removed.
  - Source: `spec.md` optional database and feature workflow; TD-6, TD-8.
  - Owner/surface/resources: PostgreSQL/migration packages, `cmd/migrate`,
    generated checks, examples and docs.
  - Depends on: T3
  - Proof: focused migration tests, service dependency assertion, OpenAPI and
    reference-service tests.

- [x] T5: GHCR publication is opt-in and tool/version ownership no longer
  contaminates or obscures the runtime upgrade surface.
  - Source: `spec.md` delivery and maintenance behavior; TD-7.
  - Owner/surface/resources: workflows, Makefile, Go module/tool manifests,
    Dependabot and upgrade docs; no registry mutation.
  - Depends on: T4
  - Proof: module/version consistency, workflow checks, lint/security gates.

- [ ] T6: Review, validate, commit, publish, wait for required GitHub checks,
  and merge the exact accepted candidate.
  - Source: `spec.md` Success criteria; TD-9; user publication authorization.
  - Owner/surface/resources: complete branch diff, local caches/Docker if
    available, GitHub branch/PR/checks/main.
  - Depends on: T5
  - Proof: repository-native local gates pass on the commit; PR checks are
    successful; merge commit is present on remote `main`.
