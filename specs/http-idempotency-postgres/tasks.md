# PostgreSQL-backed HTTP idempotency implementation ledger

status: ready

Completion: the reusable `HTTP_IDEMPOTENCY=postgres` pack is implemented in the
health-only template, every repository-local TD-IDEM-001 through TD-IDEM-015A
oracle passes on its accepted carrier, `none|postgres` generation is deterministic
and dependency-clean, and the existing production-image/migration aggregate proves
both the health-only and active-registration source identities. No completion claim
includes a concrete endpoint, live writer/failover topology, restore, activation, or
external effect.

Blocked stop: leave the affected task unchecked and record the unavailable claim,
current narrower evidence, and reopen owner. Reopen Technical Design when a
required deterministic carrier or fixed owner/file boundary is infeasible. Reopen
Specification only for the recorded R2 raw-framing-OWS, R3 authorization-input, R6
zero-cross-replica-wait, or R9 exact-commit-epoch behavioral falsifier. Missing
adopter inputs do not block this reusable-template ledger; they remain the
TD-IDEM-016 scope exit below.

Global constraints:

- HTTP transport validates, carries generated operation identity, maps classified
  outcomes, and freshly renders retained semantic results; it never begins,
  commits, retries, or otherwise owns the feature transaction.
- The endpoint/application adapter calls the existing `postgres.Pool.InTx` once.
  Acquire, feature mutation, optional outbox append, result encoding/bound check,
  and completion share that exact caller-owned `pgx.Tx`; completion is the final
  application statement and baseline execution has no hidden retry.
- Initial arbitration, any absence that may authorize execution, replay,
  reconciliation, expiry, cleanup eligibility, and commit epoch use only the
  current authoritative writer. The process-local publication group carries only
  a completion signal and never becomes correctness authority.
- The reusable guarantee ends at the exact PostgreSQL commit. Direct outbound
  effects remain forbidden from the covered sequence; a concrete endpoint owns
  outbox/downstream idempotency, reconciliation, or compensation and its final
  result semantics.
- The 20-row `test-plan.md` matrix is the proof authority. Implementation may add
  its named tests and fixtures but does not add scenarios, substitute mock-only
  proof for PostgreSQL arbitration/visibility, replace owned event gates with
  sleeps, or add a production fault seam.
- No numeric endpoint or deployment value becomes a template default. No new
  dependency, deployable, cache, broker, scheduler, cleanup binary, elected leader,
  store interface, or fake OIDC trust path is introduced.
- Canonical OpenAPI, migration, and SQL sources precede generated output. The
  health-only image and disposable active-registration image are distinct source
  identities, each built once and reused for only its accepted claims.

## Obligation reconciliation

| Accepted obligation | Ledger disposition |
| --- | --- |
| R1 complete fail-closed opt-in and no generic guarantee | T1 owns declaration/registration rejection; T3 owns exact empty/nonempty runtime activation; T4 owns removable profile closure. |
| R2 exact opaque wire key and bad-key action | T1. |
| R3 authenticated scope, current authorization, admission, and response precedence | T1 owns the HTTP order and authority seam; T2 consumes the resulting scoped identity only on the writer. |
| R4 recorded-version semantic fingerprint | T1 owns canonical/versioned value contracts; T2 owns persisted-version comparison and writer replay. |
| R5 exact caller-owned atomic boundary | T2 owns reusable acquire/mutation/optional-outbox/result/completion composition. Concrete endpoint external-effect recovery is the TD-IDEM-016 scope exit. |
| R6 one owner, bounded classification, rollback/death, and whole-attempt retry policy | T2. |
| R7 bounded original semantic replay with fresh transport metadata | T1 owns the envelope/renderer contract; T2 owns committed result authority and fresh-process replay. |
| R8 bound-before-commit and every authoritative ambiguity branch | T2. |
| R9 exact commit horizons, retention, erasure, and cleanup | T2 owns exact epoch capture and fail-closed loss; T3 owns maintenance, capacity closure, and lifecycle. Live restore/activation is the TD-IDEM-016 scope exit. |
| R10 stable Problem/client actions and disclosure boundary | T1 owns the catalog and HTTP mapping; T2 and T3 supply only closed Store/lifecycle dispositions. |
| R11 writer-only correctness | T2 owns every repository-local writer/non-writer falsifier. Live promotion/failover certification is the TD-IDEM-016 scope exit. |
| R12 privacy, telemetry, and cached readiness | T1 owns HTTP/result redaction; T3 owns closed telemetry, aggregate observations, and readiness. Adopter privacy/erasure/restore authority is the TD-IDEM-016 scope exit. |
| R13 profile, migration, mixed-version safety, and rollback | T3 owns zero-registration runtime behavior; T4 owns generated-profile purity; T5 owns additive migration and production-image rejection/compatibility proof. Activation remains the TD-IDEM-016 scope exit. |

