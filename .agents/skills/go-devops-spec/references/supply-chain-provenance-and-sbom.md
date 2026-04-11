# Supply Chain Provenance And SBOM

## When To Load
Load this when specifying release-trust evidence, image signing, SBOM, provenance attestation, OIDC permissions, digest pinning, GHCR publish policy, SLSA expectations, or verification before deploy.

## Local Source Of Truth
- `.github/workflows/cd.yml` grants `contents: read`, `packages: write`, `id-token: write`, and `attestations: write`; builds images; scans with Trivy; generates CycloneDX SBOM through Trivy; pushes tags; resolves digest; installs cosign; signs by digest; attests build provenance with `actions/attest-build-provenance`; and uploads the SBOM artifact.
- `.github/workflows/ci.yml` and `.github/workflows/nightly.yml` use SHA-pinned actions and image scanning gates.
- `build/docker/Dockerfile` pins base images by digest.

## Enforceable Policy Examples
- Publish jobs must sign and attest the immutable image digest, not only mutable tags.
- Provenance and SBOM generation require the minimum job permissions that the action needs; jobs that do not publish or attest keep read-only permissions.
- Release tags run `release-preflight` before `publish-release`; publish without successful preflight is blocked.
- SBOM artifacts must be uploaded with `if-no-files-found: error` or equivalent fail-closed behavior.
- Consumers or deployers that verify release trust must verify the attestation identity, source repository/ref, subject digest, and signer/workflow identity before accepting the artifact.
- Base images and third-party actions should be pinned to immutable digests or SHAs with an update policy, not floating names alone.

## Non-Enforceable Anti-Patterns
- Signing `latest` or `main` tags without resolving and signing the digest.
- Generating SBOMs after publish without tying them to the released digest.
- Granting `write-all` permissions to the full workflow when only publish/attestation jobs need write scopes.
- Treating artifact attestations as meaningful without a verification step or policy consumer.
- Uploading SBOM/provenance as optional artifacts where missing files do not fail the job.
- Rebuilding the release artifact after the artifact was scanned or attested.

## Evidence Artifacts
- CD workflow run URL and commit SHA for `release-preflight` and `publish-release`.
- Published image references: version tag, sha tag, latest/main tag where used, and immutable digest.
- Cosign signing log for the digest.
- Build provenance attestation log and attestation identifier or registry attachment.
- SBOM artifact name and uploaded file, for example `sbom-<version>` or `sbom-main-<short-sha>`.
- Verification command output from `gh attestation verify`, `slsa-verifier`, cosign, or the deployment platform's admission verification.

## Hand-Off Boundary
Do not define vulnerability acceptance, cryptographic trust model, or deployment admission architecture here beyond delivery proof requirements. Route unresolved security policy to the security spec and unresolved platform admission design to platform architecture.

## Exa Source Links
- GitHub Docs: [Using artifact attestations to establish provenance for builds](https://docs.github.com/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds)
- GitHub Docs: [OpenID Connect reference](https://docs.github.com/en/actions/reference/security/oidc)
- SLSA: [Build: Distributing provenance](https://slsa.dev/spec/v1.2/distributing-provenance)
- SLSA Framework: [slsa-verifier](https://github.com/slsa-framework/slsa-verifier)
- OpenSSF: [scorecard-action](https://github.com/ossf/scorecard-action)
- Docker Docs: [Building best practices](https://docs.docker.com/build/building/best-practices/)

