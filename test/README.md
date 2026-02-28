# Integration Tests

Store end-to-end/integration tests and large test fixtures in this directory.

Integration tests use the `integration` build tag and are not executed by default.

Run locally:

```bash
make test-integration
```

This requires a working Docker daemon (used by `testcontainers-go`).
