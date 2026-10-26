# Runtime Executable Migration

The template now ships one runtime executable for every role instead of one
executable per role. Use this guide to move a derived repository that still
has a `main` package in each `cmd/<role>` directory. Template sync mirrors the
tooling that assumes the new layout, so migrate before relying on synced role
build and run targets.

## Read When

- A synced role build or run target, such as `make build` or
  `make run-worker`, exits 1 and names this guide.
- Porting role-local customizations from a `cmd/<role>/main.go` layout.
- Verifying an image built after the migration.

## What Changes And What Does Not

Deployment configuration does not change. The image keeps the same entry
paths: `/service` (also the image `ENTRYPOINT`), and, when that role is
present, `/worker`, `/jobs-worker`, `/outbox-relay`, and `/migrate`. Each takes
the same `-config` and `-config-overlay` arguments, environment, and signals,
and keeps its exit statuses, output, and drain behavior. Railway service
settings, including the `/migrate` pre-deploy command, Kubernetes `command`
values, and `docker run --entrypoint` need no edit.

The image shrinks because every role used to carry its own copy of the Go
runtime, the shared internal packages, and their dependencies. Now one file,
`/service`, holds them once, and every other entry path is a relative symbolic
link to it. The executable selects its role from the final element of the path
it was started through (`argv[0]`). The match is exact; any other name exits 2
with one stderr line that lists the roles the image contains. No role is
selected by default.

The role selection runs before any configuration, environment, or signal
handling. Package initializers of every linked role, however, run in every
role's process. Role code therefore keeps I/O, registration, and logging out
of package-level variable initializers and `init` functions; an `init` may only
assign a role-local value, such as a builder.

## Customization Seams

| Seam | Old location | New location | How to port |
| --- | --- | --- | --- |
| Service composition root: handler wiring, domain error mapping, gRPC runtime construction | `cmd/service/internal/bootstrap/` | Unchanged | Nothing to port. |
| `make integration-init` additions | `cmd/service/internal/bootstrap/startup_*.go` | Unchanged | Nothing to port; they stay active only in the service role. |
| Messaging `HandlerBuilder` | The `bootstrap.Run(os.Args[1:], <builder>)` call in `cmd/worker/main.go` | The `bootstrap.Run(args, <builder>)` call in `Main` in `cmd/worker/role.go` | Pass your builder in place of `nil` in `Main`. |
| Jobs `WorkersBuilder` | `cmd/jobs-worker/builder*.go`, package `main` | Same files, package `jobsworker` | Change only the package clause. Build tags and the build-tag test builders stay. |
| Inbound webhook handler registration | `registerInboundWebhookHandlers` in `cmd/jobs-worker/builder_webhooks.go` and `cmd/jobs-worker/inbound_webhook_bindings.go`, package `main` | Same files and functions, package `jobsworker` | Change only the package clause. |

A customization still lives in one role-local place and still changes only its
own role.

## Repository Changes

Make these changes in one commit, then run template sync as usual.

1. For each present role directory, rename `cmd/<role>/main.go` to
   `cmd/<role>/role.go`. Change its package clause to the role's package name:
   `service`, `worker`, `jobsworker`, `outboxrelay`, or `migrate`. Replace
   `func main()` with `func Main(args []string) int`. Use `args` where the
   body read `os.Args[1:]`, return 1 where it called `os.Exit(1)`, and return 0
   at the end. Keep the imports, the stream writes, and the body otherwise
   unchanged.
2. Change the package clause of every other non-`internal` Go file directly in
   that role directory, including `_test.go` files, the same way. For example,
   `cmd/migrate/run.go`, `cmd/migrate/run_test.go`, and the
   `cmd/jobs-worker/builder*.go` files.
3. Add `cmd/entrypoint/main.go` from the template. Keep one import and one table
   entry for each role your repository has, in the order `service`, `worker`,
   `jobs-worker`, `outbox-relay`, `migrate`, and delete the rest. Keep
   `pinnedRole` unset in shipped builds; only local Make artifacts set it.
4. In `build/docker/Dockerfile`, build once with
   `go build ... -o /out/bin/service ./cmd/entrypoint`, keeping your existing
   PGO, `-trimpath`, `-mod=readonly`, `-buildvcs=false`, and `-ldflags`
   arguments. In the same `RUN`, create `ln -s service /out/bin/<role>` for each
   other role you ship. Pin timestamps with
   `find /out -exec touch -h -d "@${SOURCE_DATE_EPOCH}" {} +` so the links are
   reproducible too. In the runtime stage, replace the per-role `COPY` lines
   with one `COPY --link --from=build /out/bin/ /`, and keep the
   `/migrations` copy. Copying a link file by file would duplicate the
   executable, so copy the directory.
5. Replace any repository-owned `go build ./cmd/<role>` or `go run ./cmd/<role>`
   command, for example in process integration tests, with a build of
   `./cmd/entrypoint` written to a file named after the role, such as
   `<tmp>/worker`, then run that file.

Synced tooling assumes this layout. The `make` role build and run targets build
`./cmd/entrypoint` with the role pinned at link time, because their output
names, such as `bin/<service>-worker`, are not role names. On the old layout
they fail with a non-zero status and name this guide rather than build the wrong role. Other synced
scripts address `cmd/<role>/internal/bootstrap` and the image's entry paths,
which do not move.

## PGO

The accepted-build fingerprint now covers `cmd/entrypoint`, so an accepted PGO
manifest from before the migration no longer matches and the build fails
closed. Re-accept the profile with `make pgo-manifest` against the migrated
tree before building with `PGO_PROFILE`.

## Verify The Migrated Image

1. Build the image with `make runtime-image-build`.
2. Run `make runtime-image-check` for the default `ENTRYPOINT`, readiness,
   version, and SIGTERM exit 0.
3. If the repository has the migration role, run `make migration-validate`.
4. Start each other entry path your deployments use with
   `docker run --rm --entrypoint /<role> <image> ...` and compare its exit
   status and stderr with the pre-migration image.
5. Compare sizes with `docker image inspect -f '{{.Size}}' <image>` against the
   pre-migration image. The migrated image should be no larger, and an image
   with several roles should be markedly smaller.
