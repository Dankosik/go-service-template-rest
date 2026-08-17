# Durable Background Jobs — Go Code / Ownership Design

status: ready

System authority: [`overview.md`](overview.md). This fixes Go ownership and file
shape only. It does not authorize implementation or enter Test Design.

## Import and composition model

Arrows mean “imports”; the graph is acyclic.

```text
internal/<feature> ───────────────────────────────> internal/jobs
internal/infra/postgresjobs ──────────────────────> internal/jobs
internal/infra/postgresjobs ──────────────────────> internal/infra/postgres
internal/infra/postgres/<feature>_repository.go ──> internal/<feature>
internal/infra/postgres/<feature>_repository.go ──> internal/jobs + pgx/sqlcgen
cmd/service/internal/bootstrap ───────────────────> postgres feature adapter
cmd/service/internal/bootstrap ───────────────────> postgresjobs + internal/config
cmd/jobs-worker/internal/bootstrap ───────────────> jobs + postgresjobs + internal/config
```

The flat feature repository is in package `postgres`, so it must not import
`postgresjobs`, which already imports `postgres`. At the composition root,
bootstrap passes the bound methods `store.Stage` and
`store.ResolveAcceptance` as concrete function values to the
feature-repository constructor. Their signatures contain only `context`, pgx,
and `internal/jobs` types. The adapter binds staging to the caller-owned
`pgx.Tx` and invokes writer readback after `ErrCommitUnknown` behind the
feature's existing `Atomically` port. Feature core never sees pgx or a runtime
adapter.

`internal/jobs` is a standard-library-only contract/policy leaf.
`internal/infra/postgresjobs` is the one concrete engine. There is no engine
interface, factory, provider abstraction, River adapter, random lease token, or
shared worker framework. `jobs.Handler[A]`, the injected stage function, and
the binary-local registry builder are function types, not product interfaces.

## Responsibility map

