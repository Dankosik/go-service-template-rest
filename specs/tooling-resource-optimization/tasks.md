# Optimize build and test resource use
status: done
Completion: Native cache ownership, split secret-scan gates, bounded local
profiles, deterministic slow tests, fail-closed docs-only CI, one runtime-image
build, PostgreSQL image authority, persistent Trivy data, and duplicate
module/vet/OpenAPI/Docker work removal are current without weakened proof.
Exact-version prebuilt CI tools, runtime-only test caches, filtered source-bind
Docker builds, and duplicate Trivy/watch/template work are included.
Evidence: actionlint and static trust/cache assertions passed, both database
profiles built as nonroot, a test-only file retained the cached runtime compile
layer, and the final `make check-full-gentle` passed in 105.35s with 3.01 GB
peak RSS and zero swaps.
The final follow-up also passed `make -j8 check-full-gentle`: heavy prerequisites
remained serial, set-mode coverage held at 82.40%, both generated database
profiles built, Distroless supplied named timezone data, and Trivy reported no
HIGH/CRITICAL findings. Removing embedded tzdata reduced the service binary by
393,216 bytes with otherwise identical build flags.
The final local-loop correction kept module-download and generator integrity in
`check-full` while reducing the warm `ci-local-gentle` path to 21.71s. The
complete Docker-backed gate passed in 138.25s with 1.94 GB peak RSS and zero
swaps. Minimal and PostgreSQL generator profiles passed independently;
GitHub-hosted critical-path improvement remains post-publication evidence.
The duplicate-removal pass moved module-download and generator proof to
`pr-check`; the resulting `check-full-gentle` passed in 97.82s with 2.16 GB peak
RSS and zero swaps. The DigitalOcean fake-provider lifecycle retained create,
sync, snapshot, golden-image, and cleanup proof while its measured time fell
from 11.66s to 2.28s. Minimal and PostgreSQL vertical-slice profiles passed in
68.53s and 71.67s. Exact nightly improvement remains post-publication evidence.
The path-routing follow-up now skips both generated-service jobs for explicitly
template-independent implementation changes, reports the ten slowest tests from
the existing coverage artifact, and exposes a manual setup-go cache A/B without
changing push or pull-request cache behavior. Warm minimal and PostgreSQL
profile proof passed in 40.59s and 24.48s.
Global constraints: Preserve every required test/security proof and unrelated
working-tree content. Do not mutate remote policy or unattributed Docker data.

- [x] T1: Cache and secret-scan owners implement the accepted fast/fail-closed
  behavior.
  - Source: `spec.md` Behavior and contract delta; `design/overview.md`
    Ownership and flow.
  - Owner/surface/resources: `scripts/ci/template-init-check.sh`,
    `scripts/ci/secret-scan.sh`, `Makefile`; local Go cache and disposable Git
    repositories.
  - Depends on: none
  - Proof: `make secret-scan-check`, `make secret-scan`,
    `make secret-scan-history`, and cache-path absence after
    `make template-init-check`; `REQUIRE_DOCKER=1 make test-integration`
    executes uncached container proof.

- [x] T2: CI and release routing preserve change-scoped PR proof and complete
  historical main/release proof.
  - Source: `spec.md` Invariants and edge cases.
  - Owner/surface/resources: `.github/workflows/ci.yml`,
    `.github/workflows/nightly.yml`, `.github/workflows/cd.yml`.
  - Depends on: T1
  - Proof: workflow YAML parses; exact static inspection shows PR base SHA
    routing and historical main/release targets.

- [x] T3: Human and agent instructions make cache ownership, scanner tiers, and
  measured resource limits durable.
  - Source: `spec.md` Decisions, constraints, and authorities.
  - Owner/surface/resources: `AGENTS.md`,
    `docs/build-test-and-development-commands.md`,
    `docs/configuration-source-policy.md`, `CONTRIBUTING.md`,
    `scripts/ci/project-structure-check.sh`.
  - Depends on: T1
  - Proof: `make project-structure-check`,
    `make template-owned-purity-check`, and `git diff --check`.

- [x] T4: The integrated candidate passes focused and broad proof; only
  explicitly attributable obsolete local caches are removed.
  - Source: `spec.md` Success criteria and proof expectations.
  - Owner/surface/resources: complete bounded diff, Docker daemon, exact
    `.cache/go-build` and `.cache/golangci-lint` directories.
  - Depends on: T1, T2, T3
  - Proof: shell syntax, Make dry runs, secret-scan proofs,
    `make template-init-check`, `make check`, and Docker-backed integration;
    broader gates run when their changed-surface claim requires them.

