# Keep build and test tooling fast without weakening proof
status: done

## Scope and non-goals

Optimize the existing Go, Make, GitHub Actions, Gitleaks, BuildKit, and
Testcontainers paths for repeated local and CI execution. Keep the laptop
responsive without replacing native tooling or reducing test and security
coverage.

In scope:

- standard Go and golangci-lint cache ownership in local and GitHub execution;
- change-scoped secret scanning for local and pull-request work;
- full-history secret scanning on main and release paths;
- an opt-in bounded local Go-concurrency profile;
- deterministic synchronization in the three measured wall-clock-bound tests;
- fail-closed Markdown-only CI routing and one runtime-image build per CI run;
- one PostgreSQL test-image authority for tests, Make, and benchmark tooling;
- durable instructions and mechanical proof against the observed regressions;
- removal of obsolete repository-local cache copies after their owner changes.
- persistent local Trivy database reuse without automatic pruning;
- separation of module-manifest drift from downloaded-content verification;
- one `govet` owner for current-tree proof and retained vet where lint does not
  cover the tested generated/integration surface;
- removal of redundant subprocesses, analyzer passes, Docker compilation, and
  successful-run output from confirmed hot tooling paths.
- prebuilt, exact-version CI installation for golangci-lint and Cosign;
- dependency-accurate Go cache keys for runtime-only test jobs;
- BuildKit source bind mounts and runtime-only Docker context ownership;
- removal of duplicated Trivy setup, OpenAPI compilation, watch vet, and
  template-initialization work.
- statement-presence coverage instrumentation instead of unused atomic counts;
- serial heavy-gate and golangci-lint admission across concurrent local
  sessions;
- scheduler-friendly gentle targets, one Go/gosec concurrency budget, and an
  independent measured golangci-lint worker cap;
- removal of duplicate embedded timezone databases from Distroless images;
- prebuilt exact-version golangci-lint installation in every workflow job that
  invokes it.
- disposable PostgreSQL container startup and write-path tuning without
  changing SQL, transaction, migration, or isolation semantics;
- narrower production build invalidation and lower steady-state Compose
  healthcheck wakeups.

No new build system, automatic cache or Docker pruning, cross-run Testcontainers
reuse, GitHub Actions Docker-layer cache, global tooling `GOMEMLIMIT`, or
unmeasured Compose/BuildKit resource limit is introduced.

## Behavior and contract delta

- `template-init-check` inherits the canonical cache paths selected by Go and
  golangci-lint. It does not redirect work into repository-local `.cache`
  directories that GitHub Actions does not restore.
- `make secret-scan` scans the current tracked/untracked non-ignored worktree
  and every commit after its merge base with `SECRET_SCAN_BASE_REF`, defaulting
  to `origin/main`. If that ref is unavailable, it fails safe by scanning full
  history.
- `make secret-scan-history` retains the complete historical Gitleaks gate.
- Pull-request CI uses the pull request base SHA for change-scoped scanning.
  Main pushes, workflow dispatch, and release preflight use the full historical
  gate. Nightly does not rescan unchanged Git history.
- `make check-gentle`, `make ci-local-gentle`, and
  `make check-full-gentle` run the unchanged underlying gates with bounded Go
  concurrency and lower CPU scheduling priority. The settings remain
  overrideable and do not limit Docker separately.
- Heavy aggregate prerequisites remain serial under inherited `MAKEFLAGS`, and
  golangci-lint callers wait on its native runner lock rather than allowing
  duplicate analyzer processes. One same-target reference-laptop A/B measured
  138.7s serially versus 294.6s with `make -j4`.
- Time-driven tests use `testing/synctest`; the accepted-connection limit test
  waits for its owned saturation event instead of sleeping or retaining idle
  keep-alive connections.
- CI skips runtime-heavy jobs only when every changed path belongs to the
  explicit Markdown-only allowlist. Empty, mixed, unknown, or unresolvable
  comparisons run the full matrix.
