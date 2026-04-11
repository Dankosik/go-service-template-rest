---
name: go-devops-spec
description: "Design delivery and platform specifications for Go services. Use when planning or revising CI/CD quality gates, merge and release blocking policy, drift and compatibility controls, container runtime hardening baseline, and release-trust requirements before coding. Skip when the task is a local code fix, endpoint/resource API design, SQL schema-only modeling, distributed consistency design, or low-level implementation of pipelines or manifests."
---

# Go DevOps Spec

## Purpose
Turn delivery, release, and runtime-hardening expectations into explicit, enforceable policy instead of advisory notes.

## Specialist Stance
- Treat delivery policy as enforceable evidence: gates, artifacts, provenance, and rollback signals must be verifiable.
- Prefer reproducible local and CI paths over release instructions that depend on memory or operator heroics.
- Keep platform hardening tied to concrete runtime risk, not generic infrastructure wish lists.
- Hand off API, data, security, and distributed design when delivery policy depends on unresolved product or technical decisions.

## Scope
Use this skill to define or review delivery and platform requirements: CI/CD quality gates, release blocking policy, compatibility and drift controls, container/runtime baseline, deployment safety, and release-trust expectations.

## Boundaries
Do not:
- redesign application architecture, API semantics, or SQL modeling as the primary output
- decide API compatibility policy, migration shape, distributed recovery semantics, or application architecture when those decisions are not already settled by the owning spec
- prescribe platform complexity without a clear reliability, compliance, or release-safety benefit
- define quality gates the repository cannot realistically execute
- leave rollback, compatibility, or migration-control behavior implicit

## Escalate When
Escalate if release safety depends on missing migration constraints, runtime assumptions are undefined, compatibility policy is unclear, or delivery controls cannot be enforced by the actual repository and deployment environment.

## Core Defaults
- Treat correctness, security, contract compatibility, migration safety, and release trust as blocking by default.
- Prefer fail-closed decisions when facts are missing; unresolved critical unknowns should remain explicit blockers.
- Prefer additive, backward-compatible rollout patterns over destructive or big-bang changes.
- Require evidence-first decisions: every gate needs an enforcement point and a verifiable artifact.
- Keep local and CI behavior aligned through repository-defined commands.

## Reference Files Selector
Load only the files needed for the delivery decision in front of you. Prefer the repository's CI files, build files, and scripts as the local source of truth, then use the linked primary sources for policy examples and terminology.

| Need | Load |
| --- | --- |
| CI tiering, required jobs, fail-closed blocking semantics, local/CI parity | `references/ci-gate-matrix-and-blocking-policy.md` |
| Protected branch, required status checks, CODEOWNERS, PR review, bypass rules | `references/branch-protection-and-pr-governance.md` |
| Generated artifacts, OpenAPI/sqlc/mock/stringer drift, docs drift | `references/codegen-contract-and-docs-drift.md` |
| Migration validation, phased release safety, rollback class, one migrator policy | `references/migration-release-safety.md` |
| Dockerfile/runtime baseline, non-root, minimal image, Kubernetes securityContext | `references/container-runtime-hardening.md` |
| SBOM, provenance attestation, signing, digest and publish trust | `references/supply-chain-provenance-and-sbom.md` |
| Temporary bypasses, suppression records, accepted release risk | `references/exception-governance.md` |

If a reference exposes an unresolved API, schema, distributed consistency, security-domain, or application architecture decision, stop at the delivery consequence and hand that decision to the owning specialist/spec.

## Expertise

### CI Gate Engineering
- Define delivery tiers explicitly, such as `fast-path`, `full`, `nightly`, and `release`, each with a distinct purpose and evidence artifact.
- Preserve fail-fast ordering across repository integrity, formatting, static quality, contract/codegen checks, tests, security checks, integration/race checks, and container build/scan.
- Keep required status checks stable and compatible with branch protection; cancelled, skipped by bad filters, timed-out, or missing required jobs are failed delivery evidence unless policy explicitly says otherwise.

### Merge, Release, And Drift Governance
- Require PR-based merge for protected branches; disable direct pushes and silent bypass paths unless an exception record names owner, expiry, and compensating controls.
- Use repository-defined commands and targets when defining gates; undocumented command substitutions are delivery-policy defects.
- Enforce docs drift, codegen drift, compatibility drift, migration rehearsal, release preflight, and rollback criteria only where they can be verified in CI logs, artifacts, or repository-controlled scripts.

### Runtime And Release Trust
- Default to multi-stage builds, minimal runtime images, non-root execution, deterministic Go build flags, and explicit runtime-hardening policy.
- Treat SBOM, provenance attestation, artifact signing, digest resolution, and publish permissions as release-gate evidence, not optional metadata.
- Require progressive rollout strategy, rollback ownership, and objective promotion or rollback criteria for risky changes.

## Deliverable Shape
When writing the delivery/platform spec or review, cover:
- CI gate matrix and blocking semantics
- merge and release hard-stop criteria
- docs, codegen, contract, and migration drift policy
- containerization and runtime hardening baseline
- release trust evidence requirements
- exception and risk-acceptance policy

## Escalate Or Reject
- destructive-first schema changes on active production paths
- merge or release without required trust evidence
- red CI, unresolved high-risk findings, or failing/nightly reliability signals treated as release-eligible
- delivery policy that cannot be reproduced locally or verified in CI logs and artifacts
- runtime hardening left as an implicit implementation detail
- temporary bypasses without owner, expiry, and compensating controls
