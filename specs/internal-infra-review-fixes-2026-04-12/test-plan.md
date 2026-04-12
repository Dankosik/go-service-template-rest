# Test Plan

## Focused Checks

Run after Phase 1:

```sh
go test ./internal/observability/... ./internal/config ./internal/infra/telemetry
```

Run after Phase 2:

```sh
go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap
```

Run after Phase 3:

```sh
go test ./internal/infra/http -run 'TestServer'
```

Run after Phase 4:

```sh
go test ./internal/infra/postgres
rg createAndListRecentInTx internal/infra/postgres
```

The `rg createAndListRecentInTx` command should return no matches after T008; exit status 1 is expected for that no-match check.

Run after Phase 5:

```sh
go test ./internal/infra/http -run 'ManualRootRoute|RootRouter|OpenAPIRuntimeContract'
```

## Aggregate Checks

Run before completion:

```sh
go test ./internal/infra/...
go test ./internal/observability/... ./internal/config ./cmd/service/internal/bootstrap
git diff --check
```

If the Postgres integration environment is available, also run:

```sh
go test ./test -run TestPingHistoryRepositorySQLCReadWrite
```

## Manual Diff Checks

- Confirm `internal/infra/postgres/sqlcgen/*` is not edited.
- Confirm no new `common`, `shared`, or `util` package is introduced.
- Confirm `resource.WithFromEnv()` is absent from runtime code unless the spec was reopened.
- Confirm `SetupTracing` no longer owns fallback defaults for resource identity values that belong to config.
- Confirm metric names and labels in telemetry tests remain stable.
- Confirm `/metrics` route behavior remains stable.
- Confirm docs mention the `internal/observability/otelconfig` boundary without turning it into a generic observability package.