- [x] T5: Replace measured wall-clock progress with deterministic test-owned
  synchronization.
  - Owner/surface/resources: bootstrap listener/shutdown tests and HTTP
    idempotency test.
  - Depends on: none
  - Proof: focused repeated tests and race detector.

- [x] T6: Route Markdown-only CI changes without weakening the stable required
  gate.
  - Owner/surface/resources: `.github/workflows/ci.yml`,
    `scripts/ci/ci-change-scope.sh`, `Makefile`.
  - Depends on: none
  - Proof: scope self-check, workflow parse, aggregate result-policy fixtures,
    and database profile initialization.

- [x] T7: Reuse one runtime image and restore the PostgreSQL image authority.
  - Owner/surface/resources: CI container job, `pgtest.DefaultImage`, Make and
    benchmark scripts.
  - Depends on: T6
  - Proof: benchmark infrastructure self-check, profile initialization, and
    Docker-backed migration/container gate.

- [x] T8: Close the follow-up with focused concurrency proof and the complete
  gentle gate.
  - Owner/surface/resources: full accepted diff and local Docker daemon.
  - Depends on: T5, T6, T7
  - Proof: focused repeated/race tests, instruction/structure checks,
    `make template-init-check`, and `make check-full-gentle`.

- [x] T9: Reuse the local Trivy database without weakening image findings.
  - Owner/surface/resources: `Makefile`, shared `trivy-cache` Docker volume.
  - Depends on: T1
  - Proof: cold/warm `container-security` runs use the explicit cache directory,
    retain HIGH/CRITICAL fail-closed flags, and leave the named volume present.

- [x] T10: Remove duplicate module and vet work while preserving their owners.
  - Owner/surface/resources: `Makefile`, template-init contract, merge/nightly
    workflows.
  - Depends on: T1, T6
  - Proof: module subtargets and aggregate pass; current-tree tests use
    `-vet=off`; lint retains `govet`; generated/integration tests retain default
    vet; template-init passes.

- [x] T11: Remove redundant OpenAPI, Docker, and successful-output work.
  - Owner/surface/resources: `Makefile`, `build/docker/Dockerfile`,
    `scripts/ci/template-init-check.sh`.
  - Depends on: T10
  - Proof: OpenAPI runtime/lint/check pass, generated database profiles build,
    Docker produces both `/service` and `/migrate`, and test/gosec output
    formats retain failure detail.

- [x] T12: Avoid Go and Dependency Review startup for proven Markdown-only CI.
  - Owner/surface/resources: `.github/workflows/ci.yml`, aggregate result
    policy, CI documentation and structure check.
  - Depends on: T6
  - Proof: YAML parses, static job policy shows heavy steps/jobs skipped only
    for `expensive_required=false`, and full/unknown routing remains fail-closed.

- [x] T13: Review the complete optimization candidate and run focused plus
  broad proof without clearing warm caches.
  - Owner/surface/resources: complete bounded diff and current local tools.
  - Depends on: T9, T10, T11, T12
  - Proof: shell/YAML/Dockerfile syntax, project and template contracts,
    module/lint/OpenAPI/security tests, Docker-backed gates, `git diff --check`,
    and a full aggregate appropriate to the changed delivery surface.

- [x] T14: Use prebuilt exact-version CI tools without changing command or
  release-trust ownership.
  - Owner/surface/resources: CI lint job and both CD publish jobs.
  - Depends on: T13
  - Proof: workflow parse and static checks prove tools-module version
    derivation, exact action SHAs, retained Make entrypoint, Cosign version,
    digest signing, and verification.

- [x] T15: Narrow runtime-only caches and Docker invalidation.
  - Owner/surface/resources: CI test jobs, `.dockerignore`, production
    Dockerfile.
  - Depends on: T13
  - Proof: cache-path inspection, both database profiles, warm build comparison,
    and rebuild after a test-only context change.

- [x] T16: Remove duplicate watch, OpenAPI, Trivy, and template-fixture work.
  - Owner/surface/resources: Make, CI/CD workflows, template initialization.
  - Depends on: T13
  - Proof: Make dry runs, OpenAPI checks, workflow inspection, and both template
    profiles.