- CI builds one production runtime image and reuses it for Trivy and applicable
  migration validation.
- `pgtest.DefaultImage` is the PostgreSQL test-image authority consumed by
  integration tests, Make, and local/remote benchmark checks.
- `make mod-tidy-check` owns root/tools manifest drift and Go-version parity;
  `make mod-verify` owns downloaded-content hashes; `make mod-check` composes
  both. Template fixtures invoke only the first because the repository gate
  already verifies the same root/tools download cache.
- The ordinary `ci-local` and Docker-backed `check-full` loops run
  `mod-tidy-check` but not `mod-verify` or `template-init-check`; `pr-check`,
  merge CI, and release proof retain both expensive contracts.
- Merge CI runs the minimal and PostgreSQL generator profiles as independent
  fail-closed jobs. The complete local/release target still runs both profiles
  when `TEMPLATE_INIT_PROFILE` is unset. Generated repositories stop those
  jobs after checkout because they no longer contain `scripts/profiles`.
- Mandatory golangci-lint owns `govet` for the current repository. Repeated
  unit/race/coverage/fuzz/flake/OpenAPI tests disable their duplicate vet pass;
  disposable generated profiles and integration-tag tests retain it.
- Local Trivy scans reuse a shared named cache volume and suppress progress
  noise without suppressing findings.
- OpenAPI runtime proof executes and discovers its required test in one
  `go test -json` process. Redocly prefers its warm npm cache without weakening
  its exact pin.
- The production Docker build compiles service and optional migrator packages
  in one `go build` invocation.
- The lint job resolves golangci-lint's version from `tools/go.mod`, installs
  that release through a SHA-pinned official action, uses the action-owned
  analysis cache, and calls the unchanged `make lint` owner.
- Release-preflight and template PostgreSQL proof jobs use that same
  exact-version prebuilt golangci-lint path.
- Race, coverage, and integration jobs key setup-go only from the runtime
  `go.sum`; jobs that execute repository tools continue to include
  `tools/go.sum`.
- Publish jobs install the exact Cosign release without Go and reuse the first
  Trivy setup for SBOM generation.
- Docker compilation sees source through a read-only BuildKit bind mount.
  Runtime-irrelevant API, example, tool, test, and testdata files are absent
  from the build context.
- The OpenAPI runtime test compiles the main generated package; a separate
  target compiles only the optional reference package. Watch mode disables the
  same duplicate vet pass as the ordinary unit-test target.
- The stronger PostgreSQL template fixture owns the workflow/spec preservation
  assertions formerly repeated by an earlier fixture.
- Coverage records whether statements executed with `-covermode=set`; no gate
  consumes atomic execution counts.
- `gosec` receives the repository Go version directly and follows an explicit
  `GOMAXPROCS` concurrency bound when one is set.
- The Distroless system timezone database remains the runtime owner; service
  and migrator binaries do not embed a duplicate `time/tzdata` copy.
- Markdown-only CI avoids repo-integrity Go setup, module/format/sqlc work, and
  Dependency Review while retaining repository integrity, secrets, and
  `ci-required`. The secret scanner keeps its own Go setup because it executes
  the pinned repository tool.
- The PostgreSQL Testcontainers process keeps the module's two-part
  `BasicWaitStrategies` and one isolated database per test. Its disposable
  cluster skips crash-durability sync work; no crash-recovery behavior is
  claimed by this integration suite.
- Local and GitHub-hosted Testcontainers execution keeps Ryuk enabled because
  disabling it produced no measured wall-time benefit and weakened interrupted
  process cleanup.
- Root tooling and delivery-control files that the Go compiler cannot consume
  are absent from the production Docker context, so their edits do not
  invalidate the source-bind compile step.
- Compose retains one-second startup checks but uses a longer steady-state
  interval after PostgreSQL first becomes healthy.
- The disposable Compose database also skips only initdb's final sync; the
  running server keeps its normal durability settings.
