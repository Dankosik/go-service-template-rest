---
name: go-implementation-ownership
description: "Go ownership: Use for package/file/dependency or boundary decisions. Own code placement; Skip topology, policy, semantics, and readability."
---

# Go Implementation Ownership

Every piece of logic, wiring, and proof has one **owner**. Two plausible owners
means placement is not decided.

`responsibilities -> owner -> dependency direction -> generated/manual authority -> proof placement -> cleanup`

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
changed responsibilities and executable paths from accepted behavior, callers,
wiring, generated and manual sources, tests, and cleanup. Assign one source of
truth, package/file owner, dependency direction, sequence owner, and proof
surface to each path; ownerless, duplicated, or competing paths are findings.

The nearest record wins: read package `doc.go`, `README.md`, and seam comments
before [Repository Architecture](../../../docs/repo-architecture.md), [Project
Structure](../../../docs/project-structure-and-module-organization.md), and
[Configuration Source Policy](../../../docs/configuration-source-policy.md).
Gates prove only what they inspect; use `.golangci.yml` and `scripts/ci/` to
subtract mechanical coverage rather than treating green as ownership evidence.

For a **Decision**, apply Project Structure's placement rules and record forced
consequences for every responsibility. For **Review**, account for every
affected owner and competing path in the shared finding envelope. Complete only
when implementation can proceed without choosing among packages, files,
sources, dependencies, or proof levels.

Hand runtime topology to `go-system-architecture`, behavior to its domain skill,
and implementation mechanics to `go-coder`.
