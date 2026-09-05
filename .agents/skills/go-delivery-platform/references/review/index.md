# Reference Selector

| Symptom | Load |
| --- | --- |
| Required contexts, branch protection or rulesets, merge queue, job conditions, or change-scope routing changes. | [required-checks-and-change-scope.md](../decision/required-checks-and-change-scope.md) |
| Migration files, `preDeployCommand`, overlap/draining windows, backfill, or rollback-class changes. | [migration-rollout-window.md](../decision/migration-rollout-window.md) |
| Publish conditions, `workflow_run` guards, workflow permissions, signing, attestation, or SBOM changes. | [release-publication-trust.md](../decision/release-publication-trust.md) |

A changed Dockerfile, generated artifact, or `railway.toml` knob is reviewed against the file itself and the gate that enforces it; no reference stands between them.
