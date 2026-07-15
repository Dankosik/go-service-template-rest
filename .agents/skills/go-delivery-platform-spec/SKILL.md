---
name: go-delivery-platform-spec
description: "Use when CI/CD quality gates, merge and release blocking, drift controls, container hardening, rollout policy, or release trust must be decided before coding; Own delivery and platform acceptance policy; Skip when the primary decision is API, data, distributed, security, or reliability behavior, or pipeline implementation."
---

# Go Delivery Platform Spec

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Turn merge, release, rollout, and runtime-hardening expectations into enforceable policy with verifier-facing evidence. Own CI gate tiers and blocking semantics, branch/PR governance, generated/docs drift, container and managed-runtime baselines, supply-chain trust, migration rollout controls, rollback evidence, and time-bounded exceptions.

Consume accepted application, API compatibility, data/migration, distributed recovery, security, and reliability behavior. Do not invent those policies, implementation code, pipeline jobs the repository cannot execute, or platform complexity without a concrete release-safety, compliance, or resilience benefit. Stop at the forced delivery consequence when a neighboring decision is unset.

## Owned Core

- Define proportional `fast-path`, `full`, `nightly`, and `release` tiers only where needed, each with exact repository-owned commands/jobs, trigger, enforcement point, artifact, and blocker. Order integrity, formatting, static/contract/codegen checks, tests, security, integration/race, and build/scan fail-fast; cancelled, timed-out, missing, or incorrectly skipped required checks are failed evidence unless explicit policy says otherwise. Undocumented command substitutions are delivery-policy defects.
- Keep local and CI paths aligned and required status names stable. Require PR-based protected-branch merges, review/CODEOWNERS and conversation rules where applicable, controlled bypass actors, and merge-queue compatibility; reject direct pushes, silent admin bypass, or reviewer-memory gates.
- Bind OpenAPI/sqlc/codegen, docs, compatibility, and base-reference drift to generator- or repository-controlled checks. A gate is blocking only when the repository and CI can execute it and retain auditable logs or artifacts; distinguish current blockers from approved future controls.
- Define compatibility-first migration release policy from accepted data decisions: additive expansion, mixed-version window, rehearsal, one migrator owner, backfill checkpoints, contract timing, rollback class, backup/restore evidence, and explicit irreversible boundaries. Reject destructive-first or per-pod startup migration on active paths unless accepted evidence justifies it.
- Set a risk-based runtime baseline: multi-stage deterministic Go builds, digest-pinned minimal image, non-root execution, explicit user/filesystem/capability/security-context policy, vulnerability scanning, and only required runtime contents. Keep hardening reviewable in repository artifacts.
- Treat dependency controls, least-privilege publish permissions, SBOM, provenance attestations, artifact/image signing, and digest verification as release-trust evidence rather than optional metadata or mutable-tag trust.
- Tie managed-runtime policy to repository-reviewable configuration. Define health checks, restart behavior, overlap/draining, dependency placement and network/region assumptions, capacity baseline, staged promotion checkpoints, objective rollback triggers/actions, and proof that rollback is executable for the release class.
- Treat correctness, security, compatibility, migration safety, and release trust as fail-closed by default. Every temporary exception names scope, owner/approver, rationale, expiry, compensating evidence, audit location, and automatic reopen; expiry or unmet compensation restores the original blocker.

## Symptom-Driven References

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| CI tiers, required jobs, skipped/cancelled checks, local parity, nightly, or preflight evidence | [ci-gate-matrix-and-blocking-policy.md](references/ci-gate-matrix-and-blocking-policy.md) | Choose exact jobs, commands, and fail-closed status semantics. |
| Protected branches, rulesets, reviews, CODEOWNERS, bypass, conversations, or merge queue | [branch-protection-and-pr-governance.md](references/branch-protection-and-pr-governance.md) | Choose enforceable governance and drift guards. |
| OpenAPI/sqlc generation, docs or compatibility drift, and base-reference behavior | [codegen-contract-and-docs-drift.md](references/codegen-contract-and-docs-drift.md) | Choose generator-backed drift and docs-trigger gates. |
| Schema/data movement, rollback class, mixed versions, backfill, or migrator ownership | [migration-release-safety.md](references/migration-release-safety.md) | Choose rehearsal, compatibility, sequencing, and rollback controls. |
| Dockerfile/runtime contents, non-root/minimal image, Trivy, or Kubernetes security context when in scope | [container-runtime-hardening.md](references/container-runtime-hardening.md) | Choose the digest-pinned runtime and scan baseline. |
| Railway health, draining, restart, placement, region/network, capacity, or config drift | [railway-release-runtime-policy.md](references/railway-release-runtime-policy.md) | Choose repository-reviewable managed-runtime evidence. |
| SBOM, provenance, signing, OIDC, GHCR, or verifier-facing release trust | [supply-chain-provenance-and-sbom.md](references/supply-chain-provenance-and-sbom.md) | Choose digest-bound artifacts, permissions, and verification proof. |
| Temporary bypass, suppression, override, accepted risk, or rollback exception | [exception-governance.md](references/exception-governance.md) | Require owner, expiry, compensation, auditability, and reopen conditions. |

Prefer live repository and managed-platform configuration as source-of-truth evidence; references sharpen a triggered decision but do not replace current facts.

## Return And Stop

Return only the triggered gate matrix and blocking policy; merge/release hard stops; drift, migration, container/runtime, managed-platform, provenance/SBOM/signing, rollback, and exception contracts; assumptions; verifier-facing evidence; and forced neighboring consequences.

Block release on red required CI, failing required/nightly reliability evidence, unresolved high-risk findings, missing trust or rollback evidence, unsafe migration sequencing, an implicit runtime-hardening baseline, unenforceable policy, or an expired/uncontrolled bypass. Block specification when release safety depends on unset compatibility, migration, distributed recovery, security, reliability, runtime, or capacity policy; name that owner rather than choosing its behavior.
