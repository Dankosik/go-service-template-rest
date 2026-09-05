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

Cohesion: Group declarations that must be understood and changed together.
Use package boundaries for dependency and API isolation, and files for
navigable, cohesive behavior.

Caller view: Exercise each new or changed package boundary from a real
caller's perspective; make inputs, results, errors, and lifecycle
obligations understandable without inspecting implementation details.

Debug path: Walk a representative failure symptom through the proposed
owners to the responsible rule, effect, and regression-test location.
Reduce unrelated context needed to follow that path.

For a **Decision**, apply Project Structure's placement rules and record forced
consequences for every responsibility. For **Review**, account for every
affected owner and competing path. Complete only when implementation can
proceed from a current, unambiguous map of packages, files, sources,
dependencies, and proof levels.