## Matrix reconciliation

| Test Design row | Ledger disposition |
| --- | --- |
| TD-IDEM-001 | T1. |
| TD-IDEM-002 | T1. |
| TD-IDEM-003 | T1 owns canonical vectors, formatting/default equivalence, one mismatch per declared semantic-manifest field, and codec rules; T2 owns recorded-fingerprint-version persistence/replay. |
| TD-IDEM-004 | T1 owns HTTP authorization/admission ordering; T2 consumes its authority-A/B setup in pool-headroom proof. |
| TD-IDEM-005, TD-IDEM-005A | T2. |
| TD-IDEM-006, TD-IDEM-007 | T2. |
| TD-IDEM-008 | T1 owns fresh replay rendering; T2 owns bounded commit/replay and post-commit process death. |
| TD-IDEM-009 | T2. |
| TD-IDEM-010 | T2 owns exact epoch capture/materialization; T3 owns epoch-loss task failure, cached readiness, and drain. |
| TD-IDEM-011, TD-IDEM-011A | T3. |
| TD-IDEM-012 | T2 owns the local writer/non-writer oracle; the target failover procedure is the TD-IDEM-016 scope exit. |
| TD-IDEM-013 | T1. |
| TD-IDEM-013A, TD-IDEM-014 | T3. |
| TD-IDEM-015 | T4. |
| TD-IDEM-015A | T5. |
| TD-IDEM-016 | Scope exit: `spec.md` limits this bundle to a reusable health-only capability and `test-plan.md` makes endpoint quantities, real issuer, writer/failover topology, privacy/restore authority, traffic gate, downstream oracle, and external-effect recovery adopter-owned. A concrete adopting endpoint/deployment reopens those named endpoint, database, privacy, and deployment owners; only its recorded R2, R3, R6, or R9 behavioral falsifier reopens Specification. It creates no reusable-template implementation task or completion dependency. |

## Current-surface reconciliation

- `internal/infra/postgres/transaction.go` remains the sole caller transaction
  owner; T2 composes through `Pool.InTx` and retains
  `TestInTxCommitOutcomes` as the independent lost-result boundary.
- `internal/infra/postgresoutbox/store_append.go` remains the optional
  transaction-bound durable-intent owner, and
  `internal/infra/postgresoutbox/store_receipt.go` remains unchanged evidence for
  writer-only reconciliation. T2 may exercise Append in the synthetic transaction;
  no idempotency package owns publication or a downstream effect.
- `internal/infra/http/router.go` and `api/openapi/service.yaml` are the current
  envelope/contract producers and T1 couples them to their generated binding and
  focused tests. `request_errors.go` remains the unchanged owner of earlier
  generated validation/authentication rejection.
- `internal/background`, cached `internal/health`, and the shutdown sequence in
  `cmd/service/internal/bootstrap/run.go` remain lifecycle authorities; T3 only
  registers the conditional probe/task and preserves their existing drain/join.
- `/migrate`, canonical `migrations/`, PostgreSQL query sources, and SQLC output
  retain their current authority order. T2 adds one additive no-backfill relation;
  T5 proves the service never migrates at startup and old/profile-off code remains
  compatible while opted traffic is gated.
