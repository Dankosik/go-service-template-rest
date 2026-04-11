# Supply Chain Provenance And SBOM

## Behavior Change Thesis
When loaded for symptom "the delivery spec needs release-trust evidence," this file makes the model choose digest-bound signing, provenance, SBOM, and verifier-facing proof instead of likely mistake signing mutable tags, uploading optional metadata, or granting broad workflow permissions.

## When To Load
Load for release-trust evidence, image signing, SBOM, provenance attestation, OIDC permissions, digest pinning, GHCR publish policy, SLSA expectations, or verification before deploy.

## Local Source Of Truth
- `.github/workflows/cd.yml` grants `contents: read`, `packages: write`, `id-token: write`, and `attestations: write`; builds images; scans with Trivy; generates CycloneDX SBOM through Trivy; pushes tags; resolves digest; installs cosign; signs by digest; attests build provenance with `actions/attest-build-provenance`; and uploads the SBOM artifact.
- `.github/workflows/ci.yml` and `.github/workflows/nightly.yml` use SHA-pinned actions and image scanning gates.
- `build/docker/Dockerfile` pins base images by digest.

## Decision Rubric
- Publish jobs must sign and attest the immutable image digest, not only mutable tags.
- Release tags must run `release-preflight` before `publish-release`; publish without successful preflight is blocked.
- SBOM artifacts must fail closed when missing, for example `if-no-files-found: error`.
- Jobs that do not publish or attest should keep read-only permissions; write scopes belong only where required for package publish, OIDC signing, or attestations.
- Consumers/deployers that verify release trust must check attestation identity, source repository/ref, subject digest, and signer/workflow identity before accepting an artifact.
- Base images and third-party actions should be pinned to immutable digests or SHAs with an update policy, not floating names alone.

## Imitate
- "Resolve `ghcr.io/<repo>:vX.Y.Z` to a digest, scan the image, sign `<image>@<digest>`, attest that digest, and upload `sbom-vX.Y.Z` with missing-file failure." Copy digest-bound evidence ordering.
- "Publish permissions are job-scoped to package write, OIDC, and attestations; normal CI remains read-only." Copy least-permission posture.
- "Deployment acceptance requires verification output for attestation identity, source repo/ref, and subject digest." Copy the consumer-facing proof.

## Reject
- "Sign `latest` after pushing." Mutable tags are not the trust subject.
- "Generate SBOM after publish without tying it to the released digest." This loses artifact identity.
- "Use `write-all` at workflow scope." This expands blast radius without improving release proof.
- "Attestation exists, so trust is solved." Attestation needs a verifier and acceptance policy.

## Agent Traps
- Do not rebuild after scanning or attesting and then publish the rebuilt artifact.
- Do not make SBOM/provenance optional artifacts.
- Do not define the cryptographic trust model or vulnerability acceptance here; route those to security/platform ownership while recording delivery proof needs.

## Validation Shape
Use CD workflow run URL and commit SHA, release preflight and publish job conclusions, published image tags and immutable digest, cosign signing log for the digest, build-provenance attestation log/id, SBOM artifact name/file, and verification output from `gh attestation verify`, `slsa-verifier`, cosign, or the deployment platform's verifier.

## Hand-Off Boundary
Do not define vulnerability acceptance, cryptographic trust model, or deployment admission architecture here beyond delivery proof requirements. Route unresolved security policy to the security spec and unresolved platform admission design to platform architecture.
