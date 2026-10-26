# Intent: Ship one runtime executable instead of five to shrink the image

status: ready

## Problem

The runtime image built from `build/docker/Dockerfile` is about 130 MB
uncompressed (about 49 MB gzip). About 128 MB of that is five separately
linked, mostly overlapping executables: `service` 41.7, `outbox-relay` 23.9,
`jobs-worker` 23.3, `migrate` 19.7, and `worker` 19.3 MB (measured
2026-10-26). A symbol-union estimate puts one combined executable at about
42-43 MB, so most of the image is duplicated code.

## Desired outcome

The repository owner decided: "Объединяем, размер образа важнее" (combine
them; image size matters more). The runtime image ships one executable that
provides every role the generated service selected, so the image becomes much
smaller. Existing deployments keep starting each role exactly as they do now.
Derived-service maintainers get a documented migration path.

The request covers the full cycle: specification, design, planning,
implementation, and checks.

## Affected actors and systems

- Operators and deployment configuration that start `/service` (image
  `ENTRYPOINT`), `/migrate` (Railway `preDeployCommand`), `/worker`,
  `/jobs-worker`, and `/outbox-relay`.
- Template maintainers: the Dockerfile, `cmd/`, `make/template.mk`,
  `scripts/init-module.sh` profile stripping, `scripts/integration-init.sh`,
  image and migration check scripts, process integration tests, and docs.
- Generated services and about 29 derived repositories fed by template sync.
  Their customized `cmd/` files will conflict, and their maintainers must
  migrate by hand.

## Scope and non-goals

In scope: the shape of the runtime executable and image, how each role is
entered, how profile selection limits the roles shipped, the template
customization seams that move with it, and the migration guide for derived
services.

Non-goals: changing any role's runtime behavior, configuration, ports, or
deployment topology. Each role still runs as its own process or deployment.

## Constraints

- Distroless static, nonroot, no shell in the image.
- Derived-service merge conflicts in customized `cmd/` files are accepted. The
  migration must be documented, not avoided.
- Reproducible digest, PGO policy, ldflags version stamping, and the Railway
  policy stay as they are unless this change cannot avoid them.

## Success signal

The image built with all roles is much smaller (estimated about 45 MB
uncompressed, about 17 MB gzip; this is an estimate, not a build). Every
existing role invocation behaves as before, and derived-service maintainers
can follow a written migration path.
