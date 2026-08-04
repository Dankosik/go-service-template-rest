---
name: go-implementation-ownership
description: "Implementation ownership: Use for Go package/file/dependency, cleanup, proof placement, or boundary review. Own code placement; Skip system topology, domain policy, Go semantics, or local readability."
---

# Go Implementation Ownership

Every piece of logic, wiring, and proof has exactly one **owner**: which package, file, and dependency direction it lives in is a decision, and two plausible owners means the decision has not been made.

`responsibilities -> single owner -> dependency direction -> generated vs manual -> proof placement -> cleanup`

Dependency direction follows stability — the frequently changing depends on the stable; generated and manual sources never share a responsibility; proof lives next to what it proves; and an ownerless path, a duplicated implementation, or two competing sources of one truth is a finding even while both currently agree.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct changed responsibilities and executable paths from accepted behavior, current callers and owners, generated/manual sources, wiring, tests, and cleanup surfaces; assign one owner for each source of truth, package/file responsibility, dependency direction, sequence, and proof surface, exposing ownerless, duplicated, or competing paths.

## Where Ownership Is Recorded

Ownership here is written down, and the nearest record wins. Read the package's
own `doc.go`, `README.md`, and the doc comment on the crossed seam before any
repository-wide document: `internal/openapi/README.md`, `test/README.md`,
`internal/infra/http/doc.go`, `internal/infra/http/identity.go`, and
`internal/infra/postgres/postgres.go` each name a boundary with the failure it
exists to prevent. Repository-wide owners are
[Repository Architecture Baseline](../../../docs/repo-architecture.md)
for component boundaries, source-of-truth, and dependency direction;
[Project Structure & Module Organization](../../../docs/project-structure-and-module-organization.md)
for placement, file naming, generated authority, and test level; and
[Configuration Source Policy](../../../docs/configuration-source-policy.md) for
what `internal/config` owns and which external variables an adapter honors
directly.

A green gate is not ownership evidence. `depguard` reads imports,
`project-structure-check` reads file and directory names plus the `test/`
build-tag shape, and each `*-check` regenerates and compares bytes. None sees a
re-derived rule, a constant copied under a comment claiming it mirrors another
file, a bucket named something the check does not spell, or a test placed at a
level it does not need. Read `.golangci.yml` and `scripts/ci/` for what the
gates already prove, then spend findings on what is left.

## Choose The Branch

- **Decision** — select when accepted behavior exists but implementation placement is absent or changing. [Project Structure & Module Organization](../../../docs/project-structure-and-module-organization.md) owns the placement rule, its worked results, and the first-feature order; apply it and record the forced consequences rather than restating it. Stop on any unresolved prerequisite; otherwise complete when shared Decision dispositions cover every responsibility and path with forced consequences explicit.
- **Review** — select when changed Go must conform to accepted placement. Account for every affected owner and competing path through the shared finding envelope.

Hand runtime topology to `go-system-architecture` and behavior policy to its domain skill. On the implementation path, `go-coder` owns import direction against `depguard`, regeneration mechanics for each canonical source, and helper-versus-interface placement.
