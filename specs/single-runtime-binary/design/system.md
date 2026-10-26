# Single runtime executable: system and repository design

status: ready

Inputs: [spec.md](../spec.md) (ready) and [intent.md](../intent.md). Placement
detail lives in [ownership-map.md](ownership-map.md). Technical Design Review:
`CONCERNS` (reviewer a82fa04f0062f5c0e). Dispositions are under Review
dispositions.

## Drivers and hard constraints

- R1: one executable payload in the image; unselected role code is not linked.
- R2/R3: existing exec-form entry paths keep their behavior; the role comes
  from the final element of `argv[0]`, exact match; an unknown name exits 2
  before any program action.
- R4: local `make` artifacts keep their names and run their role.
- R5/R6: profile stripping and role-local seams keep working. Go's `internal`
  rule must keep each role's composition private to that role.
- R7: synced template-owned tooling must either work on an unmigrated
  five-binary layout or fail with a message that names the migration guide.
- R9: distroless static, nonroot, reproducible digest, PGO fingerprint
  coverage, unchanged `railway.toml`, and unchanged role dependency graphs.

## Selected mechanism

### M1. Multi-call executable chosen by `argv[0]`

A new main package `cmd/entrypoint` is the only `package main` under `cmd/`.
It holds an ordered table of the selected roles: `service`, `worker`,
`jobs-worker`, `outbox-relay`, `migrate`, in R2 order. Each optional entry and
its import sit inside the same profile markers the Dockerfile uses today:
`messaging-nats-jetstream`, `jobs-postgres`, `outbox-postgres`, and
`database-postgres`. `init-module.sh` already finds and strips markers in every
file, so it removes an unselected role's import together with its directory.
This keeps the role's code and its exclusive modules out of the link (R1).

Selection order:

1. If the link-time variable `pinnedRole` is non-empty, the candidate name is
   `pinnedRole`. Otherwise it is the final `/`-separated element of
   `os.Args[0]`. An empty `os.Args` gives an empty name.
2. An exact, case-sensitive match against the table selects that role, and the
   process exits with `os.Exit(<role>.Main(os.Args[1:]))`.
3. With no match, the process writes one stderr line containing the candidate
   name and the selected role names in table order, writes nothing to stdout,
   and exits 2. A `pinnedRole` naming an unselected role is rejected the same
   way.

Before step 2 the dispatcher reads no file, no environment variable, and no
flag, installs no signal handler, and does no I/O other than that stderr line.
Each role keeps its own configuration, signal, telemetry, and logging setup
behind its `Main`.

`pinnedRole` exists only for R4's local artifacts, which are named
`bin/<service-name>-worker` and so on, so R3's name rule cannot match them.
Only `make/template.mk` sets it, with `-ldflags -X main.pinnedRole=<role>`.
The Dockerfile never sets it, so the shipped executable always selects by
`argv[0]`. `go version -m` cannot show this: it omits `-ldflags` under
`-trimpath` (go.dev/issue/52372), which the Dockerfile uses. The guarantee is
static: the Dockerfile is the only shipped build owner and its ldflags carry
no `-X main.pinnedRole`. A runtime falsifier also exists. A pinned image runs
its pinned role for every entry path. The R2 proof of the template's
`/worker`, which must fail with `natsjs.ErrRejected` and exit 1, therefore
fails on a service-pinned image. So does `/migrate` validation.

### M2. Role packages stay in their current directories

Each `cmd/<role>` becomes an importable non-main package: `service`, `worker`,
`jobsworker`, `outboxrelay`, and `migrate`. Each exports exactly one
declaration, `func Main(args []string) int`. That function holds today's
`main` body unchanged: the same `bootstrap.Run` or `run` call, the same stderr
or stdout writing, and status 1 on failure. `cmd/<role>/main.go` becomes
`cmd/<role>/role.go`. Every `cmd/<role>/internal/bootstrap` stays in place, so
Go's `internal` rule still stops one role from importing another role's
composition. The existing seams stay in their directories and change only
their package clause:

- `cmd/worker/role.go` keeps the nil `HandlerBuilder` argument and its comment.
- The `cmd/jobs-worker/builder*.go` files, including the build-tag test
  builders, `builder_webhooks.go` with `registerInboundWebhookHandlers`, and
  `inbound_webhook_bindings.go`, stay in package `jobsworker`.
- `cmd/migrate/run.go` moves into package `migrate`.

`cmd/outbox-relay/outboxworker` shows that a non-internal package under
`cmd/<role>/` already has precedent here.

### M3. Image layout: one file plus relative symlinks

The Dockerfile build stage runs one `go build` of `./cmd/entrypoint` with the
existing PGO, trimpath, readonly, buildvcs, and ldflags arguments, writing
`/out/bin/service`. In the same `RUN`, it creates each selected optional
role's link with `ln -s service /out/bin/<role>`. Each `ln` sits inside that
role's profile marker and keeps the existing `if [ -d ./cmd/<role> ]` guard.
The `migrate` link moves inside `database-postgres` markers so that a
generated service keeps no residue (R5). The timestamp step becomes
`find /out -exec touch -h -d "@${SOURCE_DATE_EPOCH}" {} +`. The `-h` flag
pins the symlinks' own mtimes for R9 reproducibility.

The runtime stage runs `COPY --link --from=build /out/bin/ /`. A directory
copy keeps the symlinks as symlinks. `/migrations` keeps its own marked
`COPY`. `ENTRYPOINT ["/service"]`, `USER`, `STOPSIGNAL`, `EXPOSE`, labels, and
the base digest are unchanged.

Exec of `/worker` resolves the symlink in the kernel, and the runtime still
passes `argv[0]="/worker"`. Docker and OCI `--entrypoint`, a Kubernetes
`command`, and Railway `preDeployCommand = ["/migrate"]` all use exec form.
The distroless image has no shell to wrap them.

### M4. Local commands (R4)

`make/template.mk` builds and runs every role from `./cmd/entrypoint` with
`-X main.pinnedRole=<role>` added to the existing ldflags. Output paths, the
`not applicable: ...` probes on `cmd/<role>` directories, and PGO handling for
`build` and `build-pgo` do not change. Process integration tests build
`./cmd/entrypoint` unpinned, writing to `<tmp>/<role-name>`. They therefore
exercise the real `argv[0]` dispatch.

## Alternatives and dispositions

| Choice | Selected | Rejected | Decisive constraint | Cost accepted | Reverse when |
| --- | --- | --- | --- | --- | --- |
| Entry path to shared content | Relative symlinks in a directory `COPY` | Per-file `COPY` (it follows the source link and duplicates the bytes); hardlinks (no extra benefit, and whether BuildKit `COPY --link` keeps them is less certain) | R1 single payload; R9 reproducibility | `ls -l /` shows links; A1 already excludes deployments that hash or copy a role's file | An image-count proof finds more than one payload, or a platform is found resolving the link before exec |
| Role packages | M2: `cmd/<role>` becomes a library; a new `cmd/entrypoint` main | Moving every role under `cmd/entrypoint/internal/<role>` or `cmd/internal/<role>`; keeping `cmd/service` as the dispatcher main | Template-owned `integration-init.sh`, `integration-record-check.sh`, depguard globs, init-module stripping, and the `go list -deps ./cmd/service` guard all address `cmd/<role>/...` today. Moving them would break R7 on unmigrated repositories and force compatibility branches. A dispatcher in `cmd/service` would put every role into the service graph and split each role's entry between two owners | Five `cmd/<role>` packages each export `Main` and are importable by path. This module is an application, not a library. Composition stays `internal` | The repository accepts publishing reusable API from `cmd/`, or an external importer of `cmd/<role>` appears |
| R4 local role | Link-time `pinnedRole` | Per-role main packages (six mains, and they cannot coexist with importable role packages); build tags per role; suffix matching (forbidden by R3) | R4 output names cannot match R3 | One link-time string used only by local artifacts | The Dockerfile or any shipped build path needs a pin |
| R7 unmigrated layout | Fail-closed precondition in role build/run Make targets; every other synced tool is layout-neutral (table below) | Old-layout compatibility branches in Make | AGENTS.md rejects compatibility paths without an accepted requirement; R7 allows failure | Unmigrated repositories cannot `make build` or `make run*` until migrated | A derived-repository owner accepts a requirement to keep building the old layout |