- Nightly owns only time-varying or repeated evidence: flake/fuzz execution,
  fresh integration, benchmark lifecycle proof, current Go security databases,
  and a fresh image scan. Deterministic module, format, lint, race, OpenAPI,
  and history-secret gates remain mandatory in merge/main/release paths instead
  of repeating on the same commit.
- The DigitalOcean lifecycle self-check keeps provisioning, source sync,
  snapshot, golden-image boot, and cleanup proof while setting its synchronous
  fake provider's SSH and deletion-poll delays to zero.

## Invariants and edge cases

- Uncommitted files and secrets added and deleted inside a feature branch remain
  detectable.
- A missing or misspelled base ref never converts secret scanning into success.
- Ignored local files such as `.env`, caches, artifacts, and `.git` objects are
  not treated as the reviewable worktree.
- The historical baseline remains the only accepted suppression owner; scanner
  performance never authorizes a new baseline entry or target-size exclusion.
- Full-history security proof remains fail-closed on main and release.
- Tool caches stay warm between runs. Cleanup is explicit and limited to a
  cache proven obsolete or a separately inventoried Docker resource.
- Testcontainers reuse does not cross test-process boundaries. Ryuk,
  isolated databases, explicit termination, and Mac-safe port readiness remain
  intact.
- Unattributed Docker containers, images, volumes, and builder cache are
  inventory only and are never removed by this work.
- Docker-backed integration execution is fresh; a cached Go result is not
  current container startup, health, behavior, or cleanup evidence.
- Repository integrity, secret scanning, and `ci-required` remain active for
  Markdown-only changes; Dependency Review remains active for non-doc PRs.
- Real-time deadlines in deterministic tests are diagnostic ceilings, not the
  mechanism that makes the expected path progress.
- A warm scanner/package cache is not correctness evidence by itself; exact
  pins, module verification, generated drift, lint, and vulnerability exit
  policy remain unchanged.

## Decisions, constraints, and authorities

- `Makefile` remains the command authority; `scripts/ci/secret-scan.sh` owns the
  cross-platform snapshot/range logic that would be misleading in a recipe.
- Git and Gitleaks remain the only change and secret scanners. No dependency is
  added.
- `actions/setup-go` owns GitHub Go cache restoration; `go env GOCACHE` and the
  tool defaults own local paths.
- Testcontainers owns interrupted-process cleanup through Ryuk in local and
  GitHub-hosted execution.
- `scripts/ci/ci-change-scope.sh` owns the finite Markdown-only allowlist and
  its self-check; GitHub owns base-to-HEAD resolution and fails full when that
  comparison is unavailable.
- `internal/infra/postgres/pgtest/pgtest.go` owns the PostgreSQL test image.
- `GENTLE_GOMAXPROCS=6` and `GENTLE_NICE=10` form an opt-in profile for the
  measured 10-core/16-GiB development Mac, not a universal runtime or Docker
  limit.
- `GOMAXPROCS` is the shared gentle-mode concurrency input for Go and gosec.
  golangci-lint retains the independent `LINT_CONCURRENCY=4` default because
  automatic concurrency did not improve measured wall time.
- A shared Docker volume is the local Trivy cache owner. It is portable across
  derived repositories and remains outside repository-local ignored state.
- `govet` is not removed: golangci-lint is its current-tree owner, while
  generated checkout and integration paths retain Go's default pass.
- Technical design is limited to the existing Make/script/workflow owners; no
  runtime topology, API, data, generated source, or Go package boundary changes.
- A separate test plan is unnecessary: the finite security branches have an
  executable self-check, while existing repository gates own syntax, docs,
  workflow, and integration proof.

## Success criteria and proof expectations

- Running `template-init-check` with cache environment variables unset does not
  recreate `.cache/go-build` or `.cache/golangci-lint`.
- The secret-scan self-check proves clean success, untracked-secret rejection,
  base-to-HEAD rejection after a secret is deleted, full-history rejection, and
  fail-safe missing-base behavior.
