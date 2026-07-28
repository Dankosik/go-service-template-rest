# Tooling resource optimization design

## Ownership and flow

- `scripts/ci/template-init-check.sh` inherits Go and linter cache paths.
- The same script accepts `TEMPLATE_INIT_PROFILE=all|minimal|postgres`.
  Local and release proof keep the complete default; merge CI runs the minimal
  and PostgreSQL contracts on separate runners without skipping either.
  Generated repositories detect the absent `scripts/profiles` owner before
  installing language toolchains or the linter.
- `scripts/ci/secret-scan.sh` owns three modes:
  - `change`: materialize the current index plus worktree in a temporary
    directory without `.git` or ignored files, scan it once, then scan
    `merge-base(base, HEAD)..HEAD`;
  - `history`: scan complete Git history;
  - `self-test`: prove the positive and negative routing cases in a disposable
    repository using the pinned Gitleaks tool.
- `Makefile` exposes those modes and the opt-in bounded-concurrency wrappers.
- GitHub workflows select change mode only for pull requests and history mode
  for main, manual, and release execution. Nightly does not rescan unchanged
  history.
- `scripts/ci/ci-change-scope.sh` classifies a NUL-delimited changed-path set.
  Only explicit Markdown instruction, `docs/`, and `specs/` paths produce
  `docs-only`; no paths or any other path produces `full`.
- `repo-integrity` publishes that classification. Runtime-heavy jobs use it,
  while `ci-required` verifies that only those named jobs were skipped.
- The container job builds `service:ci` once, scans it, and passes the same tag
  and embedded version to migration validation.
- Local `container-security` mounts one shared `trivy-cache` volume and points
  Trivy at it explicitly. The volume is persistent input acceleration, not
  scan-result authority.
- `mod-tidy-check` and `mod-verify` split manifest computation from downloaded
  content hashing; `mod-check` remains their fail-closed aggregate.
- `ci-local` and `check-full` keep the manifest check in ordinary host and
  Docker-backed loops. `pr-check`, merge CI, and release proof retain
  downloaded-content and template-initialization ownership.
- Current-tree test commands skip their default vet subprocess because
  mandatory golangci-lint already runs `govet`. Generated checkout and
  integration-tag commands keep default vet.
- OpenAPI runtime discovery and execution share one JSON test stream; absence
  of a matching run event fails the target.
- Docker chooses `./cmd/service` and optional `./cmd/migrate` before one
  multi-package build, preserving the database profile without a second
  compiler process.
- CI installs the exact golangci-lint release named by `tools/go.mod`, restores
  and saves its action-owned analysis cache, then enters through `make lint`;
  runtime-only test jobs do not restore the tools dependency graph.
- Publish jobs use the SHA-pinned Cosign installer and reuse the first Trivy
  setup for SBOM generation.
- BuildKit bind-mounts the filtered source context for compilation and
  migration copying, leaving only module manifests and build outputs in cached
  layers.
- OpenAPI runtime proof owns main-package compilation, the reference target
  owns optional example compilation, and the final PostgreSQL fixture owns the
  removed duplicate initialization assertions.
- `.NOTPARALLEL` serializes the prerequisites of broad/heavy Make aggregates
  even when a caller exports parallel `MAKEFLAGS`; direct prerequisites replace
  recursive multi-goal dispatch where that serialization must apply. A
  same-target reference-laptop A/B measured `make -j4` at 294.6s versus 138.7s
  serially.
- Gentle wrappers add a lower scheduler priority and export one
  `GOMAXPROCS` budget. gosec consumes it only when explicitly present, while
  golangci-lint retains its independently measured four-worker default.
- All golangci-lint entrypoints wait on the tool's serial-runner lock. Workflow
  jobs that need the linter install the exact `tools/go.mod` release binary and
  pass either the Make command override or the template-check binary override.
- Coverage uses set-mode counters because only statement presence and the
  resulting ratio are consumed.
- The pinned Distroless filesystem owns named timezone data; removing
  `timetzdata` avoids embedding a second database into each compiled command.
- The shared PostgreSQL Testcontainers process retains its readiness and
  per-test database boundaries while its disposable cluster disables only
  crash-durability sync work. Cross-run reuse remains forbidden.
- Local and GitHub-hosted jobs retain the Ryuk sidecar as the
  interrupted-process cleanup owner.
- `.dockerignore` removes compiler-irrelevant root tooling inputs from the
  source-bind checksum. Compose skips only disposable initdb sync, checks
  startup every second, and lowers only its post-readiness healthcheck cadence.

The worktree snapshot starts from `git checkout-index`, overlays modified,
deleted, and untracked non-ignored paths, and is always removed. This preserves
path-based Gitleaks rules while avoiding a directory scan through `.git` and
large ignored caches.

## Failure and cleanup

- Snapshot or Git failures stop the scan.
- An unavailable base ref prints the reason and invokes the historical mode.
- Expected secret findings are successful only inside `self-test`; production
  modes preserve Gitleaks' exit status.
- Every temporary directory is created with `mktemp` and removed by its owning
  process.

## Deterministic test flow

- The listener-limit test closes requests explicitly and observes the point at
  which the configured concurrency limit is reached.
- Time-driven shutdown and idempotency tests execute inside `testing/synctest`;
  timers advance only when the test bubble is blocked.
- Outer real-time waits remain finite diagnostics for broken synchronization.

## PostgreSQL image authority

`pgtest.DefaultImage` is read by integration code and extracted by Make and the
benchmark scripts. The benchmark infrastructure fixture copies that authority,
so removal or path drift fails its local self-check before nightly execution.
The fake DigitalOcean provider sets its SSH and deletion-poll delays to zero
while retaining create, sync, snapshot, golden-image, and cleanup lifecycle
assertions.
The infrastructure self-check executes the Go benchmark path once and reuses
that valid artifact as the comparison input; a second identical compiler run
would exercise no new branch.

## Deliberately unchanged

- The Gitleaks configuration and historical baseline semantics do not change.
- The application cgroup-derived memory limit remains runtime-only.
- BuildKit cache-mount ownership and OrbStack machine-wide settings remain
  unchanged.
- Testcontainers' two-stage PostgreSQL wait strategy, Ryuk cleanup,
  per-test database isolation, and PostgreSQL image family remain unchanged.
- Coverage parsing remains unchanged: its measured saving was too small to
  justify a second stateful artifact or a new script.
- Required merge, main, PR, release, generated-contract, migration, and
  container gate membership remains unchanged. Nightly omits deterministic
  duplicates already required for the same commit.