## Material flows

1. **Container start through an entry path.** The platform execs the path,
   for example `/worker`. The kernel follows `/worker` to `/service`, and
   `argv[0]` stays `/worker`. The dispatcher selects `worker`, and
   `worker.Main(args)` runs today's `bootstrap.Run(args, nil)`. Its exit code
   reaches `os.Exit`. Failure, cancellation, and drain stay owned by the role
   bootstrap. The dispatcher adds no retry or state.
2. **Unknown name.** For a misnamed link, a copied file, or an empty `argv`,
   the dispatcher rejects with exit 2 and one stderr line. No role code runs.
   The platform's restart policy owns what happens next. No default role is
   used.
3. **Generation.** `init-module.sh` strips marked imports and table entries in
   `cmd/entrypoint/main.go` and marked `ln` lines in the Dockerfile, then
   removes the role directories as today. The result links and links only the
   selected roles.
4. **Local build and run.** A `make` target passes the pinned role, and the
   artifact runs that role whatever its name. If the pin names an unselected
   role, the artifact exits 2.

No persisted state, wire contract, or configuration crosses a new boundary.

## Enforcement of critical invariants

| Invariant | Mechanism and boundary | Paths that could bypass it, and how they are closed |
| --- | --- | --- |
| A role runs no other role's composition (R2) | The dispatcher calls exactly one `Main`. `internal` keeps each bootstrap private | A role package importing another role package. Closed in the template by a new guard in `scripts/ci/init-module-contract-check.sh`, next to the existing `go list -deps ./cmd/service` use: for each present `cmd/<role>`, its dependency list contains no other role package and no `cmd/entrypoint`, and `./cmd/service` excludes `internal/infra/postgresmigrate`. **Activation boundary:** the guard runs only when `changed-surfaces.sh` marks `module_initializer`, and only in the template, because the contract check is source-only. A PR that only edits `cmd/<role>/role.go` to import another role is not gated by it. Neither are derived repositories. Review of that diff is the remaining gate, and the reviewer confirmed today's per-role `go list -deps` sets are clean. Accepted as bounded: a cross-role import is a visible import line, and importing a role does not run its composition. **Initializer exposure:** after the merge, every linked package's `init` and package-level variable initializers run in every role. Today the only `init` functions in `cmd/**` assign builders (`cmd/jobs-worker/builder_webhooks.go` suppresses `gochecknoinits` to do so), and telemetry registers metrics only at runtime. A new side-effecting initializer would run in every role, and no gate catches that beyond `gochecknoinits`, which a `nolint` can bypass. The migration guide (R8) and the `cmd/<role>` package comments (R6 seams) must state that role code keeps I/O, registration, and logging out of initializers. The OTel resource uses explicit attributes only (`internal/infra/telemetry/resource.go`), so no executable-name detector reports `/service` for every role |
| No default role (R3) | Exact table match, else exit 2 | A pinned image build. Closed statically because the Dockerfile ldflags are the only shipped build owner and contain no `-X main.pinnedRole`. Falsified at runtime by the R2 role checks: the template `/worker` must fail with `natsjs.ErrRejected` and exit 1, and `/migrate` must validate. A pinned image runs one role for every path and fails these checks |
| Unselected role not linked (R1) | Profile-marked imports in the dispatcher, plus directory removal | Linking: an unmarked import of a removed role fails compilation of every generated profile (`go test -vet=off -run '^$' ./...` in the init contract check), so R1 linking is gated. Residue: `init-module.sh` `assert_no_profile_markers` checks only for leftover markers. No current gate finds an unselected role's name in an unmarked comment or string in `cmd/entrypoint/main.go`, or an unmarked Dockerfile `ln` line. (The old residue grep went with `scripts/ci/template-init-check.sh`, deleted in 949e92705.) An unmarked `ln` line does not fail the build (`ln -s` succeeds for any name); the final-validation `rg` sweep is what catches it. Comment and string residue for R5 is closed by the RS1 placement rule: no role name outside markers except `service`. Implementation final review owns it, and proof is an `rg` over each generated profile's `cmd/entrypoint` and Dockerfile at final validation. No new gate is added |
| One payload (R1) | A directory `COPY` of the file and its links | A per-file `COPY` of a link. Closed in the Dockerfile shape, and proved by counting payloads in the image layers |
| Role dependency graphs are unchanged (R9) | Each `role.go` keeps its `main.go` imports exactly | Guarded by the contract-check loop above |
| PGO fingerprint covers the service role (R9) | `scripts/dev/benchmark.sh` `pgo_candidate_fingerprint` adds `cmd/entrypoint` to its path list. `cmd/service` and `cmd/internal` are already listed, and other roles' directories are not added | The changed file set invalidates existing manifests, which fail closed (accepted by the spec) |