- Current worktree and historical scans pass against the real repository.
- Make, shell, workflow, documentation, and template-owned instruction checks
  pass.
- The bounded profile executes the same target graph as its normal counterpart.
- Docker-backed integration proof passes before any container limit is proposed.
- Focused repeated and race tests pass for every changed synchronization path.
- CI change-scope self-check, workflow parsing, profile generation, benchmark
  infrastructure check, and the broad gentle gate pass.
- CI skips generated-service proof only when every changed path belongs to the
  explicit template-independent implementation allowlist. Empty, mixed,
  unknown, generator-owned, or unresolvable path sets run both profiles.
- The coverage job reports the ten slowest tests from its existing test2json
  artifact without executing tests again.
- Manual CI can disable only the `actions/setup-go` module/build cache for a
  sequential same-ref A/B; push and pull-request behavior remains cache-on.
- Focused minimal and PostgreSQL profile commands both pass, and the required
  aggregate rejects failure or an unexpected skip from either CI job.
- The module subtargets compose to `pr-check`, merge, and release gates; the
  OpenAPI runtime check rejects an absent matching test, and Docker profile
  generation preserves both database-none and database-postgres builds.
- Focused before/after evidence confirms the retained optimizations reduce
  wall time or output without increasing peak resource use materially.
- Workflow parsing proves exact action pins, runtime-only cache keys, retained
  preflight Go setup, action-owned golangci-lint caching, and unchanged digest
  signing/SBOM verification order.
- Both database profiles build through the source-bind Dockerfile; a test-only
  file change does not invalidate the production compilation layer.
- Make graph inspection and structure checks prove aggregate serialization,
  native linter locking, coverage mode, gosec version ownership, and prebuilt
  linter installation in every CI, template, and release job that invokes it.
- Nightly workflow inspection proves that repeated/time-varying checks remain
  while deterministic merge gates and advisory-only jobs are absent.
- The runtime image still resolves named timezones after removing embedded
  tzdata, and both database profiles build successfully.
- Repeated integration proof preserves all PostgreSQL tests while measuring
  container startup/write tuning against the same warm image, daemon, test
  command, and idle-host condition.
- A tooling-only root-file edit leaves the production Go build step cached.
  Compose startup remains one-second-granularity while its healthy steady state
  no longer executes `pg_isready` every five seconds.

## Risks, assumptions, and reopen conditions

- Exact GitHub wall-time improvement remains a post-publication measurement;
  reopen performance only if the template job still bypasses the restored cache
  or remains on the critical path after this change.
- The profile split is accepted structurally, not as a claimed GitHub speedup,
  until a run on the same workflow revision shows the new critical path.
- Exact coverage and link-time savings require an idle equivalent-host A/B;
  structural removal and semantic equivalence are not reported as a measured
  speed percentage.
- Reopen embedded timezone data if the final runtime image no longer supplies
  `/usr/share/zoneinfo`.
- Reopen Docker resource policy only with a representative peak CPU/RSS sample
  and enough headroom to avoid throttling or OOM-driven flakiness.
- Reopen GitHub Docker cache only when repository cache storage has durable
  headroom and the selected Buildx driver supports the backend.
- Reopen the template-independent path allowlist when initialization begins
  transforming another implementation path; profile-marker ownership checks
  prevent such a path from silently bypassing generated-service proof.
- Exact setup-go cache benefit remains a post-publication same-ref A/B using
  sequential manual runs with `go_cache_enabled=true` and `false`.
- Reopen heavy-prerequisite serialization only when an equivalent A/B after a
  host, toolchain, or aggregate change improves wall time without unacceptable
  peak RSS or swapping.
- Reopen Ryuk disabling only if a same-workload A/B proves a material wall-time
  benefit and another crash-cleanup owner proves cleanup after timeout, kill,
  and agent interruption.
- Reopen Alpine PostgreSQL or cross-run container reuse only with explicit
  collation/extension compatibility and stale-state isolation proof.
- Docker resources not attributable to this repository remain untouched.
