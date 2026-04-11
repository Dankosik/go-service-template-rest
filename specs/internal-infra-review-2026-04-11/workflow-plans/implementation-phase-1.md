# Implementation Phase 1

## Phase

- Current phase: implementation.
- Status: completed.
- Execution shape: lightweight local.
- Readiness: WAIVED for this narrow follow-up because the review findings are concrete, local, and already accepted by the user.

## Scope

- Fix accepted findings in `internal/infra/http`, `internal/infra/postgres`, `internal/infra/telemetry`, and `env/migrations`.
- Add or update focused regression tests for changed behavior.
- Do not broaden into unrelated cleanup or generated-code hand edits.

## Steps

- HTTP: fix response status tracking, `ResponseWriter` unwrapping, generated problem model use, and `Allow: OPTIONS` disclosure.
- Postgres: wrap empty DSN config error, detach rollback cleanup from canceled request context, remove unused producer-owned interface, and add the matching recent-history index migration.
- Telemetry: preserve scheme-less OTLP host:port endpoint parsing.
- Verification: run targeted tests as each block lands, then scoped package verification.

## Stop Rule

Stop after all accepted findings are implemented or after a blocker exposes a missing design decision. Do not start unrelated refactors.

Completion status: complete. All accepted findings were implemented.

## Validation Evidence

- `go test ./internal/infra/... -count=1` passed.
- `go test -race ./internal/infra/... -count=1` passed.
- `go vet ./internal/infra/...` passed.
- `go test -tags=integration ./test -run 'PingHistory' -count=1 -v` passed.
- `go test ./... -count=1` passed.
- `go vet ./...` passed.
