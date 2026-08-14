# Gosec Prerequisite Recovery — Technical and Test Design Addendum

status: ready

## Fixed boundary

This is the smallest upstream Technical Design and Test Design recovery for
the saved `make check-full` Gosec blocker at
`/Users/daniil/Library/Application Support/rtk/tee/1786646023_make_check-full.log`.
It is a prerequisite for the frozen HTTP-idempotency T5 coverage candidate,
not a revision of that candidate or its effective-coverage policy.

Retained authorities:

- HTTP-idempotency T2/T3/T5, `TD-IDEM-003`, `TD-IDEM-005` through
  `TD-IDEM-012`, and `TD-IDEM-015A`;
- Durable Jobs T2/T4, `TD-JOBS-004`, `TD-JOBS-010`, `TD-JOBS-014`, and
  `TD-JOBS-015`;
- [T5 Coverage Recovery](t5-coverage-recovery.md), SHA-256
  `ba5d357f791e1b43fca213b2658e16a5b0afe3a4e199129f825d38b4a4365ec0`.

The coverage floor, Makefile filter/exclusions, public and wire contracts,
caller-owned transactions, writer-only correctness, Jobs state/revision and
generation semantics, profile generation, and all real PostgreSQL/NATS carriers
remain unchanged. No schema, SQL, migration, transaction, exported API, scanner
policy, scanner baseline, or blanket/path suppression is permitted.

## Selected recovery

The saved findings divide into already-guarded conversions and one
externally-constructible telemetry value. An adjacent, finding-local `#nosec`
explanation is correct only where the current fail-closed guard proves the
conversion. The explanatory comment does not replace that guard.

| Owner and paths | Fixed mechanism | Required preservation |
| --- | --- | --- |
| Jobs transition: `internal/jobs/transition.go` | At the SHA-256 jitter conversion, add `#nosec G115` naming the accepted `JitterPermille` range `[1,1000]`; the modulo result is at most 2000 before conversion to `int64`. | Keep the exact deterministic jitter vector and policy validation. |
| Jobs PostgreSQL adapter: `internal/infra/postgresjobs/store_claim.go`, `store_finalize.go`, `store_rescue.go` | Add finding-local `#nosec G115` explanations only: `Claim`/`RescueCandidates` bound their limits to `[1, math.MaxInt32]`; negative scope generation is rejected before its unsigned conversion; `validateAttemptIdentity` rejects either generation over `math.MaxInt64` before every signed SQLC projection. | Keep existing `ErrConfig`/`ErrUnknownVocabulary`, query shapes, Session ownership, and generation fencing. |
| Jobs telemetry: `internal/infra/postgresjobs/telemetry.go` and `telemetry_test.go` | Add one private count-to-Int64 conversion at the observable-gauge boundary. It returns `math.MaxInt64` for a `StateObservation.Count` above that value, otherwise the original count; annotate the now-local checked cast with `#nosec G115`. Add one cached-snapshot test with `math.MaxUint64`, asserting a non-negative saturated point. | Store-derived PostgreSQL `count(*)::bigint` values remain exact after the existing non-negative mapper check. Do not change `StateObservation`, query types, callback I/O, attributes, or readiness. |
| HTTP semantic envelope: `internal/httpidempotency/identity.go`, `result.go` | Add local `#nosec G115` explanations at the existing checked casts: identity/result fields reject lengths above `math.MaxUint32`, and result header/value counts reject values above `math.MaxUint16`. | Keep byte framing, canonical vectors, header allowlist, decode validation, and result bound unchanged. |
| HTTP OpenAPI registration: `internal/infra/http/idempotency_registration.go` | Add a local `#nosec G115` explanation to the required-key comparison: `Contract.Validate` has already rejected a non-positive `KeyMaxBytes`, so the positive `int` is representable as `uint64`. | Keep the exact required-header/`maxLength` comparison and registration failure. |
| Config test fixture: `internal/config/configtest/configtest.go` | Add one `#nosec G101` explanation immediately above the outbound-auth fixture map: its values are deterministic test-only placeholders installed through `testing.TB.Setenv`, not deployable credentials. | Keep the value names and source policy. Do not replace them with a live-looking value, a baseline entry, or scanner configuration. |

The rejected alternatives are: a generic checked-conversion package (duplicates
the local domain guards), a change of `StateObservation.Count` from `uint64` to
`int64` (widens an existing package-facing shape), changing any SQL/query type,
and global Gosec suppression. They add scope without closing a finding that the
selected local mechanisms do not close.

## Proof obligations

