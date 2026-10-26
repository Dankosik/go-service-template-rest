# Single runtime executable

status: ready

Requester meaning: [intent.md](intent.md). Baseline: commit `aeda2664f` plus
the unrelated uncommitted edits present when this spec was written. Those
edits are a base-image bump and `COPY --link` in `build/docker/Dockerfile`,
the context-directory fix in `scripts/ci/runtime-image-build.sh`, a `go.mod`
and `go.sum` bump, and `internal/problem/problem_test.go`. This change
preserves them and does not revert them. The baseline has five role
executables, and every behavior below marked "unchanged" means unchanged
relative to that baseline.

## Terms

- **Role**: one of the five process kinds the image can start: `service`,
  `worker`, `jobs-worker`, `outbox-relay`, `migrate`.
- **Selected role**: a role whose profile the generated service selected.
  `service` is always selected. The profiles are `worker` with
  `MESSAGING=nats-jetstream`, `jobs-worker` with `JOBS=postgres`,
  `outbox-relay` with `OUTBOX=postgres`, and `migrate` with
  `DATABASE=postgres`. These are the current `init-module.sh` and Dockerfile
  marker pairings. The template checkout itself selects all five.
- **Entry path**: the absolute image path an operator executes to start a role,
  for example `/worker`.
- **Shared executable**: the one executable file whose content all of the
  image's entry paths resolve to.

## Behavior delta

### R1. One executable in the image

The image contains exactly one executable file's content. Every entry path of
a selected role resolves to that same content, so no second copy of the
program bytes exists. The image may still contain other files, such as
`/migrations/` and the base image's own files.

A generated service's shared executable contains only its selected roles. An
unselected role's code and its exclusive dependencies are not linked. For
example, a service without `MESSAGING` links no NATS client module, and one
without `JOBS` and `OUTBOX` links no River module. Falsifier: `go version -m`
on the extracted executable lists the excluded module.

### R2. Entry paths are unchanged for existing deployments

| Entry path | Present when | Starts |
| --- | --- | --- |
| `/service` (also the image `ENTRYPOINT`) | always | the HTTP/gRPC service |
| `/worker` | messaging selected | the NATS JetStream worker |
| `/jobs-worker` | jobs selected | the River jobs worker |
| `/outbox-relay` | outbox selected | the outbox relay |
| `/migrate` | database selected | the migration job |

Invoking an entry path with exec form behaves exactly as invoking that role's
baseline executable with the same arguments, environment, and signals.
Exec form means Docker or OCI `ENTRYPOINT`/`--entrypoint`, a Kubernetes
`command`, or Railway `preDeployCommand = ["/migrate"]`. "Exactly as" covers:

- Arguments: only `-config` and repeatable `-config-overlay`. Positional
  arguments, unknown flags, and `-h` are rejected as today through
  `config.ParseLoadOptions`. No new flag, subcommand, or positional form is
  accepted by any role.
- Configuration: the same file, overlay, and `APP__*` environment sources and
  the same validation.
- Output: the same log records, fields, and streams. Values that encode
  source locations, such as panic stacks and natsjs `handler_frames`, may name
  moved functions and files. `service.name`,
  `service.version`, and `deployment.environment.name` come
  from configuration and ldflags stamping as today; the build commit reaches
  the same telemetry resource attribute as today. On failure, stderr carries
  the same error text, except `migrate`, which still prints no error text and
  logs its classified terminal record to stdout.
- Exit status: 0 on success or graceful stop, 1 on failure, per role as today.
- Signals and drain: SIGINT and SIGTERM handling, readiness withdrawal,
  drain, and stop-grace behavior per role are as today. This includes `/service`
  exiting 0 within `docker stop --time 45`, the jobs-worker stop-grace
  contract, and `migrate`'s bounded timeout.
- Fail-closed defaults: the template's generic `/worker`, which has no
  registered `HandlerBuilder`, still fails before config load or broker I/O
  with the same `natsjs.ErrRejected` message and exit 1. A `/jobs-worker`
  with no registered `WorkersBuilder` still fails before database I/O. That
  case exists in a generated service with `JOBS=postgres`, `WEBHOOKS=none`,
  and `INBOUND_WEBHOOKS=none`. The template's own jobs worker installs the
  webhook builder.
