# PostgreSQL transactional outbox V1 implementation ledger

status: done

Completion target: retain the accepted capability while closing every proven
actionable finding from the fixed-candidate PostgreSQL/performance/architecture
audit. T1-T6 record the original candidate; T7 owns the remediation and its new
proof without expanding into CDC, broker ownership, partitioning, or an internal
worker pool.

Blocked stop: stop without a completion claim if canonical Goose/sqlc authority
changes, a mandatory real PostgreSQL/NATS/Docker proof input becomes unavailable,
the initializer cannot remove every owned byte without harming another retained
profile, an independent review remains FAIL, or any required command remains
red. Record the exact failing command/state and reopen the narrow source owner.

Global constraints: preserve unrelated/sibling work; use the isolated
`codex/postgres-outbox-v1-audit-fixes` worktree; no push/PR/deploy/managed infrastructure or
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

- [x] T7: The audit remediation preserves every accepted invariant while removing the measured ordering-query collapse and closing commit classification, JSON, readiness, cleanup, shutdown, and observation gaps.
  - Source: fixed-candidate audit against `16bba2b152e0772ea978a38f7e5391318fa928a7`; `design/overview.md` ordering, transaction, lifecycle, and observation decisions.
  - Owner/surface/resources: canonical migration/query/sqlc output; PostgreSQL transaction classifier; outbox store/relay/telemetry; relay bootstrap; focused and real-PostgreSQL tests; operations documentation; isolated local benchmark artifacts only.
  - Depends on: T6 — the immutable audited candidate and its original proof are the comparison baseline.
  - Proof: SQLSTATE `08007`/`40003` remain unknown; exact Go-valid JSON edge bytes persist; concurrent ordered append/finalize cannot lose the next head; claims and observations use materialized state and stable statement time; periodic observation, readiness, cleanup catch-up, redrive storage, and Publisher cleanup are bounded; focused/race/integration/sqlc/migration/purity/full gates pass serially; repeated PostgreSQL plans and workload measurements beat or bound the audited failure cases without weakening at-least-once recovery.
  - Receipt: focused Go and race packages passed; the exact-tree real-PostgreSQL `^TestPostgresOutbox` suite passed; `make sqlc-check`, `make migration-check`, `make migration-validate`, `make template-init-check`, `make check-full`, and `git diff --check` passed serially. The external benchmark manifest records exact candidate/fixed identities, query plans, relay/request/maintenance matrices, three 1M hot-key migration runs, three 1M-key migration runs, and their dispersion. Fresh independent acceptance returned PASS on the frozen diff.
  - Reopen if: materialized head maintenance can diverge under a demonstrated transaction interleaving, the ordered finalization round trips violate an accepted workload budget, or production observation shows connection, vacuum, or cleanup pressure outside the measured local envelope.

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
- Every mandatory proof and original completion/local-commit boundary: earliest
  owning task above, then fixed-candidate closeout T6 and audit-remediation
  acceptance T7.

No parallel wave is planned: T2 consumes T1 transaction semantics; T3 consumes
T2 generated/store APIs; T4 enumerates T3 runtime/config/profile surfaces; T5
requires the complete retained/removed composition; T6 consumes every accepted
output. T7 is a later serialized acceptance unit against the immutable T6
candidate. Running implementation and generated/test-fixture gates concurrently
would create ownership conflicts without a valid handoff.
