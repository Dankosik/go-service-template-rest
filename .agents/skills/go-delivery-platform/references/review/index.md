# Reference Selector

| Symptom | Load |
| --- | --- |
| Required contexts, branch protection or rulesets, merge queue, job conditions, or change-scope routing changes. | [required-checks-and-change-scope.md](../decision/required-checks-and-change-scope.md) |
| Migration files, `preDeployCommand`, overlap/draining windows, backfill, or rollback-class changes. | [migration-rollout-window.md](../decision/migration-rollout-window.md) |
| Publish conditions, `workflow_run` guards, workflow permissions, signing, attestation, or SBOM changes. | [release-publication-trust.md](../decision/release-publication-trust.md) |

A changed Dockerfile or generated artifact is reviewed against the file itself and the gate that enforces it; a changed Railway service setting is reviewed against `docs/railway-deployment-profile.md` and the live service settings. No reference stands between them.