- Externally observable surfaces: listeners, ports, health endpoints, exported
  metric names, and outbound connections. A role exposes none that belong to
  another role, because starting one role runs none of another role's
  composition, configuration checks, I/O, metric registration, or logging.

An entry path of an unselected role is absent from the image, exactly as
today. Starting it fails in the container runtime before any program code runs.

Existing deployment configuration needs no edit: `railway.toml`, the image
`ENTRYPOINT`, and any deployment that starts one of the paths above.

### R3. Unknown invocation name

The shared executable selects its role from the final path element of
`argv[0]`. For the exec-form entry paths in R2, that element is the entry path's
name. It compares the element against its selected role names. The match is
exact and case-sensitive, with no prefix, suffix, or extension stripping. If
no selected role matches, including an empty `argv[0]`, an unselected role's
name, or the shared executable's own file name when that differs from every
role name, the process:

- writes exactly one line to stderr containing the invoked name and the
  selected role names in the R2 table order;
- writes nothing to stdout;
- exits with status 2;
- does so before any action the program itself starts. It reads no
  configuration file and no `APP__*` or role environment, installs no signal
  handler, and does no network or file I/O other than writing that stderr line.
  Go runtime startup and linked package initializers are outside this rule.

No role is ever selected by default for an unrecognized name. A misnamed worker
deployment must not start the HTTP service.

### R4. Local developer commands keep their meaning

The `make` targets `run`, `run-worker`, `run-jobs-worker`, `run-outbox-relay`,
`build`, `build-worker`, `build-jobs-worker`, `build-outbox-relay`, and
`build-pgo` keep their names and start or build the same role as today. They
also keep their `not applicable: ...` output and success status when that role
is not selected. Each `build*` target writes to its current output path
(`bin/<service-name>`, `bin/<service-name>-worker`, and so on, and
`bin/<service-name>-pgo`). Started with only `-config` and `-config-overlay`
arguments, that artifact runs its role. R3's name matching therefore cannot
apply to these artifacts as named. Every other repository command or script
that builds or runs a role, including the benchmark and PGO harness, keeps
working with the same meaning.

### R5. Profiles and generation

`scripts/init-module.sh` still produces, for every supported profile
combination, a service whose image and shared executable contain exactly its
selected roles (R1, R2). It leaves no residue of unselected roles in shipped
source, comments, or the Dockerfile. `scripts/integration-init.sh` still adds
an integration to the service role's composition root of a generated service,
and that integration is active only in the `service` role.

### R6. Customization seams for generated services

A generated service can still make every role-local customization it can make
under `cmd/` today, and each still lives in one role-local place. These
customizations include at least:

- the service role's feature composition in its composition root
  (`cmd/service/internal/bootstrap`). That covers handler wiring, domain error
  mapping, and gRPC runtime construction, as
  [docs/first-production-feature.md](../../docs/first-production-feature.md)
  directs adopters;
- service-role integrations added by `integration-init.sh`;
- the messaging worker's `HandlerBuilder`;
- the jobs worker's `WorkersBuilder`, including the profile-installed webhook
  builders, inbound webhook handler registration (`registerInboundWebhookHandlers`
  and `inbound_webhook_bindings.go`), and the build-tag-substituted test
  builders used by process integration tests.

Their file locations may change; that is the accepted conflict cost. A
customization made for one role does not change another role's behavior. The same change updates
every tracked Markdown file outside `specs/**` and every source comment that
uses a path or command relocated by this change as current guidance, so the current
guidance names the new location. Two kinds of reference are exempt: the
migration guide's references to old locations (R8), and existing historical or
retired-path mentions such as `/webhook-worker` in
`docs/railway-deployment-profile.md`.

### R7. Template-owned synced tooling in unmigrated derived repositories

