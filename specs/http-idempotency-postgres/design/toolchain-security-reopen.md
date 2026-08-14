# Go 1.26.6 security-toolchain reopen

status: ready
macro phase: Technical Design
authority: `go.mod`, `tools/go.mod`, and `build/docker/Dockerfile`

## Decision

Upgrade the repository-controlled Go toolchain from 1.26.5 to 1.26.6. The root
module directive is the host and GitHub Actions source of truth: the composite
setup action reads `go.mod`, and Make derives `GO_REQUIRED_VERSION` and its
coverage/gosec toolchain settings from that directive. `tools/go.mod` is a
coequal version source for the tools module: `make mod-tidy-check` requires it
to match the root directive. The Docker build source of truth is its first
`FROM`; pin it to
`golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`.

Go's published 1.26.6 release supplies the required patch level. This is the
narrowest coherent repository target: no workflow, Makefile, setup action, or
application source edit is needed because each already consumes the named
authority.

## Implementation ownership and proof

One fresh implementation acceptance unit owns only:

- `go.mod` and `tools/go.mod`: `go 1.26.6` in both modules;
- `build/docker/Dockerfile`: the exact 1.26.6 Bookworm index digest above.

Before acceptance, it must prove the selected host toolchain is `go1.26.6`,
the module compiles and security scanner no longer reports GO-2026-6218,
GO-2026-6091, GO-2026-6090, GO-2026-6089, GO-2026-6088, GO-2026-5972, or
GO-2026-5026, and a Docker build uses the pinned builder image. The existing
PR-GOSEC-01 aggregate and its independent review remain the acceptance gate;
this design does not accept that unit.

## Exclusions and reopen condition

`.github/actions/setup-go/action.yml` and `Makefile` are verified consumers,
not version sources. `scripts/ci/s3-source-receipt.sh` and all
`specs/s3-compatible-object-storage/` artifacts are untracked, unrelated
candidate work in this checkout; they currently pin Go-1.26.5 image/source
identities and must be reconciled by their own S3 owner before that candidate
can claim its source receipt again. They are not part of this unit.

Reopen Technical Design if 1.26.6 changes an accepted runtime/image contract,
the pinned index cannot be resolved, or an existing consumer does not derive
the version from these three sources. Reopen S3 Technical Design if its candidate
is to be accepted against the upgraded image.