- `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, existing profile
  markers/inventories, `Makefile`, CI change-scope routing, and the runtime-image
  owner remain the only template/build surfaces. T4 and T5 consume the complete
  fixed T1-T3 path set; no second generator or migration/image harness is added.

## Readiness review

Independent Task Review / Readiness returned **PASS** on candidate SHA-256
`85a1dc163a81e0093704a27e4881811337d180f74d9c436681cb115cc46a0986` after one
Planning-owned repair restored TD-IDEM-003 formatting/default equivalence and one
mismatch per semantic-manifest field to T1. The focused fresh review found no
surviving finding. The T1 dry run reached acceptance from current owners and named
proof without a database, external input, chat history, or hidden choice; later
tasks are dependency-ordered and cannot invalidate that result. This receipt proves
ledger readiness only, not implementation, post-implementation tests, production
topology, endpoint adoption, or activation.

- [x] T1: The complete reusable declaration and HTTP envelope fail closed before Store work and render only bounded, versioned, redacted outcomes
  - Source: `spec.md` R1-R4, R7, R10, and R12; `design/overview.md` Exact identity, fingerprint, and result representations, HTTP envelope and opt-in closure, Go responsibility map, and inverse Go file map; `test-plan.md` TD-IDEM-001-TD-IDEM-004, the HTTP delta of TD-IDEM-008, and TD-IDEM-013.
  - Owner/surface/resources: keep `internal/httpidempotency/{doc.go,identity.go,result.go,outcome.go}`, change `internal/httpidempotency/contract.go` and its tests, and add `internal/httpidempotency/reservation.go`; add `internal/infra/http/{middleware_idempotency.go,idempotency.go,idempotency_registration.go,idempotency_response.go}` and their fixed tests; change `internal/infra/http/{router.go,doc.go,router_contract_test.go,openapi_contract_test.go}`; keep `internal/infra/http/request_errors.go` and its tests unchanged; change `internal/problem/{problem.go,problem_test.go}`, canonical `api/openapi/service.yaml`, and generated `internal/openapi/openapi.gen.go`; standard library and existing OpenAPI/HTTP owners only; no mutable resource.
  - Depends on: none.
  - Handoff: accepted versioned `Contract`, `DuplicateRiskPolicy`, `Scope`, `Fingerprint`, `Attempt`, `Reservation`, `ReservationRecovery`, `Result`, and `Decision` values; exact identity/result codecs; one-to-one operation registration; authenticated authorization/admission ordering; and closed Problem/render mapping consumed by T2 and T3. T1 owns the shared carrier declarations; T2 owns their Store round-trip, stale-generation, and recovery behavior proof under TD-IDEM-006, TD-IDEM-007, and TD-IDEM-009.
  - Proof: TD-IDEM-001 complete registration alone serves and every missing/duplicate/version/external-effect mutation fails before Store construction: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyRegistrationContract$' -count=1`. TD-IDEM-002 proves the exhaustive `tchar`/field-line/byte-bound oracle and zero downstream calls: `go test -vet=off ./internal/httpidempotency -run '^TestHTTPIdempotencyKeyParser$' -count=1` and `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyKeyContract$' -count=1`. TD-IDEM-003 pins both canonical vectors, rejects every excluded transport/header field, proves formatting/default equivalence through a synthetic typed canonicalizer, and produces one mismatch for each declared semantic-manifest field: `go test -vet=off ./internal/httpidempotency -run '^TestHTTPIdempotencyCanonicalVectors$' -count=1`. TD-IDEM-004 proves 431, 401, 403, key 400, authority admission 429/503, then retained-state ordering with scope isolation: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyAuthorizationAndAdmissionOrder$' -count=1`. The HTTP half of TD-IDEM-008 proves original semantic rendering with fresh transport metadata: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyReplayRendering$' -count=1`. TD-IDEM-013 proves every Problem status/code/type/title, required or forbidden `Retry-After`, fresh correlation, and sentinel absence: `go test -vet=off ./internal/problem -run '^TestHTTPIdempotencyProblemCatalog$' -count=1` and `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyProblemAndRedaction$' -count=1`. Canonical/generated OpenAPI remains identical under `make openapi-check`.
  - Reopen if: raw framing OWS must remain distinguishable after `net/http` normalization — Specification R2; current client-visible authorization needs body or external input before key validation — Specification R3; otherwise a required declaration, excluded field, or closed disposition cannot be observed/mapped through the fixed envelope — Technical Design.
  - Accepted: T1; evidence: TD-IDEM-001-TD-IDEM-004, HTTP TD-IDEM-008, TD-IDEM-013, and `make openapi-check` passed on the bounded candidate; fresh independent acceptance review PASS; handoff: `DuplicateRiskPolicy`, `Reservation`, and `ReservationRecovery` remain shared declarations only, while T2 stays blocked for Store round-trip and recovery proof.

