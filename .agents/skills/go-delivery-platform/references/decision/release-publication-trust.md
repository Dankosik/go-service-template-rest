# Release Publication Trust

## Load When
Load for GHCR publish policy, `workflow_run` triggers, signing and attestation, SBOM binding, workflow permissions, or a release-trust claim such as a SLSA level.

## Decide
- Publishing is opt-in behind the `ENABLE_GHCR_PUBLISH` repository variable. A workflow that looks correct still publishes nothing until that variable is `true`; absence of images is not evidence of a broken pipeline.
- `publish-main` triggers on `workflow_run` of `ci`, and each clause of its condition does distinct work: upstream conclusion `success`, upstream event `push`, `head_repository.full_name == github.repository` rejects fork-originated runs, `head_branch == default_branch` scopes it, and `head_sha == github.sha` requires the built commit to still be the default-branch tip. Per GitHub, `github.sha` for `workflow_run` is the last commit on the default branch — so widening this job to other branches while keeping that clause silently publishes nothing.
- Checkout pins `ref: github.event.workflow_run.head_sha`. A privileged `workflow_run` job that checks out anything else consumes code the upstream run never gated.
- Workflow-scope permission is `contents: read`; `packages`, `id-token`, and `attestations` writes are already scoped to the two publish jobs. This is the finished state, not a migration to perform.
- Both publish jobs share `concurrency: migration-publication-${{ github.repository }}` with `cancel-in-progress: false`, so a publish never races a migration-bearing publish. `scripts/ci/migration-publication-check.sh` fails when either job loses that group or promotes a public alias before the history marker.
- Signing and attestation bind to the resolved digest, not a tag: `cosign sign` on `<repo>@<digest>`, then `actions/attest` twice — build provenance and the CycloneDX SBOM — against the same `subject-digest`. The workflow then re-verifies with `cosign verify` and `gh attestation verify` before it finishes.
- Provenance generated inline by this repository's own workflow is SLSA v1.0 Build **L2**. L3 requires a reusable workflow that isolates build logic from the caller. Claim L2.
- `publish-release` requires `release-preflight`; a tag publish without it has no preflight evidence.

## Inspect
- "Resolve the pushed tag to a digest, sign and attest that digest, and let the workflow's own `gh attestation verify` be the passing evidence." Copy the digest-as-subject habit.

## Reject
- "Sign `:main` after pushing." A mutable tag is not the trust subject; the digest is.
- "Move the write permissions up to workflow scope so both jobs can share them." That widens the token blast radius to every job for no added proof.

## Reopen
- `actions/attest-build-provenance` is now a thin wrapper over `actions/attest`, which this repository already calls directly; migrating "up" to the wrapper is backwards.
- Rebuilding an image after the scan or attestation step publishes an artifact whose digest nothing verified.

## Prove
Use the CD run URL, the publish job conclusion, the resolved digest, the cosign and `gh attestation verify` step output, the SBOM artifact name, and the `release-preflight` conclusion for tag releases.
