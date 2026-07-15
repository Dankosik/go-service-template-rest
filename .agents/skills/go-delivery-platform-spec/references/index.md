# Reference Selector

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| CI tiers, required jobs, skipped/cancelled checks, local parity, nightly, or preflight evidence | [ci-gate-matrix-and-blocking-policy.md](ci-gate-matrix-and-blocking-policy.md) | Choose exact jobs, commands, and fail-closed status semantics. |
| Protected branches, rulesets, reviews, CODEOWNERS, bypass, conversations, or merge queue | [branch-protection-and-pr-governance.md](branch-protection-and-pr-governance.md) | Choose enforceable governance and drift guards. |
| OpenAPI/sqlc generation, docs or compatibility drift, and base-reference behavior | [codegen-contract-and-docs-drift.md](codegen-contract-and-docs-drift.md) | Choose generator-backed drift and docs-trigger gates. |
| Schema/data movement, rollback class, mixed versions, backfill, or migrator ownership | [migration-release-safety.md](migration-release-safety.md) | Choose rehearsal, compatibility, sequencing, and rollback controls. |
| Dockerfile/runtime contents, non-root/minimal image, Trivy, or Kubernetes security context when in scope | [container-runtime-hardening.md](container-runtime-hardening.md) | Choose the digest-pinned runtime and scan baseline. |
| Railway health, draining, restart, placement, region/network, capacity, or config drift | [railway-release-runtime-policy.md](railway-release-runtime-policy.md) | Choose repository-reviewable managed-runtime evidence. |
| SBOM, provenance, signing, OIDC, GHCR, or verifier-facing release trust | [supply-chain-provenance-and-sbom.md](supply-chain-provenance-and-sbom.md) | Choose digest-bound artifacts, permissions, and verification proof. |
| Temporary bypass, suppression, override, accepted risk, or rollback exception | [exception-governance.md](exception-governance.md) | Require owner, expiry, compensation, auditability, and reopen conditions. |

Prefer live repository and managed-platform configuration as source-of-truth evidence; references sharpen a triggered decision but do not replace current facts.
