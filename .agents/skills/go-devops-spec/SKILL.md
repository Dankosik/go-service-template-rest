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
- If another domain is only affected, record the delivery consequence or proof obligation and use `no new decision required` unless that domain must make a new decision now.

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
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default unless the task clearly spans multiple independent decision pressures. Prefer live repository files as the source of truth, then use a reference to sharpen the decision.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| CI tiers, required jobs, skipped/cancelled checks, local/CI parity, nightly or release preflight evidence | `references/ci-gate-matrix-and-blocking-policy.md` | Choose exact repo-owned jobs, Make targets, and fail-closed status semantics instead of vague "usual checks" or advisory evidence. |
| Protected branch setup, rulesets, required reviews, CODEOWNERS, bypass actors, conversation resolution, merge queue readiness | `references/branch-protection-and-pr-governance.md` | Choose enforceable branch-protection settings plus drift guards instead of generic "use branch protection" language. |
| Generated OpenAPI/sqlc/mock/stringer artifacts, docs drift, compatibility-check blocking, base-ref behavior | `references/codegen-contract-and-docs-drift.md` | Choose generator-backed drift gates and docs-trigger policy instead of reviewer-memory regeneration advice. |
| Schema migrations, data-moving release, rollback class, mixed-version window, backfill gates, one-migrator policy | `references/migration-release-safety.md` | Choose rehearsal, rollback classification, compatibility, and migrator ownership instead of "run migrations before deploy." |
| Dockerfile, runtime image contents, non-root/minimal image baseline, Trivy gate, Kubernetes securityContext when actually in scope | `references/container-runtime-hardening.md` | Choose the repo's digest-pinned non-root runtime baseline and scan gates instead of generic image-hardening advice. |
| Railway deployment policy, healthcheck, overlap/draining, restart policy, capacity baseline, platform drift in `railway.toml` | `references/railway-release-runtime-policy.md` | Choose repo-reviewable Railway platform evidence instead of generic Kubernetes rollout or manual monitoring language. |
| SBOM, provenance, image signing, OIDC permissions, GHCR publish, SLSA-style verifier-facing release trust | `references/supply-chain-provenance-and-sbom.md` | Choose digest-bound signing, provenance, SBOM, and verification proof instead of signing mutable tags or optional metadata. |
| Temporary bypass, suppression, accepted release risk, manual release override, branch-protection bypass, rollback exception | `references/exception-governance.md` | Challenge the exception path with owner, expiry, compensating proof, and reopen conditions instead of silently downgrading gates. |

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
- Tie deployment-platform policy to repo-reviewable surfaces such as `railway.toml`; require objective promotion or rollback criteria for risky changes.

## Deliverable Shape
When writing the delivery/platform spec or review, cover:
- CI gate matrix and blocking semantics
- merge and release hard-stop criteria
- docs, codegen, contract, and migration drift policy
- containerization and runtime hardening baseline
- deployment-platform health, restart, rollout, and capacity evidence when platform policy is in scope
- release trust evidence requirements
- exception and risk-acceptance policy
- downstream decision/proof consequences only when another domain must act now; otherwise use `no new decision required in <domain>`

## Escalate Or Reject
- destructive-first schema changes on active production paths
- merge or release without required trust evidence
- red CI, unresolved high-risk findings, or failing/nightly reliability signals treated as release-eligible
- delivery policy that cannot be reproduced locally or verified in CI logs and artifacts
- runtime hardening left as an implicit implementation detail
- temporary bypasses without owner, expiry, and compensating controls
