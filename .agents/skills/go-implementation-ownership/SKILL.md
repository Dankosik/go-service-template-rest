---
name: go-implementation-ownership
description: "Source ownership in Go. Use when accepted behavior is closed but its package, file, canonical source, dependency direction, or proof location is not."
metadata:
  invocation: model
  kind: method
---

# Go Implementation Ownership

Every piece of logic, wiring, and proof has one **owner**. Two plausible owners
means placement is not decided.

`responsibilities -> owner -> dependency direction -> generated/manual authority -> proof placement -> cleanup`

Load the [shared specialist contract](../../contracts/specialist-contract.md).
For every changed responsibility, build `OwnerRecord{responsibility,
canonical_source, package, file, dependency_direction, sequence_owner,
proof_location, competing_paths, cleanup}` from accepted behavior through
callers, wiring, generated and manual sources, tests, and cleanup. Ownerless,
duplicated, or competing paths are findings.

The nearest record wins: read package `doc.go`, `README.md`, and seam comments
before [Repository Architecture](../../../docs/repo-architecture.md), [Project
Structure](../../../docs/project-structure-and-module-organization.md), and
[Configuration Source Policy](../../../docs/configuration-source-policy.md).
Gates prove only what they inspect; use `.golangci.yml` and `scripts/ci/` to
subtract mechanical coverage rather than treating green as ownership evidence.

For a **Decision**, apply Project Structure's placement rules and record forced
consequences for every responsibility. For **Review**, account for every
affected owner and competing path. Complete only when implementation can
proceed without choosing among packages, files, sources, dependencies, or
proof levels.
