# Keep only service-owned tooling
status: ready

## Scope and non-goals

The template keeps commands that initialize, build, test, generate, secure, and
run the Go service. It removes repository-local platforms that validate or
administer the template's own agent methodology, mirror the host toolchain in
Docker, or mutate GitHub repository policy.

In scope:

- one native template initialization path;
- one isolated contract check for that mutating initialization path;
- native Make targets for service development and validation;
- Docker only for the runtime image, Compose dependencies, integration tests,
  migration rehearsal, container security, and an explicit external HTTP load
  benchmark;
- removal of workflow/skill eval infrastructure, self-referential instruction
  and guardrail checkers, Codex worktree transfer checks, GitHub branch
  protection automation, and Docker command mirrors;
- removal of eval manifests and fixtures that have no remaining runner;
- simplification of aggregate local and GitHub CI so coverage owns the ordinary
  test run and lint owns `govet`;
- documentation aligned with the surviving command surface.

The service runtime, public API, generated OpenAPI/sqlc authority, migrations,
security policy, release publishing, and the Codex runtime instructions and
skills themselves do not change.

## Behavior and contract delta

- `make template-init [MODULE=<host/org/repo>] CODEOWNER=@owner` is the only
  onboarding command for a repository created from the template. It derives
  the module from `origin` when `MODULE` is omitted, requires `CODEOWNER` when
  the target module differs from the template source repository, replaces the
  template owner in every CODEOWNERS rule, creates `.env` when absent, updates
  Go/protobuf import paths and module-qualified `.golangci.yml` depguard rules,
  and runs `go mod tidy`.
- `make check` remains the quick native formatting, lint, and test loop.
- `make ci-local` is a deterministic native aggregate: module/formatting,
  lint, coverage, race, generated-source, OpenAPI, Go security, and secret
  checks. It does not conditionally skip Docker-dependent gates.
- `make check-full` requires Docker and extends `make ci-local` with integration
  tests, migration rehearsal, runtime migrator smoke, and the container image
  security scan. Its Docker-backed checks reuse one freshly built runtime image.
- `make pr-check BASE_REF=<ref>` adds the OpenAPI breaking-change comparison to
  `make check-full`.
- `make migration-validate` uses an explicit `MIGRATION_DSN` for SQL up/down/up
  rehearsal when supplied. Otherwise it requires Docker, starts the
  repository's Compose Postgres, rehearses SQL up/down/up, builds or consumes
  the current runtime image, executes that image's `/migrate` entrypoint
  against the same database, and always tears the dependency down. It never
  reports a missing DSN and Docker daemon as a successful skip.
- Runtime Docker build/run, Compose, and one actual container-security target
  remain. `docker-*` mirrors of Go, lint, test, generation, security, and
  aggregate CI commands are removed.
- The retained Compose Postgres image is digest-pinned in
  `env/docker-compose.yml`; the retained local Trivy image is digest-pinned at
  the Makefile container-security target. Unused Go, Node, and migrate tooling
  image catalog entries are removed with the emulator.
- Existing `make bench*` commands and benchmark artifacts remain. Database
  benchmarks require host Go plus Docker/Testcontainers; the accepted removal
  of zero-host-toolchain support drops their Docker-Go fallback. The explicit
  HTTP load benchmark keeps digest-pinned k6 execution under
  `scripts/dev/benchmark.sh`, not the deleted general Docker emulator.
- GitHub Actions retain real module, formatting, lint, generated drift,
  OpenAPI, coverage, race, integration, migration, Go security, secret, image
  build/scan, and release gates. Both merge CI and release preflight remove
  standalone ordinary `make test` and `make vet` execution: `test-report` owns
  the ordinary all-package test run, and lint's enabled `govet` owns static vet
  analysis. Methodology/self-check steps are also removed.
- GitHub branch rules are configured through GitHub Rulesets or organization
  policy. `.github/workflows/ci.yml` is the only exact registry of job IDs;
  setup documentation says to require every blocking merge job and names only
  intentional informational exceptions. The template does not ship a second
  API client or duplicate exact context list.

## Invariants and edge cases

- Generated files are still checked against their canonical OpenAPI and sqlc
  sources.
- Lint, test-report coverage, race, integration, migration, Go vulnerability,
  gosec, secret, container image, and release checks remain fail-closed.
- Template initialization must not silently rewrite the template source
  repository to its own upstream URL. For a derived repository, a missing or
  malformed module path or CODEOWNER fails before `.env`, `go.mod`, source, or
  CODEOWNERS mutation. Module validity is established by `go mod edit` against
  a temporary copy before the real module is touched. `CODEOWNER` accepts
  exactly one whitespace-free GitHub CODEOWNERS token in standard `@username`
  or `@org/team-name` form: user/org components start and end alphanumeric and
  contain only alphanumeric or `-`; the optional team component starts and ends
  alphanumeric and contains only alphanumeric, `-`, or `_`. Account existence,
  visibility, and write permission remain external GitHub prerequisites. The
  template source repository may omit `CODEOWNER` and keeps its existing module
  and owners. Any existing `.env` is preserved.
