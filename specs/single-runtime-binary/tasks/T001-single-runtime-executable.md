# T001 — One runtime executable for every selected role

Outcome:
Today the image and `make` targets build five separately linked role
executables from five `package main` directories. After this task, a single
`cmd/entrypoint` main selects one role per process from the final element of
`argv[0]` (or the link-time `pinnedRole` for local `make` artifacts), each
`cmd/<role>` is a non-main package exporting only `Main(args []string) int`,
the image ships one executable at `/service` plus relative symlinks for each
selected optional role, and every template surface, generated profile, and
current-guidance document follows that layout. Falsified by any R1 through R9
or Success row in spec.md failing on the assembled tree.

Consumes:
- [spec.md](../spec.md) R1-R9, Success, Proof expectations, Assumptions A1-A3 — required behavior and the measured bounds.
- [design/system.md](../design/system.md) M1-M4, "Enforcement of critical invariants", "R7: synced tooling on an unmigrated layout", "Repository Architecture changes", "Release closure", "Review dispositions" F1-F4 — selected mechanism, guard activation boundary, residue rule, and doc vocabulary.
- [design/ownership-map.md](../design/ownership-map.md) RS1-RS9, Files, RS1 reuse record — exact package and file actions, allowed dependencies, and cleanup.
- Baseline working tree (commit `aeda2664f` plus the five uncommitted baseline edits) — the foundation to build on, and the reference for M1-M3 and R2 baseline comparisons.

Provides:
- Final result: the assembled single-executable candidate satisfying R1-R9 and Success.
- A preserved baseline reference (for example a separate worktree at `aeda2664f` with the baseline edits applied, or a saved patch of them) captured before any baseline-edited file changes, so final validation can build the baseline images and baseline fail-closed outputs. Do not use `git stash` or reset the main checkout to obtain it. The uncommitted Dockerfile baseline edit cannot be recovered from git once this task rewrites the Dockerfile, so capture the snapshot first and record its absolute location (worktree path or patch path) in this packet under a `Baseline snapshot:` line and in the ledger T001 `Resume` field, where final validation reads it.

Boundary:
Implements RS1-RS9 exactly as the ownership map assigns them:
- RS1 `cmd/entrypoint/main.go` (new): stdlib-only dispatcher, ordered role table with profile-marked optional entries and imports, `pinnedRole`, exact-match selection, one stderr line and exit 2 on no match, no role names outside markers except `service`. Unit tests for the R3 cases live in that package.
- RS2/RS3 role packages: `cmd/<role>/main.go` becomes `role.go` with packages `service`, `worker`, `jobsworker`, `outboxrelay`, `migrate`; bodies and imports unchanged; package comments state the role is imported only by `cmd/entrypoint` and that role code keeps I/O, registration, and logging out of initializers (system.md F3). Seam files change only their package clause and "binary" wording.
- RS4 `build/docker/Dockerfile`: one `go build -o /out/bin/service ./cmd/entrypoint` with the existing PGO, trimpath, readonly, buildvcs, and ldflags arguments; marked, `-d`-guarded `ln -s service /out/bin/<role>` lines, with `migrate` inside `database-postgres` markers; `touch -h`; one directory `COPY --link --from=build /out/bin/ /`; never `-X main.pinnedRole`. Preserve the uncommitted baseline edits (Debian 13 bases, `COPY --link`).
- RS5 `make/template.mk`: build and run `./cmd/entrypoint` with `-X main.pinnedRole=<role>`; keep target names, output paths, `not applicable` probes, and PGO handling; add the R7 precondition that exits 1 naming `docs/runtime-executable-migration.md` when a role directory exists and `cmd/entrypoint/main.go` does not.
- RS6 `scripts/ci/changed-surfaces.sh` (`cmd/entrypoint/*`, `cmd/*/role.go`, keep `cmd/*/main.go`) and `scripts/dev/benchmark.sh` fingerprint list (`cmd/entrypoint`).
- RS7 `scripts/ci/init-module-contract-check.sh`: per-profile role-isolation guard and the `./cmd/service` exclusion of `internal/infra/postgresmigrate`.
- RS8 the four `test/*_process_integration_test.go` builds target `./cmd/entrypoint` with their existing tags, output to `<tmp>/<role-name>`.
- RS9 vocabulary rewording in the docs system.md lists, the R6 sweep of current-guidance references to relocated paths (tracked Markdown outside `specs/**` and source comments; R6 exemptions kept), and the new `docs/runtime-executable-migration.md` with every R8 element, listed in `template-owned.paths` and linked from `docs/template-sync.md`, with no profile markers or service-specific content. The guide also tells maintainers that role code keeps I/O, registration, and logging out of package initializers (system.md "Enforcement of critical invariants" row 1, F3), and it names only `make` targets defined in `make/template.mk` (system.md "Repository Architecture changes").