Template sync mirrors every path in `template-owned.paths` that reads the
`cmd/` or Dockerfile layout. Today these are at least `make/template.mk`,
`scripts/integration-init.sh`, `scripts/dev/benchmark.sh`,
`scripts/ci/changed-surfaces.sh`, `scripts/ci/verify.sh`,
`scripts/ci/integration-record-check.sh` and its `integration-record-*.go`
helpers, `scripts/ci/runtime-image-*.sh`, and `scripts/ci/migration-*.sh`.
Derived repositories still have repository-owned five-binary `cmd/` and
Dockerfile layouts. Run after sync in such an unmigrated
repository, each of these surfaces either keeps working for the roles present
or fails with a non-zero status and a message naming the migration guide. It
never reports a role that is present in the old layout as `not applicable`,
never skips that role's tests or checks, and never builds or checks the wrong
role. That includes discovery globs that would silently find nothing, such as
the integration-record check. A branch kept solely so the tooling works on the old layout is a
compatibility path. It needs an owner, observable activation, and a removal
condition under [AGENTS.md](../../AGENTS.md#engineering).

### R8. Migration guide for derived services

The template ships one migration guide, linked from
[docs/template-sync.md](../../docs/template-sync.md). The guide is listed in
`template-owned.paths`, so template sync mirrors it into every derived
repository and the link and R7 messages resolve there. It therefore meets the
manifest admission rule: no repository-specific content and no profile
markers. A derived maintainer can
follow it without reading this spec. It states:

- that deployment configuration and entry paths do not change (R2), and why
  the image shrinks;
- for each customization seam in R6, its old and new location and how to port
  an existing customization of it. That includes the service composition root
  (handler wiring, domain error mapping, gRPC runtime), integration-init
  additions, the messaging `HandlerBuilder`, the jobs `WorkersBuilder`, and
  inbound webhook handler registration;
- the Dockerfile and `cmd/` changes the derived repository must make, and that
  synced tooling assumes them (R7 behavior on the old layout);
- that an accepted PGO manifest may need re-acceptance (R9);
- how to verify the migrated image: role entry paths, the size measurement in
  Success, `make runtime-image-check`, and migration validation.

### R9. Deliberately unchanged

- Base image (distroless static, digest-pinned), `USER nonroot:nonroot`,
  `STOPSIGNAL SIGTERM`, `EXPOSE 8080`, OCI labels, `/migrations/` contents and
  path, and running under a read-only rootfs with all capabilities dropped.
- Reproducibility: identical build inputs still yield the same local image
  digest under `SOURCE_DATE_EPOCH`, including any link entries.
- Version stamping: `APP_VERSION`, `VCS_REF`, and `RAILWAY_GIT_COMMIT_SHA`
  resolve to the same version and commit values in every role as they do today.
- PGO policy: `off` by default; `auto` or an empty value is rejected; an
  explicit profile is verified against its manifest before the build and is
  applied to the shipped executable. The accepted-build fingerprint must still
  cover every source compiled into the service role, so a move must not narrow
  it. A fingerprint change caused by this move invalidates existing manifests,
  which fail closed and must be re-accepted. No manifest is silently honored.
- Railway policy: `railway.toml` is byte-for-byte unchanged. Every source
  compiled into the shipped executable stays inside its `watchPatterns` and
  inside the Docker build context that `.dockerignore` leaves.
- Role dependency boundaries: each role's own composition reaches the same
  dependency packages as today. In particular, the service role's composition does not
  reach `internal/infra/postgresmigrate`, so application startup still never
  migrates. Only the role-selection entry links every selected role.
  Repository Architecture owns rewording the current rules to say role,
  process, and entry path instead of separate binary or executable. Those
  rules are in `docs/first-production-feature.md`,
  `docs/architecture/boundaries.md`, `docs/architecture/persistence.md`,
  `docs/architecture/async.md`, and `docs/repo-architecture.md`. Repository
  Architecture also owns the guard that measures the service dependency graph
  (`go list -deps ./cmd/service` in `scripts/ci/init-module-contract-check.sh`).
  The rule's meaning does not change.
- Container-security and other existing CI or release gates keep their
  policy.

## Non-goals

- Changing any role's configuration schema, environment variables, ports,
  health semantics, telemetry, or deployment topology. Roles still run as
  separate processes.
- A subcommand or flag interface for role selection.
- Executable compression or packing (for example UPX), a different base image,
  or removal of any role.
- A permanent CI gate on image size. Reopen if a later change measurably
  regresses the size.
- Treating the process command name (`/proc/<pid>/comm`) as a contract.

## Success

Measured on linux/amd64 with `PGO_PROFILE=off` and default build arguments,
using `docker image inspect -f '{{.Size}}'` (bytes; MB = 10^6 bytes):

| Configuration | Requirement |
| --- | --- |
| M1: all five roles, from `docker build -f build/docker/Dockerfile .` on the template checkout | at most 50 MB, and at most 40% of the same measurement on the baseline |
| M2: `make runtime-image-build` output (the `DATABASE=postgres` generated service) | not larger than its baseline measurement |
| M3: a generated service that selects no optional role | at most its baseline measurement + 1 MB |

The gzip size of M1 (`docker save` piped through `gzip -9`) is reported, not
gated. The 50 MB bound gives about 10% headroom over the about 45 MB estimate.
If M1 measures above it, the result is reported with a per-file breakdown, and
Specification reopens to settle the bound. The threshold is not raised
silently.

The whole outcome also requires R1 through R9 to hold. All requirements could
pass while the user outcome still fails only if a real deployment invokes a
role some way other than through an R2 entry path. Assumption A1 bounds that.

## Proof expectations

Implementation selects commands and tests. Each claim needs its nearest
falsifier:

- R1 and Success: build and measure M1, M2, and M3 against their baselines;
  count distinct executable payloads in the image; run `go version -m` on M3
  and on a no-messaging configuration.
- R2: `make runtime-image-check` (default `ENTRYPOINT`, readiness, version,
  SIGTERM exit 0) and migration validation (`--entrypoint /migrate`) on the
  new image. Invoke the template's `/worker` in the image, and `/jobs-worker`
  in the `JOBS=postgres WEBHOOKS=none INBOUND_WEBHOOKS=none` generated
  service. Compare exit status and stderr with the baseline fail-closed
  behavior.
  Run the existing process integration tests for the roles.
- R3: unit-level proof over the name-selection rule, covering the empty name,
  unselected role names, near-miss names, and the absence of side effects
  before rejection.
- R4, R5, and R7: the existing make and template-init checks across profile
  combinations, plus one run of the synced tooling against an unmigrated layout.
- R6 and R9 boundaries: no path relocated by this change remains as current
  guidance in any tracked Markdown file outside `specs/**` or in any source
  comment, with the R6 exemptions;
  the service role's package graph excludes `postgresmigrate`.
- R9: rebuild identical inputs to the same digest; `PGO_PROFILE=auto` is
  rejected and a stale manifest fails closed.

## Assumptions

- **A1**: Deployments start roles only through R2 entry paths in exec form.
  None depends on a role's baseline file being a distinct regular file, its
  byte size, or a per-file hash. Reopen if a derived service or deployment is
  found copying, hashing, or exec-ing a role through another path.
- **A2**: Per-role startup time and resident memory are not part of this
  contract. R2 forbids running other roles' composition. Linked package
  initializers, such as protobuf and gRPC registration, still run eagerly in
  every role, and the owner accepted that cost for image size. Reopen if a deployment observes a material
  regression. Measure only through the repository's DigitalOcean benchmark
  route.
- **A3**: The image targets Linux (`TARGETOS` defaults to `linux`). Local
  developer builds run on macOS or Linux, not Windows. Reopen if Windows local
  builds become supported, because R3's exact-name rule ignores `.exe`.

## Rollout

No persisted state, wire contract, or deployment configuration changes. Old
and new images can overlap during a Railway deploy (`overlapSeconds`), and
rollback to a baseline image works as before. No `rollout.md` is needed.
Derived repositories adopt the change through R8 on their own schedule.