- One narrow temporary-repository check proves successful derived-repository
  initialization and proves that missing or malformed module/CODEOWNER input
  leaves `.env`, `go.mod`, source imports, `.golangci.yml`, and CODEOWNERS
  byte-identical. After successful module renaming, the same check runs the real
  depguard configuration against one forbidden app-to-infra import and observes
  rejection.
- Migration rehearsal must clean up Compose resources on success or failure.
  Each invocation owns a unique Compose project and ephemeral host port, so its
  teardown cannot remove or collide with a developer's existing local Postgres
  project.
- The Docker-backed migration path used by `check-full` and release preflight
  must prove the shipped `/migrate` executable, not only the host migration
  tool.
- Existing benchmark tooling and the unrelated in-progress benchmark changes
  in the checkout are preserved.
- No removed target, script, eval asset, or documentation instruction remains
  referenced by the active Makefile, CI workflows, README, workflow docs, or
  runtime skill guidance.

## Decisions, constraints, and authorities

- The Makefile remains the local command authority; GitHub workflows call those
  commands rather than maintaining Docker equivalents.
- `.golangci.yml` remains the import-boundary authority; actual build, test,
  generation, and security commands prove their own behavior. Template
  initialization must rewrite every occurrence of the old module prefix in
  module-qualified depguard rules before lint can be trusted in the derived
  repository.
- `api/openapi/service.yaml`, sqlc query/config files, migrations, the runtime
  Dockerfile, and GitHub Actions remain their existing authorities.
- Compose owns its Postgres image reference, and the Makefile owns the local
  Trivy image reference; both remain immutable digest references.
- GitHub repository policy is external administrative state and is not owned by
  a cloned service repository.
- `.github/CODEOWNERS` is the source for the current template owner token.
  Initialization replaces that token in every non-comment ownership rule; it
  does not infer an owner from a repository namespace because that can silently
  assign governance to the wrong team.
- Current repository inspection found no real workflow-eval runner or judge,
  only placeholder adapter paths and a fake harness. Retaining dormant eval
  assets would therefore create an unowned compatibility promise.
- Technical design is scoped down because no runtime topology, Go package
  ownership, API, data, or new mechanism is introduced; each surviving command
  keeps its current Makefile, Compose, generated-source, or GitHub workflow
  owner.
- Dedicated test design is scoped down because initialization has a finite
  success/failure contract captured by one isolated check, while the replaced
  migration lifecycle needs one terminal forced-failure cleanup oracle rather
  than a permanent harness. Existing service gates plus a negative reference
  scan directly falsify the rest of the target state.

## Success criteria and proof expectations

- Only the minimal initialization, its contract check, benchmark, and
  generated-drift scripts remain under `scripts/`.
- `make help` exposes only surviving commands.
- `make check`, `make ci-local`, generated drift checks, OpenAPI/sqlc checks,
  race, coverage, Go security, and secret scanning pass natively.
- `make ci-local` and merge CI execute the initialization contract check.
- With Docker available, `make check-full` also passes integration, migration,
  runtime image build, and image vulnerability scan without skipped gates.
- Terminal Docker proof forces migration failure after the unique Compose
  project starts and observes no remaining container, network, or volume for
  that project.
- GitHub workflows parse and contain no removed commands or redundant ordinary
  test/vet job.
- A repository-wide reference scan finds no active references to the removed
  scripts, targets, eval platform, or Docker mirrors.
- `git diff --check` passes, and the pre-existing benchmark delta remains
  present with its commands and artifact contract intact except for the
  explicitly removed Docker-Go fallback.

## Risks, assumptions, and reopen conditions

- This intentionally drops the "host Go/Node not required" promise. Reopen only
  if a concrete consumer requires zero-host-toolchain development and accepts
  ownership of one small generic wrapper plus its ongoing proof cost.
- GitHub Rulesets cannot be verified from this local repository. Repository
  adopters own that one-time external setup. For the template source repository,
  removing a CI job creates a publication prerequisite: inspect the effective
  Ruleset/classic protection, replace retired required contexts with surviving
  blocking job IDs, and do not publish the workflow change without that
  external proof. This local task does not authorize GitHub mutation and cannot
  claim publication readiness without it.
- Reopen delivery design if a retained gate cannot run without the deleted
  emulator and cannot be expressed through its native tool or the existing
  runtime/Compose Docker boundary.
