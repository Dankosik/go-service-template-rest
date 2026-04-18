# Runtime / Release Sequence

1. A commit lands on the GitHub branch connected to the Railway service.
2. Railway detects the push and, when the operator enabled `Wait for CI`, waits for push-triggered GitHub workflows to finish successfully before continuing.
3. Railway builds the image from `build/docker/Dockerfile`.
4. Before promoting the new deployment, Railway runs the configured pre-deploy command in a separate container with the service environment and private-network access.
5. `/migrate` loads the template config namespace, checks whether Postgres is enabled, and if enabled applies `env/migrations/` against `APP__POSTGRES__DSN`.
6. If migration fails, Railway does not promote the new deployment and the previous deployment stays active.
7. If migration succeeds or Postgres is disabled, Railway starts the new `/service` deployment and promotes it only after `/health/ready` passes under the existing overlap/draining policy.

# Failure And Compatibility Notes

- Migration execution is single-owner per deployment, but schema changes still have to tolerate the old deployment during Railway overlap.
- This design intentionally does not move DDL into service startup, because retries, restarts, and multi-replica starts would create concurrent migrators.