| Responsibility and execution paths | Current evidence | Exact owner/action | Boundary, cleanup, and proof |
| --- | --- | --- | --- |
| Package contract and deliberately absent seams | Both new packages have feature/infra/operator readers, several lifecycle stages, and deliberately omit runtime migration/public control. | Add `internal/jobs/doc.go` and `internal/infra/postgresjobs/doc.go`. | Each names its package authority, import boundary, and absence rules only. It does not compensate for file placement. |
| Immutable typed revision and complete B4 policy construction | No vendor-neutral owner exists; feature core cannot import infra. | Add `internal/jobs/definition.go`. `Definition[A]` owns the exact `(kind,args_version,policy_version)`, positive per-kind payload limit up to 256 KiB, typed prepare/decode functions, and complete no-default policy validation. | Standard library only. Concrete feature owns values/effect truth. It uses acceptance and transition types but owns neither path. Proof: `definition_test.go`. |
| Acceptance identities and caller-visible staging/readback contracts | B2/B3 results cross feature composition and PostgreSQL without making infra authoritative. | Add `internal/jobs/acceptance.go` for bounded logical/producer/occurrence/effect identities, `Prepared`, its fingerprint-derived `ReadbackExpectation`, `AcceptanceIdentity`, `StageResult`, and `ReadbackResult`. | Data contracts and validation only; no definition construction, pgx, transaction, or retry/effect policy. Proof: `acceptance_test.go`. |
| Pure attempt/retry/recovery/effect decision | B5/B6 require one meaning across handler error, panic, timeout, cancellation, lost lease, retry hints, budgets, redrive, and restart. | Add `internal/jobs/transition.go` for attempt/state/outcome vocabulary, bounded outcome/budget facts, decided transition, and the immutable definition revision's pure evaluator with deterministic SHA-256 jitter. | Engine captures/invokes; store persists only the returned decision. No acceptance result or store/engine default. Proof: `transition_test.go`. |
| Strict decode and exact live-revision handler dispatch | NATS handlers are transport-specific; no jobs registry exists. | Add `internal/jobs/registry.go` for unique exact revision registration, typed-handler adaptation, strict JSON decode/validation, supported-key lookup, and missing/duplicate rejection. | It never polls/persists. Every live revision stays registered. Producer and worker import the same definition symbol. Proof: `registry_test.go`. |
| Concrete Store/Session construction and acquired-connection lifetime | Existing `postgres.Pool.Acquire` is the approved bounded owner; an acquired pgx connection does not reconnect in place. | Add `internal/infra/postgresjobs/store.go` containing only concrete `Store`, `Session`, construction, reserved-connection acquisition/release, and invariants shared by every store stage. | No replacement/reacquire loop. Session hides pgx from engine. On terminal failure, bootstrap releases it after engine/handler join so pgxpool destroys an unusable connection; unsafe handler shutdown leaves it to process exit. Proof: constructor/common invariants in unit proof; real healthy/broken release in session integration. |
| Lease-safe reserved-session operation lifetime | A single connection is safe only if claim/observe/finalize/rescue cannot starve renewal or outlive lease safety. | Add `internal/infra/postgresjobs/store_operation.go` for the concrete transaction wrapper: child context plus transaction-local statement/lock timeouts at the effective Store operation budget. | No query/state policy or scheduler. Every Session stage uses it, including read-only observation. Unit proof owns duration math/context; operation-budget integration owns server/lock enforcement. |
| Canceled-query protocol cleanup | All nine final Session stages reach `withOperation`; the current real-PG carrier shows that pgx's default deadline watcher aborts the client side before PostgreSQL has produced its cancellation result, leaving the reserved backend in an aborted transaction and replacing server SQLSTATE with a client deadline. | Change `internal/infra/postgres/postgres.go` to install pgx's existing `CancelRequestContextWatcherHandler` on every repository-created pooled connection, with immediate cancel delivery and the already-configured server statement timeout as its fallback deadline. Change `internal/infra/postgresjobs/store_operation.go` only to remove its busy-poll workaround and roll back with the wrapper-owned non-cancelled cleanup context after pgx has restored protocol idleness. | `postgres.Pool` owns driver context-watcher construction for all pooled connections; `withOperation` owns only its transaction and error classification. No Session reconnect/retirement policy, Store stage, query, config value, exported surface, generic helper, or per-stage cancellation goroutine is added. The caller cancellation remains dominant; server lock/statement cancellation remains inspectable under `ErrOperationTimeout`. Proof: TD-JOBS-011's all-stage real-PG carrier proves both cleanup and SQLSTATE. |
| Stable persistence error vocabulary | Errors cross API/worker/bootstrap and change independently of store construction. | Add `internal/infra/postgresjobs/errors.go` for bounded sentinels/typed conflicts, including operation timeout and terminal control-session failure, plus stable classification helpers. | No SQL or logging. Proof: `errors_test.go`. |
| Generated-row mapping and exact DB vocabulary conversion | SQLC rows cannot escape the adapter; Go state is semantic authority and DB strings are forced copies. | `internal/infra/postgresjobs/store_rows.go` owns cross-stage DB vocabulary and acceptance-readback conversion: state/outcome/effect and acceptance readback. Each `store_<stage>.go` owns its SQLC parameter construction and result-shape mapping after its one query. `store_claim.go` remains the package-private owner of the current shared scalar helpers `requiredTime` and `revisionRows`; moving them is a separate Go Ownership change, not T5 coverage work. | Direct SQLC stays inside the adapter. `store_rows.go` owns no query or transition policy; a stage owns no other stage's result shape. Proof: `store_rows_test.go` for shared conversion and the matching stage test for its result shape, with real DB vocabulary parity. |
| Canonical schema admission | Worker startup and every concrete API producer probe must prove every required schema capability but never migrate. | Add `internal/infra/postgresjobs/store_schema.go` for reusable `CheckSchema` over every required relation, column, check, producer-identity uniqueness constraint/index, and neutral scope. Runtime admission is required-subset: exact required descriptors must remain, while additive N+1 columns/indexes/constraints are tolerated. | It owns runtime capability facts only: no writer/privilege policy, readiness aggregation, Goose execution, or second schema-version table. Missing or changed required authority fails; canonical migration drift remains a CI/migration-history concern. Proof: schema integration mutates both additive and required descriptors. |
| API runtime producer-path admission | API readiness must re-evaluate its concrete writer/schema path without making a durable mutation. | Add `internal/infra/postgresjobs/store_producer_probe.go`; `CheckProducerPath` first calls the Store's required-capability `CheckSchema`, then verifies writer state and producer privileges. | Pool saturation keeps the existing capacity-only readiness result; every completed probe detects any required Stage/readback schema authority loss. No worker lifecycle or migration. Proof: producer-probe integration plus service readiness composition. |
| Same-transaction acceptance and `ErrCommitUnknown` readback | `postgres.Pool.InTx` already preserves caller ownership/unknown commit. | Add `internal/infra/postgresjobs/store_accept.go` for `Stage(ctx, pgx.Tx, jobs.Prepared)`, unique-collision adjudication, immutable-intent comparison, and current-writer acceptance readback. | No begin/commit/rollback/retry or feature operation receipt. Proof: acceptance integration file. |
| Claim and claim-to-attempt creation | Native row locks/SQLC are selected; outbox semantics are not reusable. | Add `internal/infra/postgresjobs/store_claim.go` for claim-snapshot coverage of all execution-required revisions by supplied sorted registry keys, scope `FOR SHARE`, eligible `SKIP LOCKED` selection, generation advance, job transition, attempt insert, mapping that query's declared claim result into `ClaimResult`/`ClaimedAttempt`, and the existing package-private `requiredTime`/`revisionRows` scalar helpers. | Any live coverage gap returns no rows and a bounded compatibility fault. Terminal history is not a global claim gate; a future redrive checks its exact target revision. It persists no handler decision or DB vocabulary conversion. Proof: package mapping test plus claim integration file. |
| Lease renewal and matching durable cancel observation | Renew is a different periodic stage and cancellation must address only the matching attempt. | Add `internal/infra/postgresjobs/store_renew.go` for batch fenced lease renewal and returned cancel-request facts. | Engine cancels contexts; this file owns no goroutine or terminal classification. Proof: renew integration file. |
| Fenced handler finalization | Handler completion, retry time, and attempt history must commit together. | Add `internal/infra/postgresjobs/store_finalize.go` for current-generation CAS and persistence of the already-evaluated transition, retry time, budget changes, and attempt result. | No policy evaluation. An unknown commit is read back before repeating. Proof: finalize integration file. |
| Expired-attempt recovery | Rescue has independent ambiguity and overlap behavior. | Add `internal/infra/postgresjobs/store_rescue.go` for expired-candidate read, mapping that query's declared candidate rows, and expired-generation lock/recheck/persistence of the pure lost-attempt decision; add `engine_rescue.go` for the coordinator's bounded discovery/evaluator/invocation stage. | No rescue goroutine/daemon. The coordinator checks renewal between at most MaxConcurrency candidates; Store/policy remain separate. Finalize/rescue linearize on the job row. Proof: package mapping test plus rescue engine test and recovery integration. |
| Cached operational observation source | State/oldest/compatibility facts change independently of transitions. | Add `internal/infra/postgresjobs/store_observe.go` for state aggregates and exact comparison of the supplied registry keys with distinct execution-required revisions. | No OTel callback or final readiness decision. Terminal history stays in state aggregates but not the compatibility gate. It returns the compatibility component fact. Proof: observation integration file. |
| Worker coordinator and component facts | Existing worker/relay lifecycle patterns are reusable, not their semantics. | Add `internal/infra/postgresjobs/engine.go` for shared coordinator state/run/facts, plus stage files `engine_claim.go`, `engine_attempt.go`, `engine_renew.go`, `engine_rescue.go`, `engine_observe.go`, and `engine_drain.go`. The serial cycle is due renewal; bounded rescue with renewal recheck; capacity-bound claim; due observation. | A coverage fault closes admission; an operation timeout/transport failure closes admission/readiness, signals attempts by the lease bound, and returns a terminal failure without reacquire/replay. No engine file owns process exit/final readiness. Matching `testing/synctest` proof; DB edges stay in integration. |
| Instruments/snapshot/recorders and observation freshness | Current OTel helpers and outbox pattern exist; job signals need their own owner. | Add `internal/infra/postgresjobs/telemetry.go` for instruments, cached DB observation/freshness, event recorders, callbacks, and unregister. | Collection performs no DB I/O. Telemetry exports only engine component readiness/freshness; it does not decide or mirror final diagnostics readiness. Proof: `telemetry_test.go`. |
| Closed metric/operator vocabulary | Project structure fixes `vocabulary.go`; unrecognized values mint series. | Add `internal/infra/postgresjobs/vocabulary.go` for every metric attribute and bounded operator-log literal/mapping. | No instruments or state. Proof: `vocabulary_test.go`. |
| Final worker readiness predicate/publication | System design fixes one predicate over schema, complete execution-required revision coverage, engine, observation freshness, and drain facts. | Add `cmd/jobs-worker/internal/bootstrap/lifecycle.go`; it alone aggregates/publishes diagnostics readiness from its lifecycle flag and engine component facts. Telemetry observes those component facts independently. | No other owner returns the final diagnostics predicate. PostgreSQL timestamps never drive local cadence/freshness. Lifecycle proof owns diagnostics withdrawal; engine/telemetry tests own component transitions. |
| Jobs config | Config snapshot owns runtime values; no job section exists. | Add `internal/config/jobs_config.go`; change `types.go`, `defaults.go`, `validate.go`. `JobsConfig` owns Enabled, PollInterval, MaxConcurrency, LeaseDuration, StoreOperationTimeout, ObservationInterval, DrainTimeout. `LoadJobsWorkerDetailedWithContext` reuses the shared source pipeline but projects and decodes only app, HTTP, log, observability, PostgreSQL, and jobs because `/jobs-worker` reads only those sections. Disabled/zero is default; enabling requires explicit positive mechanisms, PostgreSQL, and `LeaseDuration >= 6 * StoreOperationTimeout` without overflow. | Config imports no jobs/infra/cmd and owns no kind policy. The worker loader does not decode, validate, authenticate, or construct authn, outbound-auth, or object-storage profiles; another worker consumer reopens this boundary. Proof: `jobs_config_test.go`, `jobs_worker_config_test.go`, existing snapshot contract, bootstrap mapping parity. |
| API runtime producer admission and acyclic feature binding | Service dependency startup/readiness owns PostgreSQL; repository structure puts concrete feature adapters in package postgres. | At the first concrete kind, change `cmd/service/internal/bootstrap/startup_dependencies.go` to build `postgresjobs.Store`, require startup `CheckProducerPath`, and replace the generic PostgreSQL readiness probe with the same bounded Store probe. Bootstrap passes `store.Stage` and `store.ResolveAcceptance` into `internal/infra/postgres/<feature>_repository.go`; its unexported function types use only context, pgx, and jobs staging/readback types. The adapter imports jobs/feature/pgx, never postgresjobs. | No producer means no jobs probe or empty Store field. The probe preserves saturation-as-capacity; completed checks fail on writer/schema loss. Existing feature `Atomically` owns unknown-commit routing. Proof: startup/readiness dependency test, producer-probe integration, and feature adapter proof at its checkpoint. |
| Independent jobs-worker composition/process lifetime | Current worker and relay are incompatible; project structure requires new command/bootstrap. | Add `cmd/jobs-worker/main.go` and bootstrap `run.go`, `config.go`, `lifecycle.go`. Engine returns the typed terminal error; lifecycle withdraws readiness, invokes `engine_drain` exactly once, applies the hard bound, and propagates the drain result; engine drain alone quiesces/joins; `run` cleans up and returns the error unchanged; `main` alone classifies it as nonzero. | No in-process reacquire or control-op retry. Only a replacement process constructs a fresh Store/Session. Proof is split across drain mechanics, lifecycle invocation, run, main, Session integration, and black-box replacement. |
| Operator inspect/control/redaction at its named checkpoint | No roles, redaction policy, or current production adapter exists. An exported Controller or Definition placeholder now would be unused. | Add no Controller, `store_operator.go`, or self-attested operator field now. At the first accepted authenticated adapter, add `internal/jobs/operator_policy.go` for the pure minimization/redaction/control evaluator and `internal/infra/postgresjobs/store_operator.go` for its unexported transition/application path and permitted internal view. The present adapter justifies/export-bounds the Controller surface. | The future adapter owns authentication/authorization evidence and rendering only. Operator transaction/action receipt/readback semantics remain fixed by overview. No current route/OpenAPI/CLI/query surface. Reopen security/data owner if permitted representation is absent. |
| Canonical schema and generated SQL | Goose migrations/query glob/SQLC output are current authority. | Later implementation adds next `migrations/NNNNNN_postgres_jobs.sql` (currently 000003) and `internal/infra/postgres/queries/postgres_jobs.sql`, regenerating `sqlcgen/postgres_jobs.sql.go` and `models.go`. | Transactional Goose only; no runtime migration. Text/check DB vocab is a forced copy of jobs semantics with integration parity. No migration is created now. |
| Profile/image/build/docs/import enforcement | Current init, one-image build, depguard, and canonical docs own these surfaces. | Change `build/docker/Dockerfile`, `Makefile`, `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `scripts/ci/runtime-image-build.sh`, `env/.env.example`, `env/config/local.yaml`, `.golangci.yml`, `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`, and `docs/build-test-and-development-commands.md`; add `docs/postgres-durable-background-jobs.md`. | `JOBS=none` removes all marked job code/config/query/generated/migration/doc/test surfaces and regenerates surviving SQLC once. `/jobs-worker` image smoke fails before I/O without builder. Profile matrix proves independence and double init. |

## Implementation-source disposition

| Capability | Selected evidence/rung | Rejected source | Pin/reopen |
| --- | --- | --- | --- |
| Transaction, lock, fencing, pause | Native PostgreSQL via installed pgx/SQLC and current transaction owner. | River supported API fails attempt fencing/pause; River fork is larger; broker/workflow changes boundary. | Current `go.mod`, canonical Goose/query source. Reopen only on proven PostgreSQL inability or boundary change. |
| Typed strict decode/dispatch | Go generics and `encoding/json` strict decode. | River work unit inherits rejected engine/permissive decode; reflection framework/factory unnecessary. | Go 1.26.5. Reopen only after another payload format is accepted. |
| Digest/jitter | Standard-library `crypto/sha256`; attempt generation is monotonic DB state. | Random jitter changes replay; UUID/hash dependency and lease token are unnecessary. | Definition golden vectors. |
| Lifecycle/telemetry | Existing runtimeopts, health, Postgres, and OTel owners/patterns. | Reusing NATS/outbox imports wrong semantics; no second exact-policy worker justifies a framework. | Current repository source. |
| Migration/generation | Existing Goose v3 and SQLC v1.31.1 toolchain. | Runtime migrator, GORM, hand-written generated rows, River schema history. | `tools/go.mod`, `internal/infra/postgres/sqlc.yaml`, migration gates. |

No production module is added. River and probe code remain absent from source,
`go.mod`, image, and schema.

## Inverse Go file map

### Production and generated Go

| File action | One present reason | Owned declarations/path; forbidden responsibilities |
| --- | --- | --- |
| add `internal/jobs/doc.go` | State the pure package contract and absent runtime/transport seams for its multiple audiences. | Package documentation only. |
| add `internal/jobs/definition.go` | Define and validate one immutable typed revision and its complete B4 policy construction. | No acceptance result, transition evaluator, registry map, DB, goroutine, config, or telemetry. |
| add `internal/jobs/acceptance.go` | Own bounded acceptance identities, Prepared, intent fingerprint contract, and Stage/readback results. | No definition construction, transaction, transition, or effect policy. |
| add `internal/jobs/transition.go` | Own attempt/state/outcome/budget facts and the pure retry/recovery/effect evaluator. | No acceptance contracts, persistence, engine loop, or operator rendering. |
| add `internal/jobs/registry.go` | Register exact revisions/handlers and strictly decode/dispatch. | Worker builder/engine; no policy values beyond Definition and no runtime loop. |
| add `internal/infra/postgresjobs/doc.go` | State engine/storage/package lifecycle and absent migration/public-control seams. | Package documentation only. |
| add `internal/infra/postgresjobs/errors.go` | Hold stable persistence/engine error identity and conflict classes. | No SQL/log rendering. |
| add `internal/infra/postgresjobs/store.go` | Construct Store/Session and own reserved connection lifetime/common invariants. | No rows, schema, transition, operator, loop, telemetry. |
| change `internal/infra/postgres/postgres.go` | Construct each repository-owned pgx connection with the one cancellation watcher that requests server cancellation before the existing server timeout fallback. | No per-consumer option, business/session policy, SQL, retry, or connection replacement. |
| add `internal/infra/postgresjobs/store_operation.go` | Bound each reserved-session transaction with context and local statement/lock timeouts, then perform its single rollback only after the pool-owned watcher has settled the protocol. | No driver watcher configuration, stage query, retry policy, scheduler, or readiness aggregation. |
| add `internal/infra/postgresjobs/store_rows.go` | Map cross-stage DB vocabulary and acceptance readback to jobs types. | No query-specific parameter/result shape or transitions. |
| add `internal/infra/postgresjobs/store_schema.go` | Verify the required canonical schema capabilities shared by worker and producer admission while tolerating additive N+1 descriptors. | No writer/privilege policy, mutation, migration, or readiness aggregation. |
| add `internal/infra/postgresjobs/store_producer_probe.go` | Compose required-capability `CheckSchema` with the API's bounded runtime writer/privilege producer checks. | No duplicate schema check, worker lifecycle, mutation, migration, or readiness aggregation. |
| add `internal/infra/postgresjobs/store_accept.go` | Stage/adjudicate acceptance and writer readback. | No transaction lifecycle/business receipt. |
| add `internal/infra/postgresjobs/store_claim.go` | Gate all claims on execution-required revision coverage, then lock scope, atomically create a fenced attempt, and map that query's returned claim shape. | No readiness aggregation, handler, policy, finalization, or cross-stage conversion. |
| add `internal/infra/postgresjobs/store_renew.go` | Renew leases and return matching cancel facts. | No goroutine/terminal decision. |
| add `internal/infra/postgresjobs/store_finalize.go` | Persist an evaluated current-attempt outcome. | No outcome/policy evaluation. |
| add `internal/infra/postgresjobs/store_rescue.go` | Read/map bounded expired candidates and lock/recheck/persist an evaluated recovery. | No evaluator, scheduler, daemon, default recovery, or cross-stage conversion. |
| add `internal/infra/postgresjobs/store_observe.go` | Read state/oldest observations and execution-required revision coverage. | No OTel or final readiness decision. |
| add `internal/infra/postgresjobs/engine.go` | Own shared coordinator construction, state, serial-cycle orchestration, and component-fact snapshot. | No stage-specific claim/handler/renew/rescue/observe/drain logic; no pgx/SQLC/config/cmd/final readiness. |
| add `internal/infra/postgresjobs/engine_claim.go` | Own free-capacity claim admission, unknown-commit resolution, and committed-claim registration in the handler join. | No handler execution/renew/drain. |
| add `internal/infra/postgresjobs/engine_attempt.go` | Own typed handler invocation, panic/timeout/cancel outcome capture, definition-evaluator invocation, and finalization handoff. | No SQL/policy definition/renew/drain. |
| add `internal/infra/postgresjobs/engine_renew.go` | Own lease/3 renewal priority, no-later-stage admission while due, bounded renewal, matching cancel-context signal, and terminal control-session/ownership-loss response. | No reconnect/retry loop, claim/handler classification, process exit, or final readiness. |
| add `internal/infra/postgresjobs/engine_rescue.go` | Own the coordinator's bounded expired-candidate discovery, exact-definition lost-attempt evaluation, fenced Store invocation, and renewal recheck between candidates. | No goroutine, SQL, default policy, claim, or final readiness. |
| add `internal/infra/postgresjobs/engine_observe.go` | Own periodic store observation and delivery of snapshot/freshness input to Telemetry. | No OTel callback/readiness aggregation. |
| add `internal/infra/postgresjobs/engine_drain.go` | Own StartDrain admission close, coordinator quiescence acknowledgement, soft drain, cancellation, join, and cleanup-safety result. | No process diagnostics/deadline or DB transitions. |
| add `internal/infra/postgresjobs/telemetry.go` | Own instruments/recorders/cached observation/freshness/callback. | No DB I/O in collection, vocabulary literals, final readiness decision. |
| add `internal/infra/postgresjobs/vocabulary.go` | Own all bounded metric/log literal mappings. | No instruments/state. |
| add `internal/config/jobs_config.go` | Keep removable job section type/defaults/validation together. | No kind policy or runtime imports. |
| change `internal/config/types.go` | Add marked Jobs field to immutable snapshot. | Snapshot shape only. |
| change `internal/config/defaults.go` | Merge marked jobs defaults. | Merge only. |
| change `internal/config/validate.go` | Invoke jobs validation. | Dispatch only. |
| add `internal/config/jobs_worker_config.go` | Project and validate the config subset read by `/jobs-worker` while retaining common source loading and selected-section unknown-key failure. | No foreign-profile decode, validation, authentication, adapter construction, or jobs runtime mapping. |
| change `cmd/service/internal/bootstrap/startup_dependencies.go` | With the first concrete producer, construct Store, run producer-path admission, and publish the bounded runtime producer probe; inject Stage/readback. | No producer means no jobs probe. No worker/registry/control. |
| add `cmd/jobs-worker/main.go` | Supply binary-local builder and bounded exit classification. | No startup/DB/handler logic. |
| add `cmd/jobs-worker/internal/bootstrap/run.go` | Own ordered construction, safe/unsafe cleanup, and unchanged return of terminal engine failure. | No exit-code classification, Session reconstruction, or DB state machine. |
| add `cmd/jobs-worker/internal/bootstrap/config.go` | Map/validate process and engine mechanism budgets. | No config parsing or kind policy. |
| add `cmd/jobs-worker/internal/bootstrap/lifecycle.go` | Aggregate/publish final readiness, withdraw it, invoke engine drain exactly once, apply the hard bound, and propagate the result on signal or terminal engine failure. | No quiesce/cancel/join mechanics, cleanup, exit classification, Session reconstruction, or DB/policy transition. |
| add generated `internal/infra/postgres/sqlcgen/postgres_jobs.sql.go` | Generated output for jobs query source. | Never hand-edited; store stage files only. |
| change generated `internal/infra/postgres/sqlcgen/models.go` | Generated models for jobs schema. | Never hand-edited; store_rows only. |

No current `store_operator.go`, Controller, operator adapter, feature job file,
or feature Postgres adapter is added because their named adopter inputs do not
exist. At that checkpoint, the fixed paths are
`internal/<feature>/job_<kind>.go`,
`internal/infra/postgres/<feature>_repository.go`, optional
`internal/<feature>/job_<kind>_handler.go` only for an independently changing
effect dependency/lifecycle, and `postgresjobs/store_operator.go` only with a
present authenticated adapter.

The feature repository declares two unexported function types with the minimum
cross-package shapes `func(context.Context, pgx.Tx, jobs.Prepared)
(jobs.StageResult, error)` and `func(context.Context, jobs.ReadbackExpectation)
(jobs.ReadbackResult, error)`. `Store.Stage` and `Store.ResolveAcceptance` bind
directly to them at bootstrap. No postgresjobs type crosses into package
`postgres`, and no driver type crosses into feature core.

### T5-R3 replacement boundary

The current five-file T5-R3 ledger scope is invalid: its all-stage cleanup
postcondition requires the repository-wide pool owner
`internal/infra/postgres/postgres.go`, which the fixed unit excludes. Planning
must replace T5-R3 with one correction unit over
`internal/infra/postgres/postgres.go`,
`internal/infra/postgresjobs/store_operation.go`, its wrapper-local unit proof
only if the cleanup shape changes, and
`test/postgres_jobs_operation_budget_integration_test.go`. The accepted
`internal/config/{jobs_config.go,jobs_config_test.go}` timer-floor delta remains
unchanged and is not writable. The replacement retains TD-JOBS-011's exact
focused config/store/real-PostgreSQL commands and one fresh implementation
review; it adds no pool option, Session/stage edit, generic helper, or a second
integration carrier. This boundary reopens Planning only; behavior, System
Design, and Test Design remain closed because the existing carrier already
observes the required caller-error, server-SQLSTATE, rollback, backend-idle, and
lock-release results.

`sqlcgen/db.go` and `internal/infra/postgres/sqlc.yaml` remain unchanged. The
current DBTX surface/globs already cover new source. PostgreSQL core,
postgresmigrate, health, runtimeopts, NATS, inbox, and outbox production Go are
retained unchanged.

### Go proof carriers

| File action | One proof reason/fixture owner | Forbidden expansion |
| --- | --- | --- |
| add `internal/jobs/definition_test.go` | Exact revision, complete/missing B4 policy, typed prepare/decode, and payload-limit construction. | No acceptance-result or transition-policy cases. |
| add `internal/jobs/acceptance_test.go` | Identity bounds, Prepared validation, intent fingerprint contract, and closed Stage/readback results. | No definition completeness, DB, or retry/effect policy. |
| add `internal/jobs/transition_test.go` | Outcome classification, hint precedence, cap/jitter vectors, attempt/age exhaustion, recovery reset, and effect ambiguity. | No acceptance/registry/DB cases. |
| add `internal/jobs/registry_test.go` | Exact revision uniqueness, live-revision lookup, strict decode, typed dispatch. | No policy evaluator/worker loop. |
| add `internal/infra/postgresjobs/errors_test.go` | Stable error identity/classification, including operation timeout and terminal control-session failure. | No store path. |
| add `internal/infra/postgresjobs/store_test.go` | Constructor validation and common invariants that require no acquired connection. | No fake pool/session or real DB. |
| add `internal/infra/postgresjobs/store_operation_test.go` | Effective timeout selection, child-context deadline, and wrapper-local error propagation. | No lease-ratio policy, stable classification, SQL, or stage behavior. |
| change `test/postgres_jobs_operation_budget_integration_test.go` | Prove every final reserved-Session stage returns caller cancellation or inspectable server `55P03`/`57014`, then leaves its backend idle and lock-free for reuse. | No fake Store, stage-specific cleanup, elapsed-time oracle, or generic PostgreSQL fixture. |
| add `internal/infra/postgresjobs/store_rows_test.go` | Cross-stage DB vocabulary/acceptance-readback conversion and unknown rejection. | No queries or stage-result mapping. |
| add `internal/infra/postgresjobs/store_mapping_test.go` | Claim, rescue, and transition query-result shape mapping rejects malformed generated values before they become Store output. | No query execution, fake Session, or real DB claim. |
| add `internal/infra/postgresjobs/engine_test.go` | Shared coordinator construction/run/fact-snapshot invariants. | No stage behavior, wall-clock sleeps, fake SQL, or final readiness. |
| add `internal/infra/postgresjobs/engine_claim_test.go` | Admission close/race, immutable registry-key request, coverage-fault close, unknown-commit result, and committed-claim join using owned events. | No handler/renew/drain. |
| add `internal/infra/postgresjobs/engine_attempt_test.go` | Typed invocation, panic/timeout/cancel fact capture, evaluator invocation, and finalization handoff under `testing/synctest`. | No SQL/policy re-test. |
| add `internal/infra/postgresjobs/engine_renew_test.go` | Lease/3 cadence, renewal priority, worst-case two-budget cancellation, matching cancellation, and terminal connection/ownership-loss result without reacquire/replay under `testing/synctest`. | No SQL/claim/process exit/final readiness. |
| add `internal/infra/postgresjobs/engine_rescue_test.go` | Bounded candidate count, exact evaluator invocation, stale result, and renewal preemption between candidates under `testing/synctest`. | No SQL/policy re-test/goroutine. |
| add `internal/infra/postgresjobs/engine_observe_test.go` | Observation cadence and state/freshness/compatibility handoff under `testing/synctest`. | No OTel scrape/DB query proof. |
| add `internal/infra/postgresjobs/engine_drain_test.go` | Claim quiescence, registered-attempt soft drain, cancel/join, and unsafe result under `testing/synctest`. | No process hard deadline. |
| add `internal/infra/postgresjobs/goleak_test.go` | Package-wide leak gate for coordinator/attempt/renew/observe goroutines. | `TestMain` only; no behavior assertions. |
| add `internal/infra/postgresjobs/telemetry_test.go` | Instruments, cached/fresh observation, callback no-I/O, component readiness, unregister. | No final diagnostics predicate, label vocabulary ownership, or DB. |
| add `internal/infra/postgresjobs/vocabulary_test.go` | Closed mapping and fallback for every metric/operator literal. | No instruments. |
| add `internal/config/jobs_config_test.go` | Disabled/zero, enabled-explicit values, lease-to-operation ratio/overflow, and PostgreSQL requirement. | No engine map. |
| add `internal/config/jobs_worker_config_test.go` | Worker-scoped config loading ignores malformed foreign profiles but rejects invalid jobs/PostgreSQL values. | No runtime construction or foreign-profile fixture. |
| change `internal/config/snapshot_contract_test.go` | Existing snapshot/default/env corpus includes jobs fields. | No jobs semantics. |
| change `cmd/service/internal/bootstrap/startup_dependencies_test.go` | Conditional startup and repeated runtime producer-path admission, writer/schema loss, deadline, and saturation-capacity behavior without changing non-producer readiness. | No worker or real database. |
| add `cmd/jobs-worker/main_test.go` | Bounded exit classes, including terminal control-session failure to nonzero, and nil-builder fail-closed path. | No startup/cleanup sequence. |
| add `cmd/jobs-worker/internal/bootstrap/config_test.go` | Every JobsConfig field maps to engine config and grace/pool rules. | No parser/kind policy. |
| add `cmd/jobs-worker/internal/bootstrap/run_test.go` | Startup rejection, construction/cleanup order, unchanged terminal-error return, no in-process reconstruction, and unsafe cleanup transfer. | No exit mapping or real DB race. |
| add `cmd/jobs-worker/internal/bootstrap/lifecycle_test.go` | Final readiness aggregation, probe/metric agreement, withdrawal, exactly-one drain invocation, hard-bound order, and result propagation on terminal engine failure. | No quiesce/cancel/join mechanics, cleanup, exit mapping, Store reconnection, or transition semantics. |
| add `cmd/jobs-worker/internal/bootstrap/goleak_test.go` | Package-wide leak gate for diagnostics and lifecycle supervisors. | `TestMain` only; no behavior assertions. |
| add `test/postgres_jobs_fixtures_integration_test.go` | One family-only pgtest setup, canonical admitted definition fixtures, worker process harness, and termination registration shared by two or more jobs integration files. | No assertions or production export. |
| add `test/postgres_jobs_schema_integration_test.go` | Canonical migration/schema, additive N+1 tolerance, every required Stage/readback constraint/index, neutral scope, and Go/DB vocabulary parity through required-capability `CheckSchema`. | No API writer/privilege behavior or acceptance/attempt transition. |
| add `test/postgres_jobs_producer_probe_integration_test.go` | Runtime producer success plus read-only, privilege, and each required-schema authority loss propagated from `CheckSchema`. | No duplicate schema oracle, migration/vocabulary, or feature transaction. |
| add `test/postgres_jobs_session_integration_test.go` | Real reserved Session acquire/use/release, unusable-connection destruction, and pool ownership against pgtest. | No engine restart or job transition behavior. |
| add `test/postgres_jobs_operation_budget_integration_test.go` | Client-context plus transaction-local statement/lock timeout enforcement for every final reserved-Session stage: claim, renew/cancel poll, finalize, rescue read/write, and observation. | No representative sampling, engine scheduling, or handler effect. |
| add `test/postgres_jobs_acceptance_integration_test.go` | Commit/rollback/unknown readback, producer/logical/occurrence/effect conflicts, retention receipt. | No worker lifecycle. |
| add `test/postgres_jobs_claim_integration_test.go` | Execution-required all-claim gate, terminal-history exclusion, acceptance/claim snapshot order, scope-lock pause barrier, SKIP LOCKED, and generation/attempt creation. | No readiness/renew/finalization/rescue. |
| add `test/postgres_jobs_renew_integration_test.go` | Matching fenced lease renewal and durable cancellation observation. | No terminal decision. |
| add `test/postgres_jobs_finalize_integration_test.go` | Persistence/readback of evaluated outcomes, retry time/budgets, stale CAS, and finalize side of finalize/rescue linearization. | No rescue/redrive. |
| add `test/postgres_jobs_recovery_integration_test.go` | Expired-attempt rescue, overlap fence, classifications/budgets, and no cleanup. | No operator redrive or process diagnostics. |
| add `test/postgres_jobs_observation_integration_test.go` | State/oldest/execution-required revision coverage, terminal-history exclusion, and DB-unavailable freshness handoff. | No claim or OTel scrape. |
| add `test/postgres_jobs_process_integration_test.go` | Black-box independently operable jobs-worker startup, progress, readiness, and drain. | No compatibility fault, second worker, feature-specific effect, or capacity claim. |
| add `test/postgres_jobs_compatibility_process_integration_test.go` | Black-box unknown live-revision all-claim closure, visible retention, and compatible replacement-process readiness/claim restoration. | No in-process registry mutation, terminal redrive, lease recovery, or basic startup re-test. |
| add `test/postgres_jobs_recovery_process_integration_test.go` | Black-box two-process lease loss, stale-attempt fence, compatible takeover, and recovery progress. | No compatibility admission or basic startup re-test. |
| add `test/postgres_jobs_lease_safety_process_integration_test.go` | Black-box blocked/lost control operations, readiness/claim closure, attempt signal before lease expiry, bounded nonzero process exit, fresh-process Session acquisition/admission, and later fenced recovery. | No in-process reconnect, blind replay, basic lifecycle, compatibility admission, or effect-absence claim. |

There is deliberately no operator integration file until roles/redaction and a
present adapter admit `store_operator.go`. That later file is
`test/postgres_jobs_operator_integration_test.go` and owns atomic action receipt,
unknown-commit readback, stale/unauthorized actions, permitted inspection, and
redrive/delete refusal.

Test Design may select/order scenarios within these fixed owners; it cannot add
a fake engine seam, package-wide catch-all test, bootstrap real-DB test, or
another fixture package without reopening the responsibility.

## Forced representations and parity

| Semantic owner | Boundary copy | Lowest parity owner |
| --- | --- | --- |
| `internal/jobs` state/outcome vocabulary | Migration text/check constraints and SQLC strings | schema integration enumerates Go corpus through real DB and `store_rows`. |
| `internal/config.JobsConfig` | `postgresjobs.EngineConfig` | jobs-worker bootstrap `config_test.go` maps every field and rejects additions. |
| Typed definition revision | Stored kind/args/policy triple | definition/registry proof plus acceptance round trip; producer/worker import one symbol. |
| Feature immutable-intent bytes | Stored SHA-256 | future feature `job_<kind>_test.go` owns golden vector; acceptance integration proves exact compare. |

## Cleanup and invalidation

River is not added; no adapter/schema/module/notice path exists. NATS worker,
outbox relay, inbox, and their profiles remain independent. `JOBS=none` removes
the new process/packages/config/query/generated/migration/docs/tests and marker
blocks, then regenerates surviving SQLC once. A selected profile retains every
live revision and durable row until delivery/data checkpoints close.

Reopen this map only if implementation evidence shows the reserved coordinator
cannot preserve lease safety without another resource owner, Stage injection
cannot keep the flat postgres adapter acyclic, an authenticated operator adapter
becomes present, or a listed file gains independently changing reasons. That
reopens Go Ownership or System Design; it does not authorize a local alternate
package.

## Ownership review receipt

The fixed candidate passed independent responsibility, package/import, and
file/proof-cohesion review plus every affected-lane re-review. The terminal
control-session chain is singular: engine returns a typed failure without
reconnect/replay; engine drain owns quiesce/cancel/join; lifecycle invokes it
once under the hard bound; run owns cleanup/unchanged return; main owns exit
classification; only a fresh process reacquires and recovers. No material
ownership finding survives. This receipt is design-only.