## R7: synced tooling on an unmigrated layout

| Template-owned surface | Layout dependence after this change | Unmigrated behavior |
| --- | --- | --- |
| `make/template.mk` `run`, `run-*`, `build`, `build-*`, `build-pgo` | Builds `./cmd/entrypoint` | If the role directory is absent, prints `not applicable` as today. If it is present and `cmd/entrypoint/main.go` is absent, exits 1 with a message naming `docs/runtime-executable-migration.md`. This is a permanent precondition, not a compatibility branch |
| `make/template.mk` integration, race, and other targets | Unchanged package paths (`./cmd/worker/internal/bootstrap`, `./test/...`) | Works |
| `scripts/integration-init.sh` | `cmd/service/internal/bootstrap` is unchanged | Works |
| `scripts/ci/integration-record-check.sh` and its `integration-record-*.go` helpers | `cmd/service/internal/bootstrap/startup_*.go` is unchanged | Works; its discovery glob still finds files |
| `scripts/ci/changed-surfaces.sh` | Adds `cmd/entrypoint/*` and `cmd/*/role.go` to `process_integration` and keeps `cmd/*/main.go`, which on the new layout is redundant with `cmd/entrypoint/*` and is kept only so an unmigrated derived repository still routes role `main.go` edits to process integration (R7); remove it once the template no longer supports syncing into the per-role `main.go` layout | Works; the old patterns still match |
| `scripts/dev/benchmark.sh` | Adds `cmd/entrypoint` to the fingerprint list; missing paths are already skipped | Works; the fingerprint is unchanged on the old layout |
| `scripts/ci/verify.sh` | Calls `make` targets only | Inherits the Make behavior |
| `scripts/ci/runtime-image-*.sh`, `scripts/ci/migration-*.sh` | Build the repository-owned Dockerfile, use the default `ENTRYPOINT`, and run `--entrypoint /migrate` | Work for the old image |

## Repository Architecture changes

Owner: this design ([Repository Architecture](../../../docs/repo-architecture.md)
trigger: a composition root or executable changes). The meaning of the rules
does not change. Only the vocabulary moves from binary or executable to role,
process, and entry path:

- `docs/repo-architecture.md`: invariant 1 and the source table now say
  `cmd/<role>` composition root and `cmd/<role>/internal/bootstrap/*` (that
  role's composition and lifecycle). A new row says `cmd/entrypoint` selects
  one role per process from its entry path and owns nothing else.
- `docs/architecture/boundaries.md`: the `cmd/service/main.go` rows and the
  dependency graph become `cmd/entrypoint -> cmd/<role> (Main) ->
  cmd/<role>/internal/bootstrap`. The text adds that only `cmd/entrypoint`
  imports role packages.
- `docs/architecture/runtime-lifecycle.md` step 1,
  `docs/architecture/async.md` (separate worker binary becomes separate worker
  role or process), `docs/architecture/integration.md`,
  `docs/architecture/persistence.md`, `docs/first-production-feature.md`
  line 253, `docs/project-structure-and-module-organization.md`, and README's
  tree: reword in place. Profile-marked text keeps its markers.
- New `docs/runtime-executable-migration.md` carries R8. It is listed in
  `template-owned.paths` and linked from `docs/template-sync.md`. It has no
  profile markers and no service-specific content, and it names only `make`
  targets defined in `make/template.mk`.
- Paths relocated by this change, for the R6 sweep:
  `cmd/<role>/main.go` becomes `cmd/<role>/role.go`; commands
  `go build|run ./cmd/<role>` become `./cmd/entrypoint`; and the Dockerfile
  per-role `COPY` lines are replaced. Every other path under `cmd/` is
  unchanged.

## Release closure

| Node or edge | Current to target | Mixed-version obligation | Rollback |
| --- | --- | --- | --- |
| Image, and each deployment that execs `/service`, `/worker`, `/jobs-worker`, `/outbox-relay`, or the `/migrate` pre-deploy | Five files become one file plus links. Paths, arguments, environment, ports, and signals are unchanged | Old and new images may overlap during Railway `overlapSeconds`. They share no new state, and the processes are behavior-identical | Redeploy the baseline image |

Gates, all at implementation final validation: build the image, then
`make runtime-image-check`, then `make migration-validate`. A role exec check
covers the template `/worker` and a `JOBS=postgres WEBHOOKS=none
INBOUND_WEBHOOKS=none` `/jobs-worker`. Then measure M1 through M3. There is no
deployment-graph change and no `rollout.md`.

## Proof boundaries

The spec's Proof expectations stand. This design adds three proof surfaces:
the role-isolation `go list` guard in `init-module-contract-check.sh`, which is
template-only and active under `module_initializer`; the R2 role exec checks,
which also falsify a pinned image; an `rg` residue sweep over generated
profiles' `cmd/entrypoint/main.go` and Dockerfile; and process integration
tests that build to role-named outputs.

## Reopen conditions

- Specification: a deployment starts a role other than through an R2 exec
  entry path (A1), or M1 exceeds 50 MB.
- This design: a platform resolves the symlink before exec, a payload count
  finds more than one copy, an importer of a `cmd/<role>` package appears, or
  a shipped build needs `pinnedRole`.
- Research: BuildKit `COPY --link` of a directory does not preserve the
  relative links or their pinned mtimes.

## Review dispositions

Technical Design Review returned `CONCERNS`. Movement is permitted. Each item
is repaired in place above:

- F1: `go version -m` hides `-ldflags` under `-trimpath`. The "no pin in the
  image" proof is restated as static Dockerfile ownership plus the R2 role
  exec falsifier (M1 note, enforcement row, proof boundaries).
- F2: the cited residue grep no longer exists. The R1 linking gate is the
  compile check; R5 comment and string residue is closed by the RS1
  placement rule plus a final-validation `rg` sweep. No gate is added.
- F3: the isolation guard's activation boundary is template-only and runs
  under `module_initializer`. Edits to `cmd/<role>/role.go` alone and derived
  repositories rely on review. The initializer exposure is recorded, and R6
  package comments plus the R8 guide must give notice of it.
- F4: the RS1 reuse record is added in ownership-map.md.

Verified by the reviewer: symlinks with `argv[0]` dispatch, and
`COPY --link` of the directory preserving the links and a reproducible digest
(buildx 0.37.2). `-o /out/bin/service` creates the directory, and `-X` works
on package `main`.