- [x] T2: One writer-authoritative row arbitrates one caller-owned transaction and deterministically resolves reservation, rollback, death, replay, ambiguity, and exact commit epoch
  - Source: `spec.md` R3-R9 and R11; `design/overview.md` Selected authority and state model, PostgreSQL request protocol and exact transaction composition, literal authoritative-commit horizons, failure schedules, and PostgreSQL responsibility/file map; `design/rollout.md` additive migration and migration-before-service boundary; `test-plan.md` Store delta of TD-IDEM-003, TD-IDEM-005-TD-IDEM-010, and local TD-IDEM-012.
  - Owner/surface/resources: add canonical `migrations/000003_postgres_http_idempotency.sql` and `internal/infra/postgres/queries/postgres_http_idempotency.sql`, then regenerate only `internal/infra/postgres/sqlcgen/*`; add `internal/infra/postgresidempotency/{doc.go,errors.go,store.go,store_reserve.go,store_acquire.go,store_complete.go,store_release.go,store_reconcile.go,store_epoch.go}` and their fixed tests plus `docs_test.go`; add/extend `test/postgres_http_idempotency_fixtures_integration_test.go` and `test/postgres_http_idempotency_integration_test.go`; change only the matching driver/generated import exemptions in `.golangci.yml`. Mutable proof resources are one disposable PostgreSQL 17 writer with `track_commit_timestamp=on`, bounded pools, relation/row locks, application-name trigger gates, backend PIDs, helper subprocesses, IPC gates, and disposable feature/outbox probe rows; each test owns cleanup.
  - Depends on: T1 — output handoff — needed to start.
  - Handoff: accepted additive schema/query/generated authority, concrete writer Store, reservation publication group, caller-`pgx.Tx` Acquire/Complete primitives, rollback release, writer reconciliation, exact epoch materialization, and deterministic PostgreSQL fixtures consumed by T3.
  - Proof: preserve canonical authority with `make migration-check` and `make sqlc-check`. TD-IDEM-003 recorded v1/current-v2 disagreement replays under retained v1 and rejects v1 removal: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyRecordedFingerprintVersion$' -count=1`. TD-IDEM-005 proves one local publisher, at most one publication connection per Store, writer reclassification, authority-B headroom, and no feature-owner wait with `go test -vet=off -race ./internal/infra/postgresidempotency -run '^TestPublicationGroup$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyFirstPublicationAndPoolHeadroom$' -count=1`, then repeats that integration command with `-race`. TD-IDEM-005A proves one non-resetting `W_in_progress` across coordinator, pool, `ACCESS EXCLUSIVE`-blocked writer read, and publication gate, returning inner 503 and never executing: `go test -vet=off ./internal/infra/postgresidempotency -run '^TestClassificationBudgetDoesNotReset$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyClassificationBudget$' -count=1`. TD-IDEM-006 proves the four live/death schedules, backend/lock disappearance, new generation, and exactly one successor: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyOwnerRecovery$' -count=1`. TD-IDEM-007 proves acquire/mutation/optional-outbox/result/completion atomicity, definite rollback, 40001/40P01, cancellation, and whole-callback-only retry: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCallerTransactionAtomicity$' -count=1`. TD-IDEM-008 proves exact/overflow result bounds, response loss, the post-commit/pre-render SIGKILL carrier, exact epoch recovery, and one durable feature/result/outbox effect: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyReplayResultBoundAndPostCommitDeath$' -count=1`. TD-IDEM-009 keeps the real caller lost-result boundary and drives every reservation/caller writer branch with two absence successors: `REQUIRE_DOCKER=1 go test -tags=integration ./internal/infra/postgres -run '^TestInTxCommitOutcomes$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCommitReconciliation$' -count=1`. The Store half of TD-IDEM-010 proves `committed_at = pg_xact_commit_timestamp(xmin)` and no fallback epoch: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCommitEpoch$' -count=1`. Local TD-IDEM-012 proves stale/read-only absence never reserves, executes, expires, or cleans: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyWriterAuthority$' -count=1`.
  - Reopen if: any fixed reservation, lock, ambiguity, transaction, process-death, or exact-epoch carrier cannot be forced and independently observed — Technical Design; zero cross-replica publication wait/connection is required instead of the accepted replica bound — Specification R6; exact physical commit time cannot be retained or recovered and behavior must change — Specification R9.
  - Accepted: T2; evidence: every exact Proof command passed on its PostgreSQL 17 carrier, including publication race, lock/process-death, caller-transaction, replay, reconciliation, exact-epoch, and writer-authority schedules; fresh independent acceptance review PASS; candidate: current bounded diff.

