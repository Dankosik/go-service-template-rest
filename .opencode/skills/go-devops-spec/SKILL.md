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

## Expertise

### CI Gate Engineering
- Define delivery tiers explicitly, for example `fast-path`, `full`, `nightly`, and `release`, each with a distinct purpose.
- Preserve fail-fast execution order across repository integrity, formatting, static quality, contract/codegen checks, tests, security checks, integration/race checks, and container build/scan.
- Keep required status checks stable and compatible with branch protection.
- Treat cancelled or timed-out required jobs as failed gates.

### Branch Protection And PR Governance
- Require PR-based merge for protected branches; disable direct pushes and silent bypass paths.
- Require up-to-date branches, approved review, stale-approval dismissal, and resolved conversations.
- Include administrators in protection policy to avoid privileged bypass drift.
- Keep repository guardrails present on the default branch, such as `CODEOWNERS`, PR templates, and security/contribution policy files.

### Command Fidelity And Drift Control
- Use repository-defined commands and targets when defining gates.
- Preserve CI/local parity for module integrity, formatting, linting, tests, OpenAPI checks, and security checks.
- Make conditional checks explicit, such as breaking-contract checks or migration validation when applicable.
- Treat undocumented or non-reproducible command substitutions as policy defects.
- Enforce docs drift, codegen drift, and compatibility drift for behavior-changing paths.

### Migration And Data-Evolution Safety
- Require phased schema rollout: `expand -> migrate/backfill -> contract`.
- Require mixed-version compatibility across rolling or canary deployment windows.
- Use one controlled migrator process rather than migration-on-every-pod startup behavior.
- Require migration safety budgets, idempotent/resumable backfills, durable checkpoints, and verification gates before contract.
- Declare rollback class and call out irreversible steps.
- Reject DB+publish dual writes; require outbox or equivalent atomic publication when applicable.
- Treat failed restore drills or untested backup posture as release blockers.

### Security And Identity Safety In Delivery
- Keep source-security and container-security scanning blocking for merge or release as policy requires.
- Require a suppression process with owner, rationale, expiry, and review trail.
- Require secure-by-default delivery expectations on changed trust boundaries: bounded inputs, strict validation, parameterized data access, and no secret leakage in logs or errors.
- Require fail-closed authn/authz and tenant-scoping behavior for delivery-significant security changes.

### Containerization And Runtime Hardening
- Default to multi-stage builds with a minimal runtime image and no leftover build toolchain.
- Prefer distroless, non-root, exec-form entrypoints, and static linking unless cgo needs are real and explicit.
- Require CA trust and timezone strategy when outbound TLS or time logic matters.
- Require reproducible build defaults such as `-trimpath`, `-mod=readonly`, deterministic build flags, and explicit Go version control.
- Require `.dockerignore` and a hardening baseline: non-root user, read-only root filesystem, no privilege escalation, dropped Linux capabilities, and no privileged or host-level modes without exception.

### Release Trust And Supply Chain Evidence
- Treat SBOM, provenance attestation, and artifact signing as release-gate evidence, not optional metadata.
- Require release preflight gates before publish on version tags.
- Require verifiable permissions and configuration for attestations, signing, and registry publish flows.
- Prefer digest pinning and explicit tool/base-image version management.
- Reject release flows that cannot prove artifact integrity and origin.

### Observability-, SLO-, And Rollout-Aware Delivery
- Require telemetry baseline for changed production paths: structured logs, RED metrics plus saturation signals, and trace propagation across sync/async boundaries.
- Keep metric dimensions low-cardinality.
- Tie release permissions to service health and budget state where SLOs are used.
- Require progressive rollout strategy, rollback ownership, and objective promotion/rollback criteria for risky changes.
- Treat active page-level burn or unresolved high-risk findings as rollout blockers.

### Exception Governance
- Every temporary bypass needs an owner, expiry, compensating controls, and reopen condition.
- No gate may be silently downgraded from blocking to informational.
- Reject “fix later” language without a bounded remediation plan and accountability.

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
