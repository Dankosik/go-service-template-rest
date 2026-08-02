# PostgreSQL transactional outbox V1 implementation ledger

status: done

Completion: `OUTBOX=postgres` retains a canonical PostgreSQL outbox and a
separately runnable fail-closed relay whose fixed local repair candidate proves
atomic intent, leases/crash recovery, explicit at-least-once duplicates,
bounded failure/lifecycle/telemetry, real NATS acknowledgement, and physical
`OUTBOX=none` purity; the repair passes claim-scoped final validation, is
locally committed, and leaves clean task-owned status.

Blocked stop: stop without a completion claim if canonical Goose/sqlc authority
changes, a mandatory real PostgreSQL/NATS/Docker proof input becomes unavailable,
the initializer cannot remove every owned byte without harming another retained
profile, an independent review remains FAIL, or any required command remains
red. Record the exact failing command/state and reopen the narrow source owner.

Global constraints: preserve unrelated/sibling work; use the isolated
`codex/postgres-outbox-v1` worktree; no push/PR/deploy/managed infrastructure or
other external write; no broker adapter, inbox, domain-specific event,
exactly-once claim, runtime migration, noop Publisher, new module dependency,
CDC scaffolding, internal worker pool, or lease renewal; keep exact event bytes
and IDs across retries/redrive; serialize Docker and broad Go gates.

- [x] T1: The existing PostgreSQL transaction owner distinguishes definite and unknown commit outcomes without changing established `ErrTransaction` behavior.
  - Source: `design/overview.md` “Append” and “Existing transaction owner”; `test-plan.md` TD-06.
  - Owner/surface/resources: `internal/infra/postgres/postgres.go` and focused sibling/same-package integration tests; existing pgx pool and serialized `pgtest` PostgreSQL fixture.
  - Depends on: none.
  - Proof: callback and server-confirmed deferred-constraint failures remain `ErrTransaction` without `ErrCommitUnknown`; callback failure rolls back durable state; opaque post-commit result matches both and another connection observes the real commit; `go test -vet=off -count=1 ./internal/infra/postgres -run 'TestClassifyCommit|TestInTx'`, `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./internal/infra/postgres -run '^TestInTxCommitOutcomes$'`, and `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresInTxRollsBackOnError$'`.
  - Reopen if: pgx exposes a stronger canonical commit-outcome contract or the seam requires public/runtime configuration; Go Ownership Design. The generic `InTx` correction and focused tests are retained for `OUTBOX=none`; they are not profile-owned bytes.

- [x] T2: Canonical schema, exact envelope append, ordering authority, and token-fenced PostgreSQL store are complete and independently proven on PostgreSQL 17.
  - Source: `spec.md` OUT-2..4, OUT-7..8, OUT-11; `design/overview.md` “Canonical schema” and “SQL operations and concurrency”; `test-plan.md` TD-01..11, TD-16, TD-22, TD-26.
  - Owner/surface/resources: `migrations/000001_postgres_outbox.sql`; `internal/infra/postgres/queries/postgres_outbox.sql`; generated `internal/infra/postgres/sqlcgen`; `internal/infra/postgresoutbox/{event,store}.go` and focused tests; store-focused outbox integration tests under `test/`; canonical sqlc generator and one serialized PostgreSQL/Docker fixture.
  - Depends on: T1 — unknown-commit state/proof gate — needed to prove atomic append finality.
  - Proof: exact/boundary bytes, all atomic failure rows, ordered high-water, held-lock SKIP LOCKED, disjoint claims, lease expiry/fence, abandoned claim, redrive, cleanup, and migration rehearsal match durable SQL state; `go test -vet=off -count=1 ./internal/infra/postgresoutbox -run 'TestEvent|TestBackoff'`, `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox(Envelope|Atomicity|OrderingAuthority|ConcurrentClaims|OrderingClaims|LeaseExpiryAndFence|CrashAfterClaim|Redrive|Cleanup)$'`, `make sqlc-check`, `make migration-check`, and `make migration-validate`.
  - Reopen if: the first canonical migration version is no longer available or another template profile owns shared sqlc output; migration/data design.