- [ ] T3: Registered capability runtime owns bounded maintenance, telemetry, readiness, and shutdown while zero registrations remain exactly inert
  - Source: `spec.md` R9, R12, and R13; `design/overview.md` Literal authoritative-commit horizons, maintenance/process lifecycle, security/privacy/telemetry/readiness, config quantities, and bootstrap responsibility/file map; `design/rollout.md` deployment admission and maintenance/process lifecycle; `test-plan.md` lifecycle delta of TD-IDEM-010, TD-IDEM-011, TD-IDEM-011A, TD-IDEM-013A, and TD-IDEM-014.
  - Owner/surface/resources: add `internal/infra/postgresidempotency/{store_maintenance.go,telemetry.go,vocabulary.go}` and their fixed tests; add `internal/config/{http_idempotency_config.go,http_idempotency_config_test.go}` and change `internal/config/{types.go,snapshot_contract_test.go}`; add `cmd/service/internal/bootstrap/{startup_idempotency.go,startup_idempotency_test.go,startup_idempotency_integration_test.go}` and change `cmd/service/internal/bootstrap/{run.go,startup_http.go}` plus reached lifecycle tests; extend only the T2 PostgreSQL integration fixtures/tests for retention, capacity, and telemetry. Mutable proof resources are T2's disposable PostgreSQL 17 writer/data volume, controlled restart, writer-clock rows/locks, `testing/synctest`, recording telemetry exporters, cached health, and the existing background supervisor; each test owns cleanup.
  - Depends on: T2 — output handoff — needed to start.
  - Handoff: accepted complete zero-registration-safe runtime path, conditional config validation, one supervised maintenance loop/probe/snapshot, bounded telemetry vocabularies, and the complete fixed capability path set consumed by T4.
  - Proof: the lifecycle half of TD-IDEM-010 makes unrecoverable epoch loss return 503, terminate maintenance, cache unready, and begin normal drain with `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyEpochLossDrains$' -count=1`. TD-IDEM-011 proves writer-clock horizons, both row-lock serial orders, permanent/active/unknown survival, bounded batch, and monotonic restart with `go test -vet=off ./internal/infra/postgresidempotency -run '^TestMaintenanceScheduleAndBounds$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyRetentionAndCleanupRaces$' -count=1`, then repeats that integration command with `-race`. TD-IDEM-011A proves recoverable cadence, one cycle, safe-read continuation, pre-reserve capacity closure, and stale/terminal unready with `go test -vet=off ./internal/infra/postgresidempotency -run '^TestMaintenanceFailureAndCapacityClosure$' -count=1`, `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyMaintenanceReadiness$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyMaintenanceCapacity$' -count=1`. TD-IDEM-013A proves non-empty closed telemetry, sentinel/label absence, `other` fallback, one terminal outcome, stale observation, and request-scoped versus readiness-terminal classification with `go test -vet=off ./internal/infra/postgresidempotency -run '^TestHTTPIdempotencyTelemetryAndVocabulary$' -count=1`, `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyReadinessLifecycle$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyTelemetry$' -count=1`. TD-IDEM-014 proves empty registration constructs/checks nothing, while one registration validates before serving and joins maintenance before pool close: `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyZeroRegistrationIsInert$' -count=1`.
  - Reopen if: row locks cannot force both retention serial orders, bounded restart/capacity observation cannot fail closed, a bounded required signal cannot be exported without protected data, or empty/nonempty registration cannot remain the sole activation decision — Technical Design; replay/duplicate-risk behavior or exact epoch must change — Specification R9.
  - Blocked: T3; unverified: a nonempty registration can construct the accepted Store without inventing `T_owner_recovery`; evidence: `postgresidempotency.NewStore(pool, recoveryDelay)` rejects a nonpositive deployment-owned recovery delay, while the accepted active configuration names only maintenance/capacity values and `IdempotencyOperation` carries no recovery value; next proof owner: Technical Design — assign one explicit active source for `T_owner_recovery` and revise the T3 owner/proof surface; candidate: current bounded diff.

