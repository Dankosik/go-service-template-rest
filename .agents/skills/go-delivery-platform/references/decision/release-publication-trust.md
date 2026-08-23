# Release Publication Trust

## Load When
Load for GHCR publish policy, `workflow_run` triggers, signing and attestation, SBOM binding, workflow permissions, or a release-trust claim such as a SLSA level.

## Decide
- Publishing is opt-in behind the `ENABLE_GHCR_PUBLISH` repository variable. A workflow that looks correct still publishes nothing until that variable is `true`; absence of images is not evidence of a broken pipeline.
- The `publish` job accepts `workflow_run` of `ci` only when upstream conclusion is `success`, upstream event is `push`, the run came from this repository, the branch is the default branch, and `head_sha == github.sha`. It then waits for the exact-SHA main CodeQL run. Tag events wait for their own full exact-SHA CI and CodeQL runs, so release integration is never path-skipped.
- Checkout pins `ref: github.event.workflow_run.head_sha`. A privileged `workflow_run` job that checks out anything else consumes code the upstream run never gated.
- Workflow-scope permission is `contents: read`; `packages`, `id-token`, and `attestations` writes are scoped to the publish job.
- The single publish job uses `concurrency: migration-publication-${{ github.repository }}` with `cancel-in-progress: false`, so migration-bearing publications never race. Step order advances the verified history marker before public aliases.
- Signing and attestation bind to the resolved digest, not a tag: `cosign sign` on `<repo>@<digest>`, then `actions/attest` twice — build provenance and the CycloneDX SBOM — against the same `subject-digest`. The workflow then re-verifies with `cosign verify` and `gh attestation verify` before it finishes.
- Provenance generated inline by this repository's own workflow is SLSA v1.0 Build **L2**. L3 requires a reusable workflow that isolates build logic from the caller. Claim L2.
- A tag publish watches the tag-scoped push runs for CI and CodeQL to successful completion before checkout or publication.

## Inspect
- "Resolve the pushed tag to a digest, sign and attest that digest, and let the workflow's own `gh attestation verify` be the passing evidence." Copy the digest-as-subject habit.

## Reject
- "Sign `:main` after pushing." A mutable tag is not the trust subject; the digest is.
- "Move the write permissions up to workflow scope so both jobs can share them." That widens the token blast radius to every job for no added proof.

## Reopen
- `actions/attest-build-provenance` is now a thin wrapper over `actions/attest`, which this repository already calls directly; migrating "up" to the wrapper is backwards.
- Rebuilding an image after the scan or attestation step publishes an artifact whose digest nothing verified.

## Prove
Use the CD run URL, the exact-SHA CI and CodeQL run IDs, the publish job conclusion, the resolved digest, the cosign and `gh attestation verify` step output, and the SBOM artifact name.