- [x] T3: The one-at-a-time relay and telemetry implement bounded at-least-once publication, retries/poison, duplicate crash recovery, safe replicas, and supervised lifecycle.
  - Source: `spec.md` OUT-4..10; `design/overview.md` “Relay and acknowledgement sequence”, “Relay process and lifecycle”, and “Observability and recovery”; `test-plan.md` TD-12..23, TD-30.
  - Owner/surface/resources: `internal/infra/postgresoutbox/{publisher,relay,telemetry}.go` and focused tests; relay-focused integration tests under `test/`; existing OTel manual reader; serialized PostgreSQL fixture; deterministic test Publisher only.
  - Depends on: T2 — generated/store API and applied schema output handoff — needed to start.
  - Handoff: T2 supplies the fixed Event/Store/query surface and PostgreSQL schema consumed by Relay.
  - Proof: temporary/permanent/ambiguous failures, ack-crash duplicate with identical ID/bytes, replica completion, exhaustion/poison, idempotent redrive, cancellation, graceful/forced/stuck/panic paths, complete bounded telemetry, request-independent backlog, cleanup and liveness all hit their authoritative oracles; `go test -vet=off -count=1 ./internal/infra/postgresoutbox`, `go test -vet=off -race -count=1 ./internal/infra/postgresoutbox`, and `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox(PublishFailure|AckCrashDuplicate|RelayReplicas|RetryAndPoison|Observability)$'`.
  - Reopen if: one process needs more than one in-flight publish to meet a measured accepted drain/pickup budget; performance/concurrency design.

- [x] T4: The relay composition, validated config, image/commands/docs, and `OUTBOX=none|postgres` initializer form one complete structurally optional profile.
  - Source: `spec.md` OUT-1, OUT-9..11; `design/overview.md` “Relay process”, “Configuration”, “Profile and generated ownership”, and “Go code and file ownership”; `test-plan.md` TD-18..19, TD-23..24, TD-27..30.
  - Owner/surface/resources: `internal/config` outbox-marked type/default/validation/tests; `cmd/outbox-relay`; `env/.env.example`; `Makefile`; `build/docker/Dockerfile`; affected CI/image assertions; `scripts/init-module.sh`; `scripts/ci/template-init-check.sh`; `template.lock` writer/fixtures; `docs/postgres-transactional-outbox.md` and narrow doc indexes; all accepted outbox path removal entries; serialized Docker/image/template fixtures.
  - Depends on: T3 — fixed Relay/Publisher/lifecycle surface output handoff — needed to start.
  - Handoff: T3 supplies the concrete relay, minimal Publisher, telemetry, and cleanup-safety result composed by the binary and enumerated by the profile.
  - Proof: nil builder/publisher reject before mutation, config budget matrix, readiness/liveness/drain/panic/stuck cleanup order, retained image binary, positive retained profile, total none purity, invalid combination no-mutation, marker removal, lock choice, and same-choice byte stability; `go test -vet=off -count=1 ./internal/config ./cmd/outbox-relay/internal/bootstrap`, `go test -vet=off -race -count=1 ./cmd/outbox-relay/internal/bootstrap`, `make template-init-check`, and `make runtime-image-build`.
  - Reopen if: another profile introduces a migration/sqlc owner before initialization and current outbox-generated path deletion becomes ambiguous; profile/generated ownership design.

- [x] T5: Real request-path outage isolation and accepted JetStream event-ID/durable-ack conformance close the integrated local capability without creating a production adapter.
  - Source: `spec.md` OUT-3, OUT-5, success criteria 3..4; `design/overview.md` deployment graph and Publisher boundary; `test-plan.md` TD-13, TD-20, TD-25.
  - Owner/surface/resources: integration-only `httptest` feature transaction and deterministic Publisher fixture; `test/postgres_outbox_natsjs_integration_test.go` test-only mapping; both outbox/messaging profile removal lists; serialized PostgreSQL and NATS JetStream Docker fixtures.
  - Depends on: T4 — complete retained/removed profile and runnable relay image state — needed to prove.
  - Proof: HTTP requests continue committing equal feature/outbox rows while publication fails; real JetStream durable message payload and both accepted NATS identities equal the outbox event after Publisher nil; no production adapter appears in Go dependency/wiring; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox(RequestContinuesDuringBrokerOutage|NATSConformance)$'` and `make template-init-check`.
  - Reopen if: the accepted NATS adapter changes its durable acknowledgement or identity contract; messaging integration owner.