- [ ] T4: `HTTP_IDEMPOTENCY=none|postgres` generation is deterministic, orthogonal, dependency-clean, and retains exactly the complete capability pack
  - Source: `spec.md` R13; `design/overview.md` Profile generation and absence/presence contract plus repository placement/operator contract; `design/rollout.md` Profile and build gate; `test-plan.md` TD-IDEM-015.
  - Owner/surface/resources: change `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `Makefile`, and the exact profile markers/inventories across every T1-T3 capability-owned path; change `docs/project-structure-and-module-organization.md` and `docs/repo-architecture.md`, add `docs/postgres-http-idempotency.md`, and update only reached README/config/operator surfaces required by the accepted pack. `api/openapi/service.yaml`, generated `internal/openapi/openapi.gen.go`, SQL query/generated files, `.golangci.yml`, config/bootstrap files, migrations, tests, and docs stay canonical/generated/profile-coupled exactly as fixed in T1-T3. Checkout-copy generator fixtures only; no external or persistent mutable resource.
  - Depends on: T3 — output handoff — needed to start.
  - Handoff: accepted `none|postgres` selector, `template.lock` value, immutable repeat behavior, complete on/off inventory, and initialized source identities consumed by T5.
  - Proof: TD-IDEM-015 initializes `none`, postgres with auth on/off and outbox/inbox/messaging combinations, invalid database, explicit empty, unknown, same-lock, and changed-lock cases; successful trees compile with health unopted, failed inputs leave byte-identical trees, equal replay is identical, and off/on inventories are exact. Run `make template-init-check`, `make project-structure-check`, `make openapi-check`, and `make sqlc-check`; every generated/drift command must report no difference.
  - Reopen if: one coherent removable profile cannot preserve shared OpenAPI/SQLC generation — Technical Design; current declarations make the fixed package/file or profile inventory ownership cyclic or give one file two independent reasons to change — Go Ownership.

- [ ] T5: The existing migration and production-image release gate proves both source identities without promoting local evidence to adopter activation
  - Source: `spec.md` R13; `design/overview.md` Active-registration production-image proof carrier; `design/rollout.md` Runtime proof identities, migration/mixed-version sequence, repository publication gates, and rollback falsifiers; `test-plan.md` TD-IDEM-015A and the repository-local prerequisites of TD-IDEM-016.
  - Owner/surface/resources: add `scripts/ci/fixtures/postgres-http-idempotency-active.patch`; change `scripts/ci/runtime-image-build.sh`, the existing `Makefile` `migration-validate`/runtime-image aggregate, `scripts/ci/ci-change-scope.sh` and its self-test, and the existing `.github/workflows/ci.yml` route only as required to make every idempotency profile/fixture-only change take the full migration/image path; finalize `cmd/service/internal/bootstrap/startup_idempotency_integration_test.go` and the existing integration package inventory. Mutable proof resources are disposable Docker/Compose PostgreSQL volumes/networks/containers and two exact local image tags, one per source identity; the creating command registers teardown and reuses each tag without rebuild.
  - Depends on: T4 — output handoff and generated-profile gate — needed to start and prove.
  - Proof: TD-IDEM-015A first runs `make migration-check` and `REQUIRE_DOCKER=1 go test -p=1 -count=1 -tags=integration ./cmd/service/internal/bootstrap -run '^TestPostgresHTTPIdempotencyActiveBootstrap$'`. The existing broad `make check-full` is the one host aggregate: within it the health-only `make runtime-image-build RUNTIME_IMAGE=service:ci` result is reused by `make migration-validate RUNTIME_IMAGE=service:ci`, while migration validation builds the distinct disposable active fixture once and reuses that exact tag for missing schema, read-only/non-writer with commit timestamps enabled, and `track_commit_timestamp=off`. The observable is repeatable up/down/up, migrate-only schema mutation, health-only ready/shutdown and previous/profile-off compatibility, active PostgreSQL rejection before OIDC I/O/serving, and no positive active production-image claim. Run `make template-init-check` after the aggregate to retain the exact generated-profile oracle; current exact-head CI remains the publication authority.
  - Reopen if: the active patch cannot produce one complete registration, a required PostgreSQL rejection occurs only after OIDC initialization, startup order no longer exposes the design's active rejection carrier, or the existing exact-tag aggregate cannot preserve the two source identities — Technical Design.
