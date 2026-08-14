# T5 Coverage Recovery — Test Design and Go Ownership Addendum

status: ready

## Fixed boundary

This is a narrow upstream Test Design plus Go Ownership recovery for the
frozen HTTP-idempotency T5 candidate. It retains the four existing test files
and their current cases as the first correction slice; the exact mapped owner
may extend them without deleting or weakening any current oracle:

- `internal/infra/postgresjobs/store_mapping_test.go`
- `internal/infra/postgresidempotency/store_classification_test.go`
- `cmd/internal/runtimeopts/runtimeopts_test.go`
- `internal/httpidempotency/decision_test.go`

The effective coverage floor remains `80.0`; its filter and exclusions remain
the current `Makefile` authority. The recorded candidate is `77.50%`, leaving
at least 257 effective statements. Integration-tag coverage cannot contribute
to that unit-only gate. No test may add dead code, an unreachable state, a
build-tag exclusion, a database/NATS mock framework, or a generic Store/client
abstraction merely to increase coverage.

Authorities retained without semantic change: HTTP idempotency T2/T3 and
TD-IDEM-005 through TD-IDEM-015A; Durable Jobs T2/T4 and TD-JOBS-007 through
TD-JOBS-014; and the durable NATS JetStream client/topology contract. The
existing PostgreSQL and NATS integration tests remain the exclusive carrier for
real arbitration, transaction visibility, row locks, writer authority, schema
readback, broker admission, stream inspection, consumer creation, delivery,
and reconnect behavior.

## Selected correction strategy

The correction unit exercises only deterministic package-owned decisions that
exist before a database/broker call or after an authoritative row/config value
has already been obtained. A unit fake may supply a local lifecycle signal or a
generated row value; it must not stand in for a PostgreSQL/NATS claim.

| Order | Owner and exact files | Executable behavior and deterministic oracle | Non-substitutable integration carrier |
| --- | --- | --- | --- |
| 1 | `internal/httpidempotency/{contract_test.go,identity_test.go,decision_test.go}` | Complete contract rejection matrix; independent clone isolation; result envelope/header decoding rejects malformed, duplicate, forbidden, truncated, oversized, and trailing values. Oracle is the approved `Contract` validation and byte round-trip, not Store behavior. | TD-IDEM-003 retained-result Store round-trip and TD-IDEM-008 replay/process proof. |
| 2 | `internal/infra/postgresidempotency/{store_classification_test.go,store_acquire_test.go,store_reconcile_test.go,store_maintenance_test.go}` | Closed decisions from retained `storedRow` values: corrupt/missing epoch/fingerprint/result, stale generation, recovery-due, lock-unavailable classification, snapshot freshness/headroom, and first-terminal-wins. Oracle is the closed `Reservation`/`Decision` or maintenance error, with no query attempted. | TD-IDEM-005 through TD-IDEM-012 force writer reads, locks, commits, recovery, and cleanup in PostgreSQL 17. |
| 3 | `internal/infra/postgresjobs/{store_mapping_test.go,store_rows_test.go}`; retain direct `claimedAttemptFromRow` proof and the existing package-private `requiredTime`/`revisionRows` helpers in `store_claim.go`; extract only private `rescueCandidatesFromRows` in `store_rescue.go`, called once after its existing SQLC query | Generated-row mapping rejects every nullable/negative/unknown value before it becomes a claimed attempt, claim result, rescue candidate, or acceptance readback. Oracle is the exact typed value or typed rejection; input is a fixed SQLC row, never a fake transaction. `store_rows.go` retains state/outcome/effect and acceptance-readback conversion; each Store stage retains its own query shape. | TD-JOBS-009/T10/T11/T14 real PostgreSQL acceptance, claim, operation-budget, finalize, and rescue schedules. |
| 4 | `internal/infra/natsjs/{client_test.go,worker_topology_test.go}` | Client-local ready/drain/terminal/reconnect state and pure stream/consumer configuration validation. Oracle is the explicit local state or desired configuration. Existing recording JetStream support may be used only for client lifecycle; it cannot prove stream existence, topology compatibility, consumer admission, delivery, or reconnect. | Existing NATS integration and messaging-race commands prove broker topology, consumer admission, delivery, drain, and reconnect. |
| 5 | `cmd/internal/runtimeopts/runtimeopts_test.go` | Preserve the existing complete config-to-adapter mapping, including blank-list retention for adapter-owned validation. Oracle is structural equality with the typed config. | NATS adapter configuration and runtime connection remain their existing integration/bootstrap proof. |

The first four-test candidate is retained. Extend an existing file before adding
a file. The only permitted production edit is extracting the current inline
rescue row decoder as private `rescueCandidatesFromRows` in `store_rescue.go` in
row 3. No interface, new package, exported symbol, adapter, or test-support
package is admitted.

## Proof matrix and completion