- [x] T6: The fixed implementation candidate is independently accepted, passes final serialized validation, is locally committed once, and leaves no task-owned dirt.
  - Source: `spec.md` success criteria; `test-plan.md` “Commands and gate composition”; `workflow-plan.md` completion proof.
  - Owner/surface/resources: the complete task-owned diff and workflow artifacts; independent read-only implementation reviewer; repository Go/Docker caches and Git index/commit; no external writes.
  - Depends on: T1, T2, T3, T4, T5 — all accepted outputs and focused proof gates — needed to start.
  - Proof: independent implementation review returns PASS on the fixed tree; then, without overlapping broad gates, run `go test -vet=off -count=1 ./internal/infra/postgres ./internal/infra/postgresoutbox ./internal/config ./cmd/outbox-relay/internal/bootstrap`, `go test -vet=off -race -count=1 ./internal/infra/postgres ./internal/infra/postgresoutbox ./cmd/outbox-relay/internal/bootstrap`, `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox'`, `make sqlc-check`, `make migration-check`, `make migration-validate`, `make template-init-check`, `make check-full`, and `git diff --check`; inspect the commit diff/status, create one local commit, and require `git status --short` empty.

- [x] T7: The accepted performance-audit repair closes JSON admission parity, fatal-attempt exhaustion, maintenance/readiness ownership, stable PostgreSQL eligibility cutoffs, and UTF8/pool fail-closed admission without changing at-least-once, ordering, Publisher, or deployment ownership.
  - Source: `spec.md` OUT-2, OUT-4..6, OUT-9..10; `design/overview.md` “Canonical schema”, “Claim”, “Retry, poison, redrive, cleanup, observation”, and “Relay process and lifecycle”; `test-plan.md` TD-01, TD-15, TD-23, TD-26, TD-29..30; accepted audit repair request.
  - Owner/surface/resources: additive migration and canonical outbox query/sqlc output; `internal/infra/postgresoutbox/{store,relay}.go`; outbox config admission; focused component/integration tests, profile removal manifests, and docs; one serialized PostgreSQL/Docker fixture.
  - Depends on: T6 — fixed audited candidate identity — needed to keep the repair boundary explicit.
  - Proof: escaped-NUL/arbitrary-number parity and non-UTF8 rejection; only adapter-proven non-acceptance reaches poison at `MaxAttempts`, while ack-crash, timeout, panic, stuck, and progress ambiguity preserve retry or duplicate recovery at the threshold; slow publication retains fresh readiness and fatal fan-in clears it before sibling join; blocked cleanup does not block publication; one-connection relay config is rejected; 100k future/recent-row `EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)` uses bounded index access; then run the existing T6 serialized gate set, inspect the repair diff, commit locally, and require clean status.
  - Acceptance: PASS — reviewer: `/root/t7_safe_exhaustion_acceptance` (fresh independent critical review, read-only); evidence: focused/unit/race/PostgreSQL integration proof, generated/migration/profile drift, serialized template matrix, exact-tree `make check-full`, bounded PostgreSQL plans, and repeated local A/B stress measurements passed; candidate: accepted semantic unit at source fingerprint `b14cb0ffed764e08838a2039f9896b2b781001d5a6b34b0246350ddaa2772bb0`, followed only by `gofumpt` and lint-equivalent `if`-to-`switch` closure at source fingerprint `e0e875b727f42aa4615264b1ef67db6c7d3c579e530b293a2bf944239f02e58e`.
  - Reopen if: the bounded plan loses its index access, the maintenance loop can outlive `Run`, fatal recovery publishes or increments past the limit, or the new migration cannot rehearse Up/Down/Up; query/data/concurrency owner.

## Reconciliation

- Profile selection/purity and runtime admission: T4, with cross-profile broker
  test removal closed by T5.
- Exact event/atomic transaction/commit classification: T1 and T2, with the
  request-process observable closed by T5.
- Claim, lease, ordering, retry, poison, redrive, retention and data growth: T2
  and T3.
- Publisher acknowledgement, duplicates and broker outage: T3 and T5.
- Process lifecycle, readiness and no-noop composition: T3 and T4.
- Migration/sqlc/generated/image/rollout documentation: T2 and T4.
- Every mandatory proof and completion/local-commit boundary: earliest owning
  task above, then original fixed-candidate closeout T6 and accepted audit repair
  closeout T7.

No parallel wave is planned: T2 consumes T1 transaction semantics; T3 consumes
T2 generated/store APIs; T4 enumerates T3 runtime/config/profile surfaces; T5
requires the complete retained/removed composition; T6 consumes every accepted
output. Running them concurrently would create generated, test-fixture,
initializer, or interface assumption conflicts without a valid handoff.
