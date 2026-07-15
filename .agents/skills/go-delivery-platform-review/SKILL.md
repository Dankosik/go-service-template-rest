---
name: go-delivery-platform-review
description: "Use when a diff changes CI/CD, merge or release gates, generated-artifact drift controls, migration rollout, containers, deployment policy, or release evidence; Own conformance to accepted delivery and platform policy; Skip when the primary issue requires new release policy or deep data, security, reliability, or observability review."
---

# Go Delivery Platform Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target, Boundary, And Invariants

Review changed CI/CD, merge/release gates, generated-artifact controls, migrations, containers, deployment policy, exceptions, and release-trust evidence for enforceable accepted delivery behavior. Ignore ordinary application code unless it changes packaging, migration, deployment, or gate behavior; do not treat advisory evidence as blocking or absorb the domain risk a gate merely detects.

1. Required workflows, jobs, branch-protection contexts, triggers, dependencies, and aggregate status propagate failure, cancellation, timeout, missing, and skip outcomes according to accepted merge/release policy.
2. Local commands and CI/CD jobs prove the same named contract or document the accepted difference; merge, nightly, manual, and release evidence are not interchangeable.
3. Docs, OpenAPI, sqlc, and other generated outputs have one canonical source and drift/codegen gates triggered by every relevant source change.
4. Compatibility and migration changes preserve accepted mixed-version, rehearsal, rollback/restore, backfill, sequencing, and one-migrator release controls.
5. Container builds and runtime images preserve deterministic, minimal, non-root, secret-safe, pinned, and scanned behavior where those controls are accepted.
6. Deployment configuration, health/promotion gates, placement, capacity, overlap/drain, rollback, and post-deploy evidence remain repo-reviewable and target-platform specific.
7. Bypasses and waivers remain explicit, scoped, approved, expiring, compensating-proof-backed, auditable, and tied to reopen/closure conditions.
8. Published artifacts preserve least-privilege build/publish permissions, immutable identity, scan, SBOM, provenance, signature/attestation, and verifier-facing release trust.

## Symptom-Driven References

| Symptom | Load |
| --- | --- |
| Required CI tiers, status semantics, local parity, merge/release blocking, or cancelled/skipped evidence changes. | [ci-gate-matrix-and-blocking-policy.md](../go-delivery-platform-spec/references/ci-gate-matrix-and-blocking-policy.md) |
| Branch protection, required reviews/checks, CODEOWNERS, bypass actors, rulesets, or merge queue changes. | [branch-protection-and-pr-governance.md](../go-delivery-platform-spec/references/branch-protection-and-pr-governance.md) |
| OpenAPI/sqlc generation, compatibility checks, tracked output, or docs drift changes. | [codegen-contract-and-docs-drift.md](../go-delivery-platform-spec/references/codegen-contract-and-docs-drift.md) |
| Migration rehearsal, rollback class, mixed-version windows, backfill, sequencing, or migrator ownership changes. | [migration-release-safety.md](../go-delivery-platform-spec/references/migration-release-safety.md) |
| Dockerfile/image contents, runtime user, secrets, digest pinning, scan gates, or runtime hardening changes. | [container-runtime-hardening.md](../go-delivery-platform-spec/references/container-runtime-hardening.md) |
| Railway placement, health, overlap/draining, restart, capacity, promotion, rollback, or config drift changes. | [railway-release-runtime-policy.md](../go-delivery-platform-spec/references/railway-release-runtime-policy.md) |
| A required gate, scan, migration, protection, or rollout control is waived or downgraded. | [exception-governance.md](../go-delivery-platform-spec/references/exception-governance.md) |
| Publish permissions, immutable digests, SBOM, provenance, signing, attestations, or verification change. | [supply-chain-provenance-and-sbom.md](../go-delivery-platform-spec/references/supply-chain-provenance-and-sbom.md) |

## Findings And Escalation

Inspect repository gate definitions, branch/release conclusions, command parity, drift triggers, migration/deployment order, container and platform configuration, exception records, artifact identity, and verifier evidence. Each finding adds the failed delivery/release control, concrete merge or rollout risk, and reproducible CI/runtime proof. `critical` means the release path can ship high-impact broken, unsafe, or unreviewed artifacts; `high` means required gate, migration, runtime, rollout, exception, or trust evidence is materially bypassed.

Escalate unset or changed delivery policy to `go-delivery-platform-spec`; API/data compatibility or migration semantics to `go-api-contract-spec`, `go-data-architecture-spec`, or `go-db-cache-spec`; runtime failure policy to `go-reliability-spec`; trust decisions to `go-security-spec`; and signal/runbook policy to `go-observability-spec`. Stop rather than approving a release on invented policy or duplicating the deeper owner.
