# A committed event is published with durable recovery and a redelivery applies its transactional effect once

status: ready

Completion: every local unit T1-T6 is accepted; a named adopting service then
accepts rollout units T7-T15 in gate order, leaving one receipt-backed writer
mutation, selected-relay publication with W3C continuity, one transactional
consumer effect under redelivery, durable finality, and admitted audited unknown
recovery on the writer-primary authority.

Blocked stop: Leave the affected unit unchecked and record its first fixed
scenario ID, command or target procedure, and failed oracle. Reopen Specification
only for caller-visible behavior or a fixed exclusion; reopen Technical Design
only when implementation evidence cannot carry the fixed mechanism, owner, file
map, or proof surface; reopen Test Design only when the fixed oracle cannot be
produced from its accepted fixture. Missing adopter-owned rollout inputs block
only T7-T15, not local acceptance through T6.

Global constraints: Start from HEAD
`da89db83a78ca4a19fefe66d4105f69fb73b7ff0` and the ready input SHA-256 values
listed under Obligation reconciliation. Each cited Test Design scenario keeps
its falsifier, oracle, command/procedure, fixture, proof owner, and reopen
condition unchanged. Keep inbox and outbox as sibling persistence capabilities:
separate tables, canonical migrations, queries, packages, and profile markers;
never move inbox state into `postgresoutbox` or outbox state into
`postgresinbox`. Hand-written migration/query sources precede SQLC regeneration,
and generated files are never edited as authority. Preserve the stored W3C
creation carrier separately from caller metadata, the linked relay root and
creation-context NATS producer flow, writer-primary-only commit reconciliation,
and normal outbox-only fail-closed publisher admission; the DB-only legacy
classification mode is the sole exception and never builds a publisher or
reports ready. Delivery remains at-least-once, same-key NATS handlers remain
concurrent, and no generic consumer-ordering mechanism or test is added. Run
broad Go, race, Docker, migration, lint, and profile gates serially and preserve
all unrelated dirty worktree changes. A post-implementation command that finds
no named test is a proof gap, not a pass.

