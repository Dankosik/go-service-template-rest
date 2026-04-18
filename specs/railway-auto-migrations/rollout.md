# Trigger And Scope

This rollout note exists because automatic migrations intersect with deploy sequencing and mixed-version safety on Railway.

# Operator Prerequisites

1. Connect the Railway service to the intended GitHub repository and deployment branch.
2. Enable Railway `Wait for CI` if deploys should wait for push-triggered GitHub checks.
3. Ensure Railway variables provide the template's Postgres contract:
   - `APP__POSTGRES__ENABLED=true`
   - `APP__POSTGRES__DSN=<single-target DSN with explicit sslmode>`

# Release Sequence

1. Merge or push a change.
2. Railway builds the image from `build/docker/Dockerfile`.
3. Railway runs `/migrate` as the pre-deploy step.
4. Only after migration succeeds does Railway promote the new `/service` deployment.

# Safety Rules

- Same-deploy schema changes must be expand-compatible with the old deployment while Railway overlap is active.
- Do not combine destructive or contract-only schema steps with the first code deploy that still needs the old shape.
- If a migration needs backfill, cleanup, or irreversible contract work, reopen spec/design for a staged rollout instead of relying on the default one-step deploy.

# Failure / Recovery

- If `/migrate` fails, Railway blocks promotion and leaves the old deployment serving traffic.
- Recovery path: fix the migration/config issue and trigger a new deployment.
- Do not move the failed migration into app startup as a workaround; that would forfeit one-migrator ownership.
