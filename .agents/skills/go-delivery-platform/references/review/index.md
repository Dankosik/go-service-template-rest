# Reference Selector

| Symptom | Load |
| --- | --- |
| Required CI tiers, status semantics, local parity, merge/release blocking, or cancelled/skipped evidence changes. | [ci-gate-matrix-and-blocking-policy.md](../decision/ci-gate-matrix-and-blocking-policy.md) |
| Branch protection, required reviews/checks, CODEOWNERS, bypass actors, rulesets, or merge queue changes. | [branch-protection-and-pr-governance.md](../decision/branch-protection-and-pr-governance.md) |
| OpenAPI/sqlc generation, compatibility checks, or tracked output changes. | [codegen-contract-and-generated-drift.md](../decision/codegen-contract-and-generated-drift.md) |
| Migration rehearsal, rollback class, mixed-version windows, backfill, sequencing, or migrator ownership changes. | [migration-release-safety.md](../decision/migration-release-safety.md) |
| Dockerfile/image contents, runtime user, secrets, digest pinning, scan gates, or runtime hardening changes. | [container-runtime-hardening.md](../decision/container-runtime-hardening.md) |
| Railway placement, health, overlap/draining, restart, capacity, promotion, rollback, or config drift changes. | [railway-release-runtime-policy.md](../decision/railway-release-runtime-policy.md) |
| A required gate, scan, migration, protection, or rollout control is waived or downgraded. | [exception-governance.md](../decision/exception-governance.md) |
| Publish permissions, immutable digests, SBOM, provenance, signing, attestations, or verification change. | [supply-chain-provenance-and-sbom.md](../decision/supply-chain-provenance-and-sbom.md) |