- [x] T1: Existing outbox creation-context and ordering-key retirement behavior satisfies the fixed trace, privacy, recovery, and concurrency proof.
  - Source: [`../outbox-trace-continuity-and-key-lifecycle/spec.md` R1-R4](../outbox-trace-continuity-and-key-lifecycle/spec.md#behavior-and-contract-delta), [`design/overview.md` D-A-D-H](../outbox-trace-continuity-and-key-lifecycle/design/overview.md), and [`test-plan.md` TD-TRACE-001..003, TD-TRACE-005..007](../outbox-trace-continuity-and-key-lifecycle/test-plan.md#scenario-matrix).
  - Owner/surface/resources: `migrations/000001_postgres_outbox.sql`; `internal/infra/postgres/queries/postgres_outbox.sql` and derived `internal/infra/postgres/sqlcgen/{models.go,postgres_outbox.sql.go}`; `internal/infra/postgresoutbox/{event.go,tracecontext.go,store_append.go,store_rows.go,store_retire.go,relay_publish.go,telemetry.go,errors.go,vocabulary.go}` plus their fixed unit tests; `test/postgres_outbox_{integration,trace_and_retire_integration}_test.go` files selected by the design file map. Shared PostgreSQL integration database and one race run are exclusive proof resources. Current runtime behavior is retained rather than reimplemented; only evidence-backed proof/privacy repairs are authorized.
  - Depends on: none.
  - Proof: TD-TRACE-001; `go test ./internal/infra/postgresoutbox -run '^TestCreationContext' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxEnvelope$' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxAppendWithoutTraceContext$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxTraceContextAllowance$' -count=1`; the stored carrier, metadata bytes, degradation, and separate allowance match the fixed oracle. TD-TRACE-002; `go test ./internal/infra/postgresoutbox -run '^TestPublishSpan' -count=1`; sampled/unsampled/root/link/error/privacy observables match. TD-TRACE-003; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxCreationContextSurvivesRecovery$' -count=1`; retry, lease recovery, reconstruction, and redrive retain one origin. TD-TRACE-005 and TD-TRACE-006; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxRetireOrderingKey$' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxRetireSerializesWithAppend$' -count=1`, and `REQUIRE_DOCKER=1 go test -vet=off -race -tags=integration ./test -run '^TestPostgresOutboxRetireSerializesWithAppend$' -count=1`; the fixed state and lock oracles pass. TD-TRACE-007; `go test ./internal/infra/postgresoutbox -run '^TestEnvelopeLimitsMatchMigrationChecks$' -count=1`, `make sqlc-check`, `make migration-check`, and `make migration-validate`; canonical and generated authority has zero drift.
  - Reopen if: a fixed carrier, sampling, lock, or generated-authority oracle cannot be produced without changing the selected trace/link/retirement mechanism — Technical Design owns the mechanism and Test Design owns only an infeasible oracle.
  - Accepted: T1; evidence: the fixed T1 command set passed, including PostgreSQL lock schedules, race, SQLC drift, and migration rehearsal; independent implementation review `PASS`; candidate: current bounded diff.

- [x] T2: Every outbox append leaves a durable immutable receipt and an unknown commit is reconciled only from the writer primary with the same pre-transaction event.
  - Source: [`spec.md` R4](spec.md#r4--an-unknown-transaction-commit-is-resolved-by-a-stable-receipt), [`design/overview.md` Commit receipts and responsibility/file maps](design/overview.md#commit-receipts), and [`test-plan.md` TD-OUTBOX-001..002](test-plan.md#scenario-matrix).
  - Owner/surface/resources: canonical `migrations/000001_postgres_outbox.sql` and `internal/infra/postgres/queries/postgres_outbox.sql`, then derived SQLC output; add `internal/infra/postgresoutbox/store_receipt.go` and its test; change `store_append.go` and its test; `docs/postgres-transactional-outbox.md`; add `examples/reference-service/postgres_outbox_reconciliation_integration_test.go`; fixed integration deltas in `test/postgres_outbox_*_integration_test.go`. The configured writer PostgreSQL pool is the only absence authority; the receipt table has no cleanup owner.
  - Depends on: T1 — output handoff — needed to start.
  - Handoff: T1 supplies the accepted trace-context migration/query/generated state and immutable carrier behavior; T2 extends those exact canonical sources without charging trace context to the receipt or caller envelope.
  - Proof: TD-OUTBOX-001; `go test ./internal/infra/postgresoutbox -run '^TestCommitReceipt' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxCommitReceiptAtomicityAndLifetime$' -count=1`; the fixed v1 vector, one-statement atomicity, conflict, version, rejected-append, and cleanup-lifetime oracles pass. TD-OUTBOX-002; `REQUIRE_DOCKER=1 go test -tags=integration ./internal/infra/postgres -run '^TestInTxCommitOutcomes$' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./examples/reference-service -run '^TestPostgresOutboxWriterPrimaryReconciliation$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxCommitReconciliationAuthority$' -count=1`; only an authoritative absence retries the mutation, with the same event. `make sqlc-check` leaves no derived diff. T1 preservation on every shared canonical/append surface; `go test ./internal/infra/postgresoutbox -run '^(TestCreationContext|TestEnvelopeLimitsMatchMigrationChecks)' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^(TestPostgresOutboxEnvelope|TestPostgresOutboxAppendWithoutTraceContext|TestPostgresOutboxTraceContextAllowance|TestPostgresOutboxCreationContextSurvivesRecovery|TestPostgresOutboxRetireOrderingKey|TestPostgresOutboxRetireSerializesWithAppend)$' -count=1`, `make migration-check`, and `make migration-validate`; creation context, generated bounds, migration history/runtime rehearsal, recovery continuity, and retirement remain accepted after the receipt rewrite.
  - Reopen if: the one-statement append cannot insert event and receipt atomically, or `CommitNotApplied` cannot be contingent on a writable current-primary read — Technical Design owns both seams.
  - Accepted: T2; evidence: the fixed TD-OUTBOX-001/002 commands, SQLC drift, T1 preservation, migration checks, and image-backed rehearsal passed; independent implementation review `PASS`; candidate: current bounded diff.

- [x] T3: Publication uncertainty is sticky, bounded at `max_attempts`, recoverable through audited actions, observable without identity leakage, and classifiable to authoritative zero before relay start.
  - Source: [`spec.md` R3 and R2 precedence](spec.md#r3--ambiguous-publication-stops-automatically-and-remains-recoverable), [`design/overview.md` durable data, audited recovery, observability, and exact file maps](design/overview.md#durable-data-authority), and [`test-plan.md` TD-OUTBOX-006..013](test-plan.md#scenario-matrix).
  - Owner/surface/resources: extend T2's canonical migration/query and regenerate SQLC; `internal/infra/postgresoutbox/{store.go,store_claim.go,store_rows.go,store_finalize.go,relay.go,relay_finalize.go,store_operator.go,errors.go,store_legacy_classification.go,store_maintenance.go,telemetry.go,vocabulary.go,doc.go}` plus fixed tests; `cmd/outbox-relay/internal/bootstrap/{run.go,legacy_classification.go,legacy_classification_test.go}`; bounded writer readback in `docs/postgres-transactional-outbox.md`; integration deltas in `test/postgres_outbox_*_integration_test.go`. The shared legacy-state corpus is one authority for online claim and pre-start classification. PostgreSQL rows/locks, the DB-only mode, and race gates are exclusive resources.
  - Depends on: T2 — output handoff — needed to start.
  - Handoff: T2 supplies the receipt-aware canonical schema/query/generated state and writer reconciliation; T3 preserves receipts while adding nullable legacy classification, sticky quarantine, action-aware durable audit, and observation.
  - Proof: TD-OUTBOX-006; `go test ./internal/infra/postgresoutbox -run '^TestRelayPublicationDispositions$' -count=1`. TD-OUTBOX-007; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxStickyAtLimitQuarantinesWithoutPublish$' -count=1` and `REQUIRE_DOCKER=1 go test -vet=off -race -tags=integration ./test -run '^TestPostgresOutboxStickyAtLimitQuarantinesWithoutPublish$' -count=1`. TD-OUTBOX-008; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxAuditedUnknownRecovery$' -count=1`. TD-OUTBOX-009; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxConcurrentUnknownActions$' -count=1` and `REQUIRE_DOCKER=1 go test -vet=off -race -tags=integration ./test -run '^TestPostgresOutboxConcurrentUnknownActions$' -count=1`. TD-OUTBOX-010 local carrier/parity delta; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxLegacyUncertaintyClassification$' -count=1` and `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestLegacyUncertaintyClassificationMode$' -count=1`; classification is monotonic, lock-sensitive, receipt-free, DB-only, and ends on zero. TD-OUTBOX-011 outbox observation/discovery delta; `go test ./internal/infra/postgresoutbox -run '^TestErrorClassVocabularyIsBounded$' -count=1`, `go test ./internal/infra/postgresoutbox -run '^TestTelemetryBoundedContract$' -count=1`, `go test ./internal/infra/postgresoutbox -run '^TestTelemetryObservesEveryObservedCount$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxUnknownObservationAndDiscovery$' -count=1`; unknown/poison/receipt observation and finite writer discovery satisfy their fixed privacy oracle. T4 owns the same scenario's NATS telemetry delta. TD-OUTBOX-012; `go test -vet=off -race ./internal/infra/postgresoutbox -run '^TestRelayMarkPublishedReconciliationRecordsDurableProgress$' -count=20`. TD-OUTBOX-013 local schema delta; `make sqlc-check`, `make migration-check`, `make migration-validate`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxStartupRequiresRedriveLedger$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxStartupRequiresReceiptLedger$' -count=1`; canonical, generated, populated-migration, and runtime relation oracles pass. T1 preservation on every migration/query/generated/row/telemetry/operator surface changed here; `go test ./internal/infra/postgresoutbox -run '^(TestCreationContext|TestPublishSpan|TestEnvelopeLimitsMatchMigrationChecks)' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^(TestPostgresOutboxEnvelope|TestPostgresOutboxAppendWithoutTraceContext|TestPostgresOutboxTraceContextAllowance|TestPostgresOutboxCreationContextSurvivesRecovery|TestPostgresOutboxRetireOrderingKey|TestPostgresOutboxRetireSerializesWithAppend)$' -count=1`; trace/link/privacy, recovery continuity, and retirement remain accepted after sticky/operator rewrites.
  - Reopen if: claim cannot quarantine at-limit sticky work in its existing round trip, audit/event locks cannot serialize the actions in one transaction, the fixed classifier cannot run on the writer path, or authorized discovery requires identity in aggregate telemetry — Technical Design owns those mechanisms.
  - Accepted: T3; evidence: the fixed sticky-quarantine, audited-action, legacy-classification, privacy/observation, reconciliation-race, canonical/generated, migration-rehearsal, startup-ledger, and T1-preservation commands passed; the query-sensitive race and SQLC drift checks passed on the final candidate; independent implementation review `PASS`.

- [x] T4: The selected NATS adapter and supervised publisher runtime make the combined profile runnable with W3C continuity while normal outbox-only startup still fails closed.
  - Source: [`spec.md` R1-R2](spec.md#r1--the-combined-outbox-and-nats-profile-is-runnable-without-source-edits), [`design/overview.md` NATS publication and trace flow plus profile ownership](design/overview.md#nats-publication-and-trace-flow), [`test-plan.md` TD-OUTBOX-003..005 and TD-OUTBOX-011 transport-privacy delta](test-plan.md#scenario-matrix), and trace [`TD-TRACE-004`](../outbox-trace-continuity-and-key-lifecycle/test-plan.md#scenario-matrix).
  - Owner/surface/resources: add `internal/infra/natsjs/outbox_publisher.go` and its test; change NATS `doc.go` and `telemetry.go` plus privacy tests; update `internal/infra/postgresoutbox/publisher.go`; add `cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go` and test; add `internal/config/configtest/messaging.go`; change relay `run.go`, `run_test.go`, `main.go` and bounded failure tests, plus worker config-parity test; update `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `.golangci.yml`, `Makefile`, README/command/architecture/outbox docs, and replace the test-local adapter in `test/postgres_outbox_natsjs_integration_test.go`. JetStream, profile-generated checkouts, lifecycle joins, and race runs are exclusive resources.
  - Depends on: T3 — output handoff — needed to start.
  - Handoff: T3 supplies the final relay/store transitions and the DB-only `run.go` branch; T4 supplies normal-mode publisher construction, supervision, W3C forwarding, privacy, and exact profile removal around that branch.
  - Proof: TD-OUTBOX-003 and TD-TRACE-004; `go test ./internal/infra/natsjs -run '^TestOutboxPublisher' -count=1`, `go test -vet=off -race ./internal/infra/natsjs -run '^TestOutboxPublisher' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxNATSConformance$' -count=1`; fixed failure classification, concurrency, envelope, and W3C oracles pass. TD-OUTBOX-011 transport-privacy delta; `go test ./internal/infra/natsjs -run '^TestMessagingTelemetryContract$' -count=1`; the NATS owner removes message/consumer identity while retaining the fixed bounded attributes. TD-OUTBOX-004; `make template-init-check`, `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestNATSPublisherConfigParity$' -count=1`, `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestPublisherRuntimeValidation$' -count=1`, and `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelayComposition$' -count=1`; combined, outbox-only, NATS-only, neither, removal, and repeat-zero-drift outcomes pass. TD-OUTBOX-005; `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelayPublisherRuntime' -count=1`, `go test -vet=off -race ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelayPublisherRuntime' -count=1`, `go test ./cmd/outbox-relay -run '^TestReportFailureIsBoundedAndSanitized$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestNATSConnectionReconnectExhaustion$' -count=1`; readiness, terminal drain, join, shutdown, sanitized failure, and unchanged attempt accounting pass. T3 preservation on the shared relay bootstrap; `go test ./cmd/outbox-relay/internal/bootstrap -run '^TestLegacyUncertaintyClassificationMode$' -count=1`; the DB-only route remains publisher-free and zero-authoritative after normal-mode composition.
  - Reopen if: private NATS validation cannot be reused without an import cycle, messaging removal cannot retain the nil fail-closed normal builder, or one publisher interface cannot supervise `Client.Run`/`Ready`/`Shutdown` — Technical Design owns the selected adapter/runtime seam.
  - Accepted: T4; evidence: the fixed adapter classification/envelope, two-carrier reverse-release W3C concurrency, real PostgreSQL-to-JetStream continuity, NATS privacy, profile composition/removal, publisher lifecycle/race, reconnect exhaustion, fail-closed outbox-only, and T3 DB-only preservation commands passed on the final candidate; focused lint, structure, diff hygiene, full lint/test, and exact full template initialization were green; independent implementation review `PASS` after the carrier-isolation repair.

- [x] T5: The independently selectable PostgreSQL inbox applies one same-database effect per consumer and logical message across failure, concurrency, restart, DLQ, and redrive without ordering or expiry machinery.
  - Source: [`../inbox-idempotent-consumption/spec.md` R1-R7](../inbox-idempotent-consumption/spec.md#behavior-and-contract-delta), its [`design/overview.md`](../inbox-idempotent-consumption/design/overview.md), and [`test-plan.md` TD-INBOX-001..009](../inbox-idempotent-consumption/test-plan.md#scenario-matrix).
  - Owner/surface/resources: add canonical `migrations/000002_postgres_inbox.sql`, `internal/infra/postgres/queries/postgres_inbox.sql`, and `internal/infra/postgresinbox/{inbox.go,inbox_test.go}`; regenerate only inbox-derived SQLC plus the shared model; add `docs/postgres-idempotent-inbox.md`, `examples/reference-service/postgres_inbox_integration_test.go`, `test/postgres_inbox_integration_test.go`, and `test/postgres_inbox_natsjs_integration_test.go`; update only the accepted structure/architecture/README/command docs, `.golangci.yml`, `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, and required Make targets. Preserve T4's shared NATS privacy change rather than creating inbox telemetry. PostgreSQL, JetStream, generated profile checkouts, locks, and race gates are exclusive resources.
  - Depends on: T4 — output handoff — needed to start.
  - Handoff: T4 supplies the final outbox/NATS profile removal, config-parity, selected adapter, and privacy state; T5 adds sibling inbox source/derived/profile surfaces without changing outbox tables, state, or normal relay admission.
  - Proof: TD-INBOX-001; `go test ./internal/infra/postgresinbox -run '^TestClaimValidation$' -count=1`. TD-INBOX-002; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxClaimAndEffectAtomicity$' -count=1`. TD-INBOX-003; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxRedeliveryResolvesTransactionOutcome$' -count=1`. TD-INBOX-004; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxConcurrentClaimAndEffect$' -count=1` and `REQUIRE_DOCKER=1 go test -vet=off -race -tags=integration ./test -run '^TestPostgresInboxConcurrentClaimAndEffect$' -count=1`. TD-INBOX-005; `go test ./internal/infra/postgresinbox -run '^TestInboxSchemaHasNoExpirySurface$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxClaimSurvivesRestart$' -count=1`. TD-INBOX-006; `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxNATSLogicalIdentityAndAcknowledgement$' -count=1`; every atomicity, commit-unknown, cancellation, lock-wait, no-expiry, restart, logical/publication identity, ACK, and per-consumer oracle passes. TD-INBOX-007; `REQUIRE_DOCKER=1 go test -tags=integration ./examples/reference-service -run '^TestPostgresInboxAdapterPlacement$' -count=1`, `make project-structure-check`, and `make lint`. TD-INBOX-008; `make template-init-check`, `make sqlc-check`, `make migration-check`, and `make migration-validate`; inbox-only, joined, outbox-only, none, invalid selection, explicit removal/regeneration, and repeat zero-drift pass. TD-INBOX-009; `go test ./internal/infra/natsjs -run '^TestDeliveryVocabularyIsBounded$' -count=1`, `go test ./internal/infra/natsjs -run '^TestMessagingTelemetryContract$' -count=1`, `REQUIRE_DOCKER=1 make test-messaging-race`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestNATS' -count=1`; retry/DLQ/drain/privacy and same-key overlap remain unchanged.
  - Reopen if: `Claim` cannot use the caller's exact `pgx.Tx`, unique-index arbitration cannot be controlled by the fixed real-PostgreSQL oracle, profile removal cannot regenerate shared SQLC without deleting outbox output, or the selected transport changes logical identity on redrive — use the reopen owner fixed by the failing TD-INBOX scenario.
  - Accepted: T5; evidence: canonical migration/query and regenerated SQLC, stateless caller-transaction claim, atomicity/commit-unknown/cancellation/concurrency/restart proofs, joined ordinary redelivery/DLQ/redrive and forced-shutdown rollback/ACK proof, reference adapter placement, independent profile composition/removal/repeat drift, NATS privacy/regression/race, migration rehearsal through versions 1 and 2, lint/structure, and independent implementation review all passed on the fixed candidate.

- [x] T6: The integrated local candidate proves the production-closed outbox/NATS/inbox path and every preserved regression without accepting any target rollout claim.
  - Source: [`test-plan.md` TD-OUTBOX-018 and local delta of TD-OUTBOX-016](test-plan.md#scenario-matrix), inbox [`TD-INBOX-012` local rehearsal](../inbox-idempotent-consumption/test-plan.md#scenario-matrix), and trace [`TD-TRACE-008`](../outbox-trace-continuity-and-key-lifecycle/test-plan.md#scenario-matrix).
  - Owner/surface/resources: the exact accepted T1-T5 candidate; no independent production surface. Docker/PostgreSQL/JetStream, generated-profile checkouts, race gates, lint, and repository aggregate are serialized. A failure routes to the earliest owning unit rather than being patched in this proof-only unit.
  - Depends on: T5 — output handoff and T1-T5 accepted receipts — needed to start.
  - Handoff: T5 supplies the final local candidate containing every T1-T5 accepted output; T6 supplies one integrated local receipt and does not widen any earlier acceptance verdict.
  - Proof: rehearse the two fixed canaries with `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxNATSLogicalIdentityAndAcknowledgement$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutboxNATSConformance$' -count=1`. Then run `REQUIRE_DOCKER=1 make test-outbox-race`, `REQUIRE_DOCKER=1 make test-messaging-race`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresOutbox' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestNATS' -count=1`, `make template-init-check`, `make project-structure-check`, and `make lint`; at-least-once transaction, lease, order, drain, profile, privacy, dependency placement, and generic-ordering exclusion remain green. Record only local evidence; target effects, inventory, writer reads, and recovery admission remain unverified.
  - Reopen if: an integrated failure contradicts a fixed upstream mechanism or proof surface; reopen its narrow scenario owner and preserve unaffected unit receipts.
  - Accepted: T6; evidence: both local canaries, serialized outbox and messaging race gates, all PostgreSQL outbox and NATS integration regressions, full profile initialization, structure, lint, and diff hygiene passed on the integrated candidate; the one stale recovery test carrier was aligned with the already-accepted sticky-unknown state and replayed green; independent implementation review `PASS`. Evidence is local only and accepts no target rollout fact.

- [ ] T7: The adopting service has an authoritative outbox inventory and every old relay/operator writer is paused with stable writer-primary state.
  - Source: [`rollout.md` Gate 1](rollout.md) and [`test-plan.md` TD-OUTBOX-014](test-plan.md#scenario-matrix).
  - Owner/surface/resources: adopting-service operator; exact relay/recovery deployment inventory, current writer-primary DSN, old relay leases, audit writes, and two bounded writer-primary snapshots. No template source write.
  - Depends on: T6 — local production-closure proof gate — needed to start.
  - External input/gate: a named adopting service must supply its exact inventory commands, deployed revisions, writer-primary identity, and accepted lease horizon.
  - Proof: execute TD-OUTBOX-014 exactly; zero old active relay replicas, no lease advance, and stable audit count across the two writer-primary reads is the only pass. Any unknown replica/action or replica/cache read fails safely.
  - Reopen if: the target has no authoritative replica inventory or writer-primary readback — rollout design owns the missing target surface.

- [ ] T8: The adopting service's outbox schema is expanded from canonical source and its writer-primary catalog matches receipts, sticky legacy state, and action-aware durable audit.
  - Source: [`rollout.md` Gate 2](rollout.md), [`test-plan.md` TD-OUTBOX-013 target delta](test-plan.md#scenario-matrix), and T3's accepted local TD-OUTBOX-013 receipt.
  - Owner/surface/resources: target database owner; one reviewed forward migration derived from the canonical template migration/query state; target writer PostgreSQL catalog/lock budget. No hand edit of generated SQL is authority.
  - Depends on: T7 — paused relay/operator safety gate — needed to start.
  - External input/gate: the adopting service supplies its forward-migration number and measured unfinished-row/catalog-lock envelope.
  - Proof: consume the accepted `make sqlc-check`, `make migration-check`, and `make migration-validate` receipt, apply the target migration once, and record the fixed writer-primary catalog/constraint/index/FK readback. Failure rolls back only while no new receipt, sticky fact, or action exists.
  - Reopen if: the target cannot express the forward migration without losing legacy classification — Technical Design owns the schema transition.

- [ ] T9: Every live target writer creates the stable event before its transaction, writes one matching receipt, and reconciles lost commit responses from the writer primary.
  - Source: [`rollout.md` Gate 3](rollout.md) and [`test-plan.md` TD-OUTBOX-015](test-plan.md#scenario-matrix), backed by accepted TD-OUTBOX-002.
  - Owner/surface/resources: adopting-service owner; all feature writer routes/shards and target writer-primary event/receipt/domain rows. Relay remains paused.
  - Depends on: T8 — expanded schema state — needed to start.
  - External input/gate: the adopter supplies exact writer deployment routes and canary mutation/readback commands.
  - Proof: execute TD-OUTBOX-015 exactly; every live replica is the new build, every route's canary has one matching receipt and domain mutation, forced lost-response reconciliation follows TD-OUTBOX-002, and no old-format append appears.
  - Reopen if: any writer route cannot retain one stable pre-transaction event — Technical Design owns the repository-adapter route.

- [ ] T10: Retained legacy outbox rows are classified monotonically on the target writer to an authoritative zero without starting a publisher.
  - Source: [`rollout.md` Gate 4](rollout.md) and [`test-plan.md` TD-OUTBOX-010 target delta](test-plan.md#scenario-matrix), backed by T3's accepted local parity/carrier receipt.
  - Owner/surface/resources: target database/operator owner; the new relay binary in DB-only mode, writer config/overlays, `outbox.max_attempts`, `outbox.cleanup_batch_size`, and statement budget. Every old writer, relay, and operator remains stopped.
  - Depends on: T9 — new-writer-only state — needed to start.
  - External input/gate: approved target config/overlay arguments and maintenance envelope.
  - Proof: run `outbox-relay --classify-legacy-uncertainty --config <service-config>` with the approved overlays until its successful final zero. A lock, cancellation, SQL/config error, non-zero exit, publisher/NATS construction, readiness, receipt synthesis, or ordering-head change fails the gate; committed batches remain the rerun cursor.
  - Reopen if: the fixed Store/mode cannot run on the target writer path — Technical Design owns the carrier.

- [ ] T11: The adopting consumer has one stable identity, a same-writer claim/effect boundary, and no replayable pre-cutover effect lacking protection.
  - Source: [`../inbox-idempotent-consumption/rollout.md` Gate 1](../inbox-idempotent-consumption/rollout.md) and [`test-plan.md` TD-INBOX-010](../inbox-idempotent-consumption/test-plan.md#scenario-matrix).
  - Owner/surface/resources: adopting-service owner; durable stream/consumer inventory, feature adapter path, authoritative historical logical-ID/effect set, and writer-primary seed/readback when needed.
  - Depends on: T6 — accepted local inbox mechanism and exemplar — needed to start.
  - External input/gate: a named durable consumer, exact deployment inventory, and authoritative history source; a sample or unknown history is unavailable input, not success.
  - Proof: execute TD-INBOX-010 exactly; one fixed `stream/consumer`, one same-PostgreSQL transaction, and zero replayable applied IDs lacking either a seeded claim or retained domain idempotency is the only pass.
  - Reopen if: the service cannot establish a safe forward-only boundary — Specification owns the changed historical guarantee.

- [ ] T12: The target inbox schema is authoritative and only inbox-enabled replicas own the unchanged durable consumer after a complete old-handler drain.
  - Source: [`../inbox-idempotent-consumption/rollout.md` Gates 2-3](../inbox-idempotent-consumption/rollout.md), [`test-plan.md` TD-INBOX-011](../inbox-idempotent-consumption/test-plan.md#scenario-matrix), and T5's accepted TD-INBOX-008 receipt.
  - Owner/surface/resources: target database owner then service operator; one additive inbox forward migration, writer-primary catalog, old/new worker replicas, JetStream durable-consumer inventory, and the accepted handler drain budget. Inbox schema remains separate from T8's outbox schema.
  - Depends on: T11 — identity/history/effect boundary — needed to start; T8 — target canonical outbox schema state and migration-number allocation — needed to start.
  - External input/gate: target migration number, worker deployment commands, durable-consumer inventory command, and accepted drain bound.
  - Proof: consume T5's TD-INBOX-008 local migration/profile receipt, then execute TD-INBOX-011 exactly: writer-primary schema readback matches the two-key claim table; every old replica drains; JetStream reports no old delivery owner; only the new build admits the same durable name. Any overlap or failed read leaves consumption paused.
  - Reopen if: the selected worker cannot expose durable identity/readiness — Technical Design owns the worker admission surface.

- [ ] T13: While every outbox relay is paused, the target consumer proves one durable claim/effect and successful acknowledgement across forced redelivery.
  - Source: [`../inbox-idempotent-consumption/rollout.md` Gate 4](../inbox-idempotent-consumption/rollout.md) and [`test-plan.md` TD-INBOX-012 target delta](../inbox-idempotent-consumption/test-plan.md#scenario-matrix), backed by T6's local canary receipt.
  - Owner/surface/resources: service/operator owner; target canary subject, fixed logical ID, different redelivery occurrence, durable consumer, writer-primary claim/effect query, and JetStream state. Every relay remains paused.
  - Depends on: T12 — new-worker-only consumer state — needed to start; T7 — relay pause safety gate — needed to start.
  - External input/gate: adopter-owned canary subject, effect readback, and redelivery control.
  - Proof: execute TD-INBOX-012 exactly; writer primary reports one claim and effect, both deliveries finish successfully, ACK pending returns to zero, and no old handler is live. Failure stops the new worker and keeps relay start blocked.
  - Reopen if: the target cannot force/read a duplicate without publishing production work — Technical Design owns the canary surface.

- [ ] T14: Only new relay/operator replicas publish a stable-receipt canary through W3C continuity to every already-protected consumer and record durable finality.
  - Source: [`rollout.md` Gate 5](rollout.md) and [`test-plan.md` TD-OUTBOX-016 target delta](test-plan.md#scenario-matrix), consuming T10 and T13 receipts.
  - Owner/surface/resources: adopting service/operator; new relay/recovery deployment identity, NATS stream/config, stable event/receipt canary, sampled W3C origin, all affected durable consumers, writer-primary inbox/effect/outbox finality reads, readiness, and bounded telemetry.
  - Depends on: T10 — authoritative zero legacy classification — needed to start; T13 — consumer-before-relay duplicate-suppression receipt — needed to start.
  - External input/gate: exact target subjects, consumers, effect rows, relay inventory, and canary readbacks.
  - Proof: execute TD-OUTBOX-016 exactly; one identity and origin traverse append/publish/consume, forced redelivery leaves one transactional effect per consumer, outbox finalizes, readiness is true, bounded telemetry leaks no identity, and no old relay/handler is live. Failure drains new relays and preserves backlog/rows.
  - Reopen if: the target cannot observe the fixed identity/trace/finality path — Technical Design owns the target composition proof surface.

- [ ] T15: Target operators can discover and safely repeat audited unknown recovery without telemetry leakage or an unprotected redrive.
  - Source: [`rollout.md` Gate 6](rollout.md) and [`test-plan.md` TD-OUTBOX-017](test-plan.md#scenario-matrix), backed by accepted TD-OUTBOX-008, TD-OUTBOX-009, and TD-OUTBOX-011 receipts.
  - Owner/surface/resources: target operator; authorized finite writer readback, one bounded unknown canary, audited confirm/redrive access, writer-primary event/head/audit rows, and the still-valid T13 consumer receipt.
  - Depends on: T14 — selected relay finality and consumer idempotency state — needed to start.
  - External input/gate: adopter-owned authorized discovery route, runbook, canary unknown, and action/readback commands.
  - Proof: execute TD-OUTBOX-017 exactly; discovery returns only approved fields, same-action replay returns its first result, conflicting audit reuse changes no state, confirm performs no broker call or redrive retains identity/sticky uncertainty, audit survives, and leaving unresolved evidence quarantined is safe. Disable access on any failure.
  - Reopen if: discovery requires durable identity in aggregate telemetry or redrive cannot remain behind current consumer protection — Technical Design owns the missing recovery surface.

## Dependency and shared-surface disposition

No local wave is planned: T1-T5 deliberately serialize the shared outbox
migration/query/generated pair, relay bootstrap, NATS telemetry, initializer,
profile fixture, and shared documentation; T6 consumes their accepted integrated
candidate. T7 and T11 become separately eligible only after T6, but no rollout
wave is recorded until a named adopter supplies positive evidence that their
deployment commands, writer database, durable consumers, and operators are
pairwise independent. T8 and T12 serialize target migration authority without
merging their schemas. T14 cannot start before both the outbox legacy-zero and
inbox consumer-canary receipts.

## Obligation reconciliation

Ready input SHA-256 values:

| Input | SHA-256 |
| --- | --- |
| `../inbox-idempotent-consumption/spec.md` | `741980b2e59431817a4982c5fd248a8ca17356b8d39ffce40e62d0e93dbc56ed` |
| `../inbox-idempotent-consumption/design/overview.md` | `0636caf7dbbddd7a1bd0f48f7db371a1f3d98195e2727043bd8094ddac3a76b3` |
| `../inbox-idempotent-consumption/test-plan.md` | `dd2a824aaad88aaa9109145b0c0825ed2150f46245491060ed7c38160e48078a` |
| `../inbox-idempotent-consumption/rollout.md` | `3362177702f58d2635adeccfba75c1c2b986550a461ba88e369a69e4012fbee2` |
| `spec.md` | `b1b57db365a10b73a4f86ad095aa66603ea947dca5e705f8594c83aee942c9ea` |
| `design/overview.md` | `1e6dff47249643ed9d28baba0ec36af6876008d1084a1508c5ec25b3dec34861` |
| `test-plan.md` | `e14396a666b1af81cade336b3b073c66c0aaa39b0ca77a702f9a25720d7b03c0` |
| `rollout.md` | `c6bb0cdbfe77da0de3343b7b8de0708ec119d5f93be7afd5b13db48a6c5d4f8a` |
| `../outbox-trace-continuity-and-key-lifecycle/spec.md` | `1e6142fce398e1444acd665138e8dc425dd1d049ed4b324f16c906dcdb1abd3a` |
| `../outbox-trace-continuity-and-key-lifecycle/design/overview.md` | `7ba5182e7ca8fe1b7eeb5c7f9aeb1c9c68bbe9577e170da1615d60c4daf9ef58` |
| `../outbox-trace-continuity-and-key-lifecycle/test-plan.md` | `3ae3f556f5dc3aed4313abf7dc0a29980b4a80a8b6e1e7cdb9e728c62f67fbac` |

| Accepted obligation | Reconciliation disposition |
| --- | --- |
| Trace R1-R2 capture/link/repetition/privacy and R4 key retirement | T1; TD-TRACE-001..003 and TD-TRACE-005..007 retain current runtime behavior and close only fixed proof/privacy deltas. |
| Trace R3 selected-adapter forwarding | T4 via TD-TRACE-004 / TD-OUTBOX-003. |
| Trace R5 ownership wording | Scope exit to the accepted production-closure reference adapter; T6 rechecks TD-TRACE-008. It is not completion evidence. Reopen its named Go Ownership owner only if depguard fails. |
| Closure R4 receipt and commit-unknown reconciliation | T2 via TD-OUTBOX-001..002. |
| Closure R3 sticky uncertainty, bounded quarantine, audited recovery, legacy transition, and observation | T3 via TD-OUTBOX-006..013 local deltas except TD-OUTBOX-011's NATS privacy command, which stays with its T4 transport owner; T7-T10 and T15 carry the fixed target deltas. |
| Closure R1 and R2 adapter classification, selected NATS composition, W3C flow, lifecycle, and outbox-only fail-closed startup | T4 via TD-OUTBOX-003..005; T3 carries R2's relay-disposition precedence through TD-OUTBOX-006; T14 carries the fixed target canary. |
| Inbox R1-R7 atomic claim/effect, logical identity, concurrency, no expiry, legal placement, and independent profile | T5 via TD-INBOX-001..009; T11-T13 carry TD-INBOX-010..012 target deltas. |
| Integrated production-closure and preserved invariants | T6 proves only the local deltas of TD-INBOX-012 and TD-OUTBOX-016 plus TD-OUTBOX-018 and TD-TRACE-008; T14 proves the target canary. |
| Inbox rollout Gates 1-4 | T11, T12, T13 in exact gate order, with T13 also consuming T7's relay-pause gate. |
| Outbox rollout Gates 1-6 | T7, T8, T9, T10, T14, T15 in exact gate order; T14 also consumes the inbox Gate 4 receipt. |
| Exactly-once delivery, external effects outside the claim transaction, generic broker abstraction, second attempt limit, tenant column, automatic key retirement, automatic inbox expiry/cleanup, caller metadata/source forwarding, and generic consumer ordering | Scope exits under the ready specs' named non-goals and reopen owners. None is implemented or used as completion evidence. |

Review disposition: Independent Task Review / Readiness of candidate SHA-256
`15109e769f98e8a755ed2d0b35eb0e363ba49ae04b749c3f1028d23160e5025a`
returned `PASS` with no surviving findings. The reviewer dry-ran T1 through
acceptance, confirmed the T2/T3 preservation gates and TD-OUTBOX-011 proof
placement, and ran no implementation command.
