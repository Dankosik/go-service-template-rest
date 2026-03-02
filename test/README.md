# Integration Tests

Store end-to-end/integration tests and large test fixtures in this directory.

Integration tests use the `integration` build tag and are not executed by default.

Run locally:

```bash
make test-integration
# zero-setup mode:
make docker-test-integration
```

Local behavior:
- if Docker daemon is unavailable, integration tests are skipped.

CI behavior:
- workflows set `REQUIRE_DOCKER=1`, so Docker unavailability fails the job.