- [x] T17: Review and validate the complete follow-up candidate.
  - Owner/surface/resources: complete bounded diff.
  - Depends on: T14, T15, T16
  - Proof: syntax/static checks, focused targets, Docker-backed proof,
    `git diff --check`, and the broad gentle gate.

- [x] T18: Serialize heavy local work and remove the remaining redundant
  coverage, linter, security, and container work.
  - Owner/surface/resources: Make, production Dockerfile, template initializer,
    CI/nightly/release workflows, tooling instructions and contract checks.
  - Depends on: T17
  - Proof: Make/YAML/shell/static contract checks; focused coverage, lint,
    gosec, and template-binary execution; both Docker profiles with system
    timezone data; exact binary-size A/B; and `make -j8 check-full-gentle`.

- [x] T19: Apply the measured serial and CI analysis-cache correction.
  - Owner/surface/resources: Make concurrency defaults, CI/nightly/release
    golangci-lint installers, tooling instructions, and structure guards.
  - Depends on: T18
  - Proof: workflow YAML parsed; static inspection found four pinned
    cache-owning action usages and no `skip-cache: true`; a Make query kept
    `LINT_CONCURRENCY=4` under `GOMAXPROCS=6`; `make check`,
    `make template-owned-purity-check`, shell syntax, and `git diff --check`
    passed. No broad parallel rerun was needed because the accepted A/B already
    rejected it.

- [x] T20: Tune Docker and Testcontainers without weakening isolation or
  cleanup.
  - Owner/surface/resources: PostgreSQL test harness, `.dockerignore`, Compose
    healthcheck, CI/nightly/release Testcontainers steps, documentation, and
    structure guards.
  - Depends on: T19
  - Proof: same-host integration improved from a 17.6s baseline median to a
    4.6s tuned median. A controlled A/B found no Ryuk benefit (4.19/4.04s
    enabled versus 4.49/4.48s disabled), so every run retains it. Tmpfs and a
    shared admin pool were measured and rejected. The warm Docker build was
    1.13s, and a Makefile-only edit kept every layer cached at 0.80s. Compose
    reached healthy in 2.95s, reported `fsync`, `synchronous_commit`, and
    `full_page_writes` all `on`, and its trap removed the project. Focused
    integration race, workflow/Compose parsing, project and template contracts,
    the generated PostgreSQL proof, `make check-gentle`, and `git diff --check`
    passed.

- [x] T21: Keep expensive module and generator integrity outside the ordinary
  local aggregate without weakening full or merge proof.
  - Owner/surface/resources: `Makefile`, tooling command documentation.
  - Depends on: T20
  - Proof: Make dry runs show `ci-local` owns manifest drift while `check-full`
    retains `mod-verify` and the complete template contract; focused and broad
    gates pass.

- [x] T22: Split minimal and PostgreSQL generator proof across merge-CI runners.
  - Owner/surface/resources: template initialization harness, CI workflow,
    required aggregate, delivery documentation, and structure guards.
  - Depends on: T21
  - Proof: shell/YAML parsing, invalid-selector failure, both focused profile
    commands, required-result fixtures, and the broad Docker-backed gate.

- [x] T23: Remove repeated nightly/local work while preserving every required
  proof and the complete DigitalOcean benchmark lifecycle.
  - Owner/surface/resources: Make aggregates, nightly workflow, template
    initializer, DigitalOcean runner self-check, tooling instructions and
    structure guards.
  - Depends on: T22
  - Proof: `pr-check` owns module/template integrity; merge/main/release retain
    deterministic gates; one generated-profile linter pass proves clean output
    plus depguard rewriting; the fake remote lifecycle retains create, sync,
    snapshot, golden-image, and cleanup assertions without real retry sleeps;
    benchmark execution and comparison use one valid measured artifact.

- [x] T24: Route generated-service proof by owned paths and expose cheap
  regression evidence.
  - Owner/surface/resources: CI change-scope classifier, required aggregate,
    coverage summary, manual setup-go cache input, and tooling instructions.
  - Depends on: T23
  - Proof: classifier self-tests cover ordinary, mixed, unknown, empty, and
    generator-owned paths; required-result fixtures reject unexpected success
    and missing proof; workflow YAML, shell syntax, structure/purity contracts,
    coverage plus slow-test extraction, and both generated profiles pass.
