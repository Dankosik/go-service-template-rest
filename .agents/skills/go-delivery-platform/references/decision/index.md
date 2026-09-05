# Reference Selector

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Required status checks, branch protection or ruleset contexts, merge queue, change-scope skips, or a green check that never compared anything | [required-checks-and-change-scope.md](required-checks-and-change-scope.md) | Require the one aggregate context instead of a job list that change scope can strand. |
| A release carrying a schema migration: sequencing, rollback class, backfill, mixed-version safety, or migrator ownership | [migration-rollout-window.md](migration-rollout-window.md) | Size the window in which the previous version runs against the new schema. |
| GHCR publish, `workflow_run` triggers, signing, attestation, SBOM binding, workflow permissions, or a SLSA claim | [release-publication-trust.md](release-publication-trust.md) | Keep each trust and freshness clause load-bearing and bind proof to the digest. |

Containers, generated-artifact drift, and Railway runtime knobs have no reference here: `build/docker/Dockerfile`, the `openapi-*` and `sqlc-check` Makefile targets, and `railway.toml` are short, current, and self-describing. Read them.

Prefer live repository and managed-platform configuration as source-of-truth evidence; references sharpen a triggered decision but do not replace current facts.