| ID | Wrong result rejected | Focused proof after correction | Independent oracle and retained carrier |
| --- | --- | --- | --- |
| SEC-G115-01 | A permitted jitter value, bounded claim/rescue limit, scope generation, or attempt generation truncates or changes the existing error before SQLC projection. | `go test -vet=off ./internal/jobs -run '^TestJobsTransition$' -count=1`; `go test -vet=off -race ./internal/infra/postgresjobs -run '^(TestStoreMappingRejectsMalformedDatabaseValues|TestStoreClaimMappingPreservesOnlyValidRows|TestStoreDurationAndTimeMappingRejectInvalidValues|TestStoreRescueMappingRejectsMalformedRows)$' -count=1` | Existing policy/range validators and typed mapping errors. TD-JOBS-010 claim and TD-JOBS-014 finalize/recovery PostgreSQL schedules remain the non-substitutable fencing/transaction oracles. |
| SEC-G115-02 | An in-memory count above `math.MaxInt64` becomes a negative or wrapped OTel gauge point. | `go test -vet=off -race ./internal/infra/postgresjobs -run '^TestTelemetryExportsCachedObservationWithoutStoreCalls$' -count=1` | Manual OTel reader observes the exact saturated `math.MaxInt64` point; the callback still performs no Store I/O. TD-JOBS-015 `TestPostgresJobsObservation` remains the independent SQL count/oldest oracle. |
| SEC-G115-03 | A bounded HTTP identity/result or required header is framed or accepted differently. | `go test -vet=off ./internal/httpidempotency -run '^(TestHTTPIdempotencyCanonicalVectors|TestEncodeResultRejectsResponsesThatCannotBeSafelyReplayed|TestDecodeResultRejects(CorruptOrOverlongRetainedData|InvalidRetainedFields)|TestResultReaderRejectsTruncatedRetainedFields)$' -count=1`; `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyRegistrationContract$' -count=1` | Existing byte vectors, retained-result rejection matrix, and OpenAPI required-key contract. TD-IDEM-003, TD-IDEM-005 through TD-IDEM-012, and TD-IDEM-015A retain their real PostgreSQL carriers. |
| SEC-G101-01 | A scanner exception hides a live credential or changes production configuration. | `make gosec` | The only exception is the precise `TB.Setenv` fixture annotation; scanner policy and all non-test credentials remain scanned. |
| SEC-AGG-01 | A partial repair is treated as T5 acceptance or the unchanged coverage result is reused across changed sources. | After all focused proofs, one serialized `make check-full` on the integrated tree. | The saved 80.10% result is invalid once any scanner-relevant source, test source, or final tree changes. The final aggregate must again prove its unchanged Makefile policy, race/static gates, coverage floor, and configured integration/container surfaces. |

Existing carriers remain non-substitutable and are not replaced with unit fakes:

```sh
REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotency(RecordedFingerprintVersion|FirstPublicationAndPoolHeadroom|ClassificationBudget|OwnerRecovery|CallerTransactionAtomicity|ReplayResultBoundAndPostCommitDeath|CommitReconciliation|RetentionAndCleanupRaces|WriterAuthority)$' -count=1
REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresJobs(Acceptance|Claim|Session|OperationBudget|Finalize|Recovery)$' -count=1
make test-messaging-race
```

## Correction routing and reopen conditions

The five path partitions above are mechanically independent, but they do not
form a safe concurrent Local wave: the checkout has broad unrelated dirt, the
frozen T5 candidate overlaps `store_rescue.go`, and none of the partial slices
can independently clear the repository-wide Gosec prerequisite. Planning must
therefore record one serial Local singleton, **Gosec prerequisite preserves
bounded conversions and telemetry observability**, with those partitions as its
ordered internal ownership map. Its Acceptance-Unit Lead alone integrates the
focused proofs and runs the one final aggregate.

The current ledgers are immutable in this phase. Next owner is a fresh Planning
Upstream Reopen Lead, which may materialize that singleton and its dependency
before the frozen Coverage Correction Lead or HTTP T5 may resume. No existing
candidate is accepted, staged, committed, or handed off by this addendum.

Reopen Go Security if a cited guard is removed/weakened, a `configtest` caller
can construct an outbound client or publish the fixture value, a live credential
is introduced, or a scanner exception cannot cite its adjacent invariant.
Reopen Go Ownership if a correction needs a shared converter, exported API,
changed query/schema/transaction owner, or another package as semantic owner.
Reopen Test Design if a required oracle needs a fake PostgreSQL/NATS claim or
moves an existing real integration claim into a unit test. Reopen Technical
Design if telemetry saturation would change its closed observable contract or
the caller-owned/writer-only boundaries.