Cleanup: no `main.go` and no `package main` remains in any role directory; the Dockerfile `build_packages` assembly and per-role `COPY` lines and the per-role `go build|run ./cmd/<role>` commands are removed. `scripts/init-module.sh` gets no planned change; if generation fails, fix marker placement in RS1 or RS4 first.

Exclusions: no change to role runtime behavior, configuration, flags, ports, telemetry, `railway.toml`, base image digests, PGO policy, or other CI and release gates; no subcommands, compression, or image-size CI gate; no edits to derived repositories or template sync fan-out; no compatibility branch for the old layout beyond the RS5 precondition.

Mutable owners:
- `cmd/entrypoint` (new) and the five `cmd/<role>` role-entry packages, including the jobs-worker and migrate seam files' package clauses
- Shipped image build (`build/docker/Dockerfile`)
- Local role build and run targets (`make/template.mk`)
- Synced CI and dev tooling: `scripts/ci/changed-surfaces.sh`, `scripts/dev/benchmark.sh`, `scripts/ci/init-module-contract-check.sh`
- Process integration tests in `test/`
- Architecture and current-guidance docs, the new migration guide, `docs/template-sync.md`, and `template-owned.paths`

Exclusive locks:
- `build/docker/Dockerfile` (carries uncommitted baseline edits)
- `make/template.mk`
- `template-owned.paths` manifest
- `cmd/` role packages and the profile markers `init-module.sh` strips

Final validation:
- Claim: R1-R9 and the Success table in spec.md hold on the assembled tree; the design's added proof surfaces (role-isolation guard, R2 role exec checks that also falsify a pinned image, generated-profile residue `rg` over `cmd/entrypoint/main.go` and the Dockerfile, role-named process integration builds) pass.
- Checks: ordinary local completion per AGENTS.md Validation budget (matching build and relevant unit tests, including the new `cmd/entrypoint` R3 tests). Explicit additional requirements, sourced from spec.md "Success" and "Proof expectations" and system.md "Release closure": build and measure M1, M2, M3 against baseline measurements taken from the preserved baseline; count executable payloads in the image; `go version -m` on M3 and a no-messaging configuration; `make runtime-image-check` and `make migration-validate` on the new image; exec the template `/worker` and a `DATABASE=postgres JOBS=postgres WEBHOOKS=none INBOUND_WEBHOOKS=none` generated service's (`JOBS=postgres` requires `DATABASE=postgres`, `scripts/init-module.sh`) `/jobs-worker` and compare exit status and stderr with baseline; existing process integration tests for the roles; `make template-init-check` across profile combinations (this also runs the RS7 guard); one run of the synced tooling against an unmigrated five-binary layout; the R6 `rg` sweep; rebuild identical inputs to the same digest; `PGO_PROFILE=auto` rejected and a stale manifest failing closed; `docs-contract-check` and `template-owned-purity-check`. Heavy gates need `ALLOW_HEAVY=1`/`ALLOW_FULL=1` and run serially. Concrete commands are executor-owned. Docker is required for the image claims; if unavailable, those claims are Blocked, not passed.
- Observable: the M1 size and the gzip figure are reported with baseline values; every R2 entry path behaves as baseline; an unknown name exits 2 with one stderr line; generated services contain exactly their selected roles; the unmigrated-layout run either works or exits non-zero naming the migration guide. If M1 exceeds 50 MB, report the per-file breakdown and reopen Specification instead of passing.

Reopen if:
Specification when a deployment starts a role other than through an R2 exec entry path (A1) or M1 exceeds 50 MB. Technical Design when a platform resolves the symlink before exec, a payload count finds more than one copy, an importer of a `cmd/<role>` package appears, a shipped build needs `pinnedRole`, or BuildKit `COPY --link` does not preserve the relative links or their pinned mtimes.

Baseline snapshot: `/Users/daniil/Projects/Opensource/go-service-template-rest.single-runtime-baseline` (detached worktree at `aeda2664f` with the five baseline edits applied, byte-identical to the main checkout at capture); patch copy `/Users/daniil/Projects/Opensource/go-service-template-rest.single-runtime-baseline.patch`.