| ID | Wrong behavior rejected | Command after implementation | Completion oracle |
| --- | --- | --- | --- |
| CR-COV-01 | HTTP contract/result validation accepts a malformed retained semantic response or caller mutation changes an accepted declaration. | `go test -vet=off ./internal/httpidempotency -run '^(TestContract(DuplicateRiskPolicy|CloneDoesNotAliasDeclaration|ValidationRejectsIncompleteOrUnsafeDeclarations)|Test(EncodeResultRejectsResponsesThatCannotBeSafelyReplayed|DecodeResultRejects(CorruptOrOverlongRetainedData|InvalidRetainedFields)|ResultReaderRejectsTruncatedRetainedFields)|TestDecisionValidateEnforcesReplayResultOwnership|TestHTTPIdempotencyCanonicalVectors)$' -count=1` | Accepted/rejected value and encoded/decoded result agree with the closed contract. Result cases extend `identity_test.go`; no new sibling is added. |
| CR-COV-02 | A retained PostgreSQL row maps to execute/replay/mismatch/in-progress/unavailable/integrity contrary to T2/T3; local maintenance admits first work after terminal/stale/headroom loss. | `go test -vet=off -race ./internal/infra/postgresidempotency -run '^(Test(ClassifyRowPreservesIdempotencyOutcomes|ClassifyRowRejectsInvalidPersistedState|LockedReservationClassificationPreservesClosedDecisions|StoredRowsCopyDatabaseValues|StoreInputValidationAndClassificationBudget|MaintenanceScheduleAndBounds|MaintenanceFailureAndCapacityClosure|AcquireLockUnavailable|ReconcileClassificationErrorDecisions))$' -count=1` | Closed `Decision`, `Reservation`, and safety error match the supplied typed row/snapshot. |
| CR-COV-03 | A generated Jobs row becomes a valid claim/rescue/readback despite invalid identity, revision, vocabulary, timestamp, budget, or typed transition. | `go test -vet=off -race ./internal/infra/postgresjobs -run '^(Test(StoreMappingRejectsMalformedDatabaseValues|StoreClaimMappingPreservesOnlyValidRows|StoreDurationAndTimeMappingRejectInvalidValues|StoreRows(VocabularyIsClosedAndBijective|AcceptanceReadbackRequiresCompleteIdentity)|StoreRescueMappingRejectsMalformedRows|OperationErrorClassificationPreservesSessionSafety))$' -count=1` | Exact typed mapping or named rejection; `TestStoreRescueMappingRejectsMalformedRows` exercises the private post-query decoder. No database operation is constructed. |
| CR-COV-04 | A local NATS client reports ready after drain/terminal state or accepts a topology configuration that violates the declared static contract. | `go test -vet=off -race ./internal/infra/natsjs -run '^(Test(ClientStateTransitions|StreamContract|ExplicitAckPolicy))' -count=1` | Local state and desired stream/consumer config are exact; no broker claim is made. |
| CR-COV-05 | A shared composition root drops, normalizes, or silently validates an adapter field. | `go test -vet=off ./cmd/internal/runtimeopts -run '^(TestAdapterOptionsPreserveConfiguredValues|TestMessagingMappingRetainsBlankAddressesForAdapterValidation)$' -count=1` | Full `natsjs.Config` structural equality preserves URLs, credentials/root-CA paths, plaintext/authentication policy, stream, replication, retention, payload, and pending-publish bounds. |
| CR-COV-06 | The correction passes only by changing the denominator/filter or still misses the floor. | `test "$(shasum -a 256 Makefile | awk '{print $1}')" = '15db97ad2b79eaa3a13821fdc023e4f666ef721a39ed7184806c0211bfad0413'`; then `make check-full` | The current `Makefile` coverage authority is byte-identical before the aggregate; then unchanged `COVERAGE_MIN=80.0`, unchanged exclusions, and effective coverage `>=80.00%`; stop immediately at the first passing result. |

Existing non-substitutable commands remain required by their owners and are not
run as coverage substitutes: `REQUIRE_DOCKER=1 go test -tags=integration ./test
-run '^TestPostgresHTTPIdempotency(RecordedFingerprintVersion|FirstPublicationAndPoolHeadroom|ClassificationBudget|OwnerRecovery|CallerTransactionAtomicity|ReplayResultBoundAndPostCommitDeath|CommitReconciliation|RetentionAndCleanupRaces|WriterAuthority)$' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresJobs(Acceptance|Claim|Session|OperationBudget|Finalize|Recovery)$' -count=1`, and `make test-messaging-race`.

## Execution shape and reopen conditions

One fresh Local **Coverage Correction Acceptance-Unit Lead** owns this as one
serialized unit. The unit begins from the preserved four-test candidate, applies
the ordered file map above, and owns all formatting, the private rescue-decoder
extraction, and the one final aggregate. It must not create a second prerequisite
unit: every proposed test and the one extraction has an existing package owner.

Stop at the first unchanged effective coverage `>=80.00%`; do not add a sixth
test class merely for margin. If the final unchanged gate remains below 80.0,
preserve the candidate and reopen this addendum with the exact filtered uncovered
statement report. Reopen Go Ownership if a proposed unit test needs a fake
transaction/client or any new public/general seam. Reopen Test Design if an
uncovered statement cannot name a deterministic package-level oracle. Reopen
Technical Design, not this correction, if a PostgreSQL/NATS behavioral claim
would otherwise be moved out of its real integration carrier. The coverage
threshold and filter are never a repair target.

## Review-cleared receipt

The final candidate received independent read-only PASS reviews at
`gpt-5.6-terra` / `medium` for QA/Test Strategy, Go Ownership responsibility,
Go Ownership package/file cohesion, and Durable Jobs recovery/data boundary.
The reviews confirmed the explicit selectors, current helper placement, one
private rescue-decoder extraction, and the retained real PostgreSQL/NATS
carriers. This is a design handoff only; no production/test edit or proof ran.
