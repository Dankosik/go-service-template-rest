# Single runtime executable: ownership map

status: ready

Mechanism: [system.md](system.md). Format: [Ownership Map
V1](../../../docs/spec-first-workflow/interfaces/ownership-map-v1.md). Test
cases and files are the implementer's choice. Proof owners below name the
package or check where proof lives.

## Responsibilities

| responsibility | affected path | current evidence | semantic owner | exact package/file action | dependency/composition/generated boundary | cleanup | proof owner | reopen condition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| RS1 Role selection (R3, R4 pin) | `cmd/entrypoint/main.go` | Five `main` packages, each fixed to one role | `cmd/entrypoint` | Add package `main`: an unexported `pinnedRole string` set by `-X`, an unexported ordered role table (name to `func([]string) int`) with profile-marked optional entries and imports, `main`, and one unexported pure selection function. Its rejection line and exit 2 follow system.md M1 | Imports only the stdlib and `cmd/<role>` packages. Imports no `internal/**` or `cmd/internal/**`. It is the only importer of role packages. No role names appear outside markers except `service` | Delete the five per-role `main` functions (RS2) | Unit tests in `cmd/entrypoint` (R3 cases); process integration tests built to role-named outputs | A role must be selected by something other than the `argv[0]` basename or pin |
| RS2 Role process entry (args to exit status, terminal stderr/stdout) | `cmd/{service,worker,jobs-worker,outbox-relay,migrate}/main.go` to `role.go` | Each `main.go` body | Each `cmd/<role>` package | Rename the file and change the package clause to `service`, `worker`, `jobsworker`, `outboxrelay`, or `migrate`. Replace `main()` with exported `func Main(args []string) int` that returns 1 where `os.Exit(1)` was called. Keep the body, the imports, and the stream writes. Add a package comment saying the package is a role entry imported only by `cmd/entrypoint` | Same imports as today's `main.go`. Never imports another role or `cmd/entrypoint` | No `main.go` remains in role directories | Existing role tests; process integration tests; the isolation guard (RS7) | A role needs a dispatcher-supplied input |
| RS3 Role-local customization seams (R6) | `cmd/worker/role.go` (nil `HandlerBuilder`); `cmd/jobs-worker/builder.go`, `builder_testworker.go`, `builder_inbound_testworker.go`, `builder_webhooks.go`, `inbound_webhook_bindings.go`; `cmd/migrate/run.go` | Package `main` files in place | The same role package | Package clause only, `main` to the role package name. Update the "binary" wording in their comments to "role". Build tags and profile markers are unchanged | Unchanged | none | Existing `builder_webhooks*_test.go` and `run_test.go` (package clause only) | A seam must be shared across roles |
| RS4 Shipped image build (R1, R9) | `build/docker/Dockerfile` | Five `build_packages`, five `COPY` lines | Dockerfile | One `go build -o /out/bin/service ./cmd/entrypoint`, marked `ln -s service /out/bin/<role>` lines, `touch -h`, and one `COPY --link --from=build /out/bin/ /`. Keep the marked `/migrations` `COPY` and preserve the uncommitted baseline edits. It must never set `pinnedRole` | Profile markers are stripped by `init-module.sh` | Remove the `build_packages` assembly and the per-role `COPY` lines | `make runtime-image-build`/`runtime-image-check`, `migration-validate`, M1 to M3 measurements, payload count, `go version -m` for module exclusion (R1 only; it cannot show `-ldflags` under `-trimpath`), and the R2 role exec checks, which also falsify a pinned image | system.md M3 reopen |
| RS5 Local role build/run (R4, R7) | `make/template.mk` | `*_CMD := ./cmd/<role>` plus `go build`/`go run` | `make/template.mk` | Keep `*_CMD` as directory probes. Build and run `./cmd/entrypoint` with `-X main.pinnedRole=<role>` appended to the current ldflags. Add one precondition used by `run`, `run-*`, `build`, `build-*` (and so `build-pgo`): when the role directory exists and `cmd/entrypoint/main.go` does not, exit 1 naming `docs/runtime-executable-migration.md` | Template-owned, so no profile markers | Remove the per-role `go build ./cmd/<role>` commands | `init-module-contract-check` (it already runs `make run-worker`); one run against an unmigrated layout | system.md R7 reopen |
| RS6 Synced CI and dev tooling mapping (R7, R9 PGO) | `scripts/ci/changed-surfaces.sh`, `scripts/dev/benchmark.sh` | Path patterns and fingerprint list | Each script | changed-surfaces: add `cmd/entrypoint/*` and `cmd/*/role.go` to the `process_integration` case and keep `cmd/*/main.go`. benchmark.sh: add `cmd/entrypoint` to the `pgo_candidate_fingerprint` path list | Template-owned | none | `verify.sh`'s existing plan self-tests; the stale-manifest fail-closed check | none |
| RS7 Role isolation and service graph guard (R2, R9) | `scripts/ci/init-module-contract-check.sh` | `go list -deps ./cmd/service` only for authn profiles; postgresmigrate exclusion is implicit today | Repository Architecture, via this script | For every generated profile root, and for each present `cmd/<role>`, assert its `go list -deps` holds no other `cmd/<role>` package and no `cmd/entrypoint`. Assert `./cmd/service` deps exclude `internal/infra/postgresmigrate` | Source-only template check. It is active only when `changed-surfaces.sh` marks `module_initializer`; role-only edits and derived repositories rely on review (system.md F3) | none | The init module contract check over generated profiles | A role legitimately needs another role's package |
| RS8 Process integration build commands | `test/postgres_jobs_process_integration_test.go`, `test/inbound_webhook_process_integration_test.go`, `test/nats_messaging_process_integration_test.go`, `test/grpc_process_integration_test.go` | `go build ... ./cmd/<role>` | `test/` | Build `./cmd/entrypoint` with the same tags to `<tmp>/<role-name>` and exec that path | Integration tag, external package | none | Those tests | none |
| RS9 Architecture vocabulary, R6 doc sweep, and R8 guide | docs listed in system.md "Repository Architecture changes"; `docs/runtime-executable-migration.md`; `docs/template-sync.md`; `template-owned.paths`; source comments | Binary wording; no guide | Repository Architecture (docs); R8 guide | Reword in place. Add the guide with spec R8 contents and admit it to `template-owned.paths` | The guide is template-owned: no markers, no service-specific content | No current-guidance mention of relocated paths, except R6 exemptions | `rg` over tracked Markdown and comments; `docs-contract-check`; `template-owned-purity-check` | A doc rule's meaning, not just its wording, must change |

`scripts/init-module.sh` needs no planned change. It discovers markers in
every file and already removes the role directories. `template-init-check`
proves this. If it fails, the fix stays within RS1 or RS4 marker placement
before `init-module.sh` is edited.

RS1 reuse record (a non-mechanical implementation source):

- Reuse rung: the standard library (`os.Args`, `os.Exit`, `fmt.Fprintln` to
  `os.Stderr`, and `path.Base` or `strings.LastIndexByte` for the final
  element).
- Authority and version: the module's Go version in `go.mod`.
- Strongest rejected source: a multi-call or CLI library, for example cobra
  or urfave/cli. It would add subcommand and flag semantics that R3 and the
  non-goals forbid, plus a dependency, for a single table lookup.
- Parity proof: `cmd/entrypoint` unit tests over the R3 cases, and process
  integration tests built to role-named outputs.
- Upgrade condition: role selection needs more than an exact name lookup.
  The spec rules that out today, so this reopens Specification first.

## Files

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `cmd/entrypoint/main.go` (new) | RS1 | The image's only executable must choose one role per process | unexported `pinnedRole`, the role table, the selection function, and `main` | Process start, then one `<role>.Main`, then `os.Exit` | Owns only the unknown-name rejection (exit 2). Every other status comes from the role | stdlib (`fmt`, `os`, `strings` or `path`); `cmd/<role>` packages | Config, environment, flags, signals, logging, telemetry, or any role behavior |
| `cmd/service/role.go` (from `main.go`) | RS2 | The service role's process entry | `package service`; `func Main(args []string) int` | Called by the entrypoint; calls `bootstrap.Run` | stderr `"%v\n"`, then 1 | `cmd/service/internal/bootstrap`, stdlib | Role selection; importing other roles |
| `cmd/worker/role.go` (from `main.go`) | RS2, RS3 | The worker role's entry and its `HandlerBuilder` seam | `package worker`; `Main` | Calls `bootstrap.Run(args, nil)` | stderr line, then 1; fails closed on a nil builder as today | `cmd/worker/internal/bootstrap`, stdlib | Same as above |
| `cmd/jobs-worker/role.go` (from `main.go`) | RS2 | The jobs role's entry | `package jobsworker`; `Main` | Calls `bootstrap.Run(args, buildWorkers)` | stderr line, then 1 | `cmd/jobs-worker/internal/bootstrap`, stdlib | Same as above |
| `cmd/jobs-worker/{builder,builder_testworker,builder_inbound_testworker,builder_webhooks,inbound_webhook_bindings}.go` and `builder_webhooks*_test.go` | RS3 | The existing `WorkersBuilder` seam and its tests | Package clause becomes `jobsworker`; declarations unchanged and unexported | Unchanged | Unchanged | Unchanged | Unchanged |
| `cmd/outbox-relay/role.go` (from `main.go`) | RS2 | The outbox role's entry | `package outboxrelay`; `Main` | Calls `bootstrap.Run(args)` | stderr line, then 1 | `cmd/outbox-relay/internal/bootstrap`, stdlib | Same as above |
| `cmd/migrate/role.go` (from `main.go`), `run.go`, `run_test.go` | RS2, RS3 | The migrate role's entry and run | `package migrate`; `Main`; `run` and the rest stay unexported | Calls `run(args, os.Stdout)` | Prints no error text; returns 1 | Unchanged `run.go` imports | Same as above |
| `test/*_process_integration_test.go` (the four in RS8) | RS8 | Real-process proof | unchanged | Build the entrypoint to a role name, then exec | unchanged | unchanged | unchanged |

## Reused Capabilities

| capability | new consumer | existing owner | reuse or shared owner | dependency rule forbidding direct reuse | extraction dependency |
| --- | --- | --- | --- | --- | --- |
| Per-role process entry (arguments, terminal output, exit status) | `cmd/entrypoint` | Each `cmd/<role>/main.go` body | Reuse: the same body becomes `cmd/<role>.Main` (RS2) | Go forbids importing `main`. That is why the package clause changes, not a copy | Each RS2 file before RS1 |
| Config loading, logging, telemetry, signals, lifecycle | none (the entrypoint must not use them before or around a role) | `internal/config`, `cmd/internal/runtimeopts`, role bootstraps | Unchanged, inside roles | n/a | none |

## Consumer Changes

None. No shared package loses an importer. `cmd/internal/runtimeopts` keeps
its importers.

## Import graph

`cmd/entrypoint` imports `cmd/<role>`, which imports
`cmd/<role>/internal/bootstrap` and, where present, `cmd/internal/runtimeopts`
and `internal/**`. The graph is acyclic. No role package imports
`cmd/entrypoint` or another role (RS7).
