package postgresjobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestStoreMappingRejectsMalformedDatabaseValues(t *testing.T) {
	t.Parallel()

	attempt := testAttempt()
	validTransition := jobs.Transition{
		State: jobs.StateRetryWait, Delay: time.Microsecond + time.Nanosecond,
		AttemptsUsed: 2, ElapsedUsed: time.Millisecond + time.Nanosecond,
		Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone,
	}

	params, err := transitionParams(attempt, validTransition, "retryable")
	if err != nil {
		t.Fatalf("transitionParams() error = %v", err)
	}
	if params.DelayMicroseconds != 2 || params.ElapsedUsedMilliseconds != 2 {
		t.Fatalf("transitionParams() rounded = (%d, %d), want (2, 2)", params.DelayMicroseconds, params.ElapsedUsedMilliseconds)
	}

	for _, tc := range []struct {
		name       string
		attempt    AttemptIdentity
		transition jobs.Transition
		failure    string
	}{
		{name: "missing attempt", transition: validTransition},
		{name: "non-final state", attempt: attempt, transition: jobs.Transition{State: jobs.StateRunning, AttemptsUsed: 1, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectNone}},
		{name: "terminal delay", attempt: attempt, transition: jobs.Transition{State: jobs.StateSucceeded, Delay: time.Second, AttemptsUsed: 1, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}},
		{name: "impossible facts", attempt: attempt, transition: jobs.Transition{State: jobs.StateSucceeded, AttemptsUsed: 1, Outcome: jobs.OutcomePoison, Effect: jobs.EffectNone}},
		{name: "control failure code", attempt: attempt, transition: validTransition, failure: "retry\nagain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := transitionParams(tc.attempt, tc.transition, tc.failure); err == nil {
				t.Fatal("transitionParams() error = nil, want rejected database transition")
			}
		})
	}

	for _, tc := range []struct {
		name string
		row  transitionResultRow
	}{
		{name: "unknown status", row: transitionResultRow{Status: "other"}},
		{name: "missing final timestamp", row: transitionResultRow{Status: string(TransitionApplied), HasResult: true, FinalState: string(jobs.StateSucceeded), Outcome: string(jobs.OutcomeSuccess), EffectStatus: string(jobs.EffectCompleted), AttemptsUsed: 1}},
		{name: "negative persisted budget", row: transitionResultRow{Status: string(TransitionApplied), HasResult: true, FinalState: string(jobs.StateSucceeded), Outcome: string(jobs.OutcomeSuccess), EffectStatus: string(jobs.EffectCompleted), AttemptsUsed: 1, ElapsedUsedMilliseconds: -1, FinalizedAt: testTimestamp()}},
		{name: "retry without retry at", row: transitionResultRow{Status: string(TransitionApplied), HasResult: true, FinalState: string(jobs.StateRetryWait), Outcome: string(jobs.OutcomeRetryable), EffectStatus: string(jobs.EffectNone), AttemptsUsed: 1, FinalizedAt: testTimestamp()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := persistedTransition(tc.row); err == nil {
				t.Fatal("persistedTransition() error = nil, want rejected database row")
			}
		})
	}

	row := transitionResultRow{
		Status: string(TransitionApplied), HasResult: true, FinalState: string(jobs.StateRetryWait),
		Outcome: string(jobs.OutcomeRetryable), EffectStatus: string(jobs.EffectNone), AttemptsUsed: 2,
		ElapsedUsedMilliseconds: 3, RetryAt: testTimestamp(), FinalizedAt: testTimestamp(),
	}
	persisted, err := persistedTransition(row)
	if err != nil {
		t.Fatalf("persistedTransition() error = %v", err)
	}
	if persisted.Status != TransitionApplied || persisted.Transition != (jobs.Transition{State: jobs.StateRetryWait, AttemptsUsed: 2, ElapsedUsed: 3 * time.Millisecond, Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone}) || persisted.RetryAt.IsZero() {
		t.Fatalf("persistedTransition() = %#v, want complete retry transition", persisted)
	}
}

func TestStoreClaimMappingPreservesOnlyValidRows(t *testing.T) {
	t.Parallel()

	params, err := attemptParams([]AttemptIdentity{testAttempt()})
	if err != nil {
		t.Fatalf("attemptParams() error = %v", err)
	}
	if len(params.LogicalJobIDs) != 1 || params.LogicalJobIDs[0] != "job-1" || params.AttemptGenerations[0] != 1 {
		t.Fatalf("attemptParams() = %#v, want stored attempt identity", params)
	}
	if _, err := attemptParams([]AttemptIdentity{{}}); err == nil {
		t.Fatal("attemptParams() error = nil, want invalid identity rejected")
	}

	revisions := []jobs.Revision{{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, {Kind: "report", ArgsVersion: "v2", PolicyVersion: "p2"}}
	if _, _, _, err := registryColumns(revisions); err != nil {
		t.Fatalf("registryColumns() error = %v", err)
	}
	if _, _, _, err := registryColumns([]jobs.Revision{revisions[1], revisions[0]}); !errors.Is(err, jobs.ErrInvalidDefinition) {
		t.Fatalf("registryColumns() error = %v, want ErrInvalidDefinition", err)
	}
	if _, err := revisionRows([]string{"email"}, nil, []string{"p1"}); err == nil {
		t.Fatal("revisionRows() error = nil, want mismatched columns rejected")
	}

	valid := testClaimedRow()
	claim, err := claimedAttemptFromRow(valid)
	if err != nil {
		t.Fatalf("claimedAttemptFromRow() error = %v", err)
	}
	if claim.Attempt != testAttempt() || claim.Revision != revisions[0] || string(claim.Payload) != `{"kind":"email"}` {
		t.Fatalf("claimedAttemptFromRow() = %#v, want complete claimed attempt", claim)
	}
	valid.CurrentWorkerID = nil
	if _, err := claimedAttemptFromRow(valid); err == nil {
		t.Fatal("claimedAttemptFromRow() error = nil, want incomplete row rejected")
	}
}

func TestStoreRescueLimitColumnsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	revision := jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}
	if _, _, _, _, err := rescueLimitColumns(nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("rescueLimitColumns(nil) error = %v, want ErrConfig", err)
	}
	if _, _, _, _, err := rescueLimitColumns([]RescueLimit{{Revision: revision}}); !errors.Is(err, ErrConfig) {
		t.Fatalf("rescueLimitColumns(zero wave) error = %v, want ErrConfig", err)
	}
}

func TestStoreSchemaRequiresCapabilitiesAndAllowsAdditiveAuthority(t *testing.T) {
	t.Parallel()
	required := []string{"table.column|type", "table.constraint|hash"}
	if !schemaContains(append(append([]string(nil), required...), "table.future|type"), required) {
		t.Fatal("schemaContains() rejected additive authority")
	}
	if schemaContains(required[:1], required) || schemaContains([]string{"table.column|changed"}, required[:1]) {
		t.Fatal("schemaContains() accepted missing or changed required authority")
	}
}

func TestStoreClaimElapsedUsesOnlyDatabaseTimestamps(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		startedAt time.Time
	}{
		{name: "database behind worker", startedAt: time.Date(2000, time.January, 1, 0, 1, 0, 0, time.UTC)},
		{name: "database ahead of worker", startedAt: time.Date(2040, time.January, 1, 0, 1, 0, 0, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			row := testClaimedRow()
			row.BudgetStartedAt = pgtype.Timestamptz{Time: test.startedAt.Add(-time.Minute), Valid: true}
			row.AvailableAt = pgtype.Timestamptz{Time: test.startedAt.Add(-2 * time.Second), Valid: true}
			row.StartedAt = pgtype.Timestamptz{Time: test.startedAt, Valid: true}
			claim, err := claimedAttemptFromRow(row)
			if err != nil {
				t.Fatal(err)
			}
			if claim.BudgetElapsed != time.Minute || claim.QueueDelay != 2*time.Second {
				t.Fatalf("claim elapsed/queue = %s/%s, want 1m/2s from PostgreSQL timestamps", claim.BudgetElapsed, claim.QueueDelay)
			}
		})
	}
}

func TestStoreDurationAndTimeMappingRejectInvalidValues(t *testing.T) {
	t.Parallel()

	if got, err := durationMicroseconds("lease", time.Microsecond+time.Nanosecond); err != nil || got != 2 {
		t.Fatalf("durationMicroseconds() = (%d, %v), want (2, nil)", got, err)
	}
	if _, err := durationMicroseconds("lease", 0); err == nil {
		t.Fatal("durationMicroseconds() error = nil, want non-positive duration rejected")
	}
	if got, err := nonNegativeDurationMicroseconds(time.Nanosecond); err != nil || got != 1 {
		t.Fatalf("nonNegativeDurationMicroseconds() = (%d, %v), want (1, nil)", got, err)
	}
	if _, err := nonNegativeDurationMicroseconds(-time.Nanosecond); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("nonNegativeDurationMicroseconds() error = %v, want ErrInvalidTransition", err)
	}
	if _, err := requiredTime("stored_at", pgtype.Timestamptz{}); err == nil {
		t.Fatal("requiredTime() error = nil, want invalid database timestamp rejected")
	}
}

func TestStoreRescueMappingRejectsMalformedRows(t *testing.T) {
	t.Parallel()

	row := testRescueCandidateRow()
	candidates, err := rescueCandidatesFromRows([]sqlcgen.ListExpiredPostgresJobAttemptsRow{row})
	if err != nil {
		t.Fatalf("rescueCandidatesFromRows() error = %v", err)
	}
	if len(candidates) != 1 || candidates[0].Attempt != testAttempt() || candidates[0].Revision != (jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}) || candidates[0].State != jobs.StateRunning || candidates[0].Elapsed != 3*time.Millisecond {
		t.Fatalf("rescueCandidatesFromRows() = %#v, want complete rescue candidate", candidates)
	}

	for _, tc := range []struct {
		name   string
		update func(*sqlcgen.ListExpiredPostgresJobAttemptsRow)
	}{
		{name: "missing worker", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.CurrentWorkerID = nil }},
		{name: "invalid attempt count", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.AttemptsUsed = 0 }},
		{name: "negative elapsed", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.ElapsedMilliseconds = -1 }},
		{name: "invalid identity", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.LogicalJobID = "" }},
		{name: "invalid revision", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.PolicyVersion = "" }},
		{name: "unknown state", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.State = "other" }},
		{name: "missing budget start", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.BudgetStartedAt = pgtype.Timestamptz{} }},
		{name: "missing attempt start", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.StartedAt = pgtype.Timestamptz{} }},
		{name: "missing lease expiry", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.LeaseExpiresAt = pgtype.Timestamptz{} }},
		{name: "missing observation", update: func(row *sqlcgen.ListExpiredPostgresJobAttemptsRow) { row.ObservedAt = pgtype.Timestamptz{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invalid := row
			tc.update(&invalid)
			if _, err := rescueCandidatesFromRows([]sqlcgen.ListExpiredPostgresJobAttemptsRow{invalid}); err == nil {
				t.Fatal("rescueCandidatesFromRows() error = nil, want rejected rescue row")
			}
		})
	}
}

//nolint:contextcheck // The table deliberately controls the caller and operation cancellation independently.
func TestOperationErrorClassificationPreservesSessionSafety(t *testing.T) {
	t.Parallel()

	parentCanceled, cancelParent := context.WithCancel(t.Context())
	cancelParent()
	operationCanceled, cancelOperation := context.WithCancel(t.Context())
	cancelOperation()

	for _, tc := range []struct {
		name         string
		cancellation string
		connection   bool
		err          error
		want         error
		wantOriginal bool
	}{
		{name: "caller cancellation wins", cancellation: "parent", err: errors.New("query"), want: context.Canceled},
		{name: "postgres query cancellation is bounded", err: &pgconn.PgError{Code: pgerrcode.QueryCanceled}, want: ErrOperationTimeout},
		{name: "postgres lock timeout is bounded", err: &pgconn.PgError{Code: pgerrcode.LockNotAvailable}, want: ErrOperationTimeout},
		{name: "operation deadline is bounded", cancellation: "operation", err: errors.New("query"), want: ErrOperationTimeout},
		{name: "closed connection retires session", connection: true, err: errors.New("connection reset"), want: ErrSessionTerminal},
		{name: "unclassified database error survives", err: errors.New("syntax"), wantOriginal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got error
			switch tc.cancellation {
			case "parent":
				got = classifyOperationError(parentCanceled, t.Context(), tc.connection, tc.err)
			case "operation":
				got = classifyOperationError(t.Context(), operationCanceled, tc.connection, tc.err)
			default:
				parent, cancelParent := context.WithCancel(t.Context())
				t.Cleanup(cancelParent)
				operation, cancelOperation := context.WithCancel(parent)
				t.Cleanup(cancelOperation)
				got = classifyOperationError(parent, operation, tc.connection, tc.err)
			}
			if tc.wantOriginal {
				if !errors.Is(got, tc.err) {
					t.Fatalf("classifyOperationError() = %v, want original %v", got, tc.err)
				}
				return
			}
			if !errors.Is(got, tc.want) {
				t.Fatalf("classifyOperationError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRetainedVocabularyMappingRejectsIncompleteOrUnsupportedRows(t *testing.T) {
	t.Parallel()
	revisions, err := revisionRows([]string{"email", "webhook"}, []string{"v1", "v2"}, []string{"p1", "p2"})
	if err != nil || len(revisions) != 2 || revisions[1].Kind != "webhook" {
		t.Fatalf("revisionRows() = %#v, %v", revisions, err)
	}
	if _, err := revisionRows([]string{"email"}, []string{""}, []string{"p1"}); err == nil {
		t.Fatal("revisionRows() accepted an invalid retained revision")
	}
	if err := unsupportedRevisions(revisions); !errors.Is(err, jobs.ErrUnsupportedRevision) || !strings.Contains(err.Error(), "email/v1/p1,webhook/v2/p2") {
		t.Fatalf("unsupportedRevisions() error = %v", err)
	}

	row := testClaimedRow()
	invalidIdentity := row
	invalidIdentity.ProducerKey = new("")
	if _, err := claimedAttemptFromRow(invalidIdentity); err == nil {
		t.Fatal("claimedAttemptFromRow() accepted an invalid retained identity")
	}
	invalidRevision := row
	invalidRevision.PolicyVersion = new("")
	if _, err := claimedAttemptFromRow(invalidRevision); err == nil {
		t.Fatal("claimedAttemptFromRow() accepted an invalid retained revision")
	}
	invalidAttempts := row
	invalidAttempts.AttemptsUsed = new(int32(0))
	if _, err := claimedAttemptFromRow(invalidAttempts); err == nil {
		t.Fatal("claimedAttemptFromRow() accepted a zero retained attempt count")
	}

	completed := transitionResultRow{
		Status: string(TransitionApplied), HasResult: true, FinalState: string(jobs.StateSucceeded),
		Outcome: string(jobs.OutcomeSuccess), EffectStatus: string(jobs.EffectNone), AttemptsUsed: 1,
		FinalizedAt: testTimestamp(),
	}
	if transition, err := persistedTransition(completed); err != nil || transition.Transition.State != jobs.StateSucceeded {
		t.Fatalf("persistedTransition() = %#v, %v", transition, err)
	}
	for _, test := range []struct {
		name   string
		update func(*transitionResultRow)
	}{
		{name: "invalid state", update: func(row *transitionResultRow) { row.FinalState = "other" }},
		{name: "invalid outcome", update: func(row *transitionResultRow) { row.Outcome = "other" }},
		{name: "invalid effect", update: func(row *transitionResultRow) { row.EffectStatus = "other" }},
		{name: "invalid budget", update: func(row *transitionResultRow) { row.AttemptsUsed = 0 }},
		{name: "missing finalized time", update: func(row *transitionResultRow) { row.FinalizedAt = pgtype.Timestamptz{} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			row := completed
			test.update(&row)
			if _, err := persistedTransition(row); err == nil {
				t.Fatal("persistedTransition() error = nil")
			}
		})
	}
}

func testAttempt() AttemptIdentity {
	return AttemptIdentity{LogicalJobID: "job-1", AttemptGeneration: 1, WorkerID: "worker-1"}
}

func testTimestamp() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC), Valid: true}
}

func testClaimedRow() sqlcgen.ClaimPostgresJobsRow {
	logicalJobID, producerScope, producerKey := "job-1", "orders", "producer-1"
	occurrenceScope, occurrenceID, effectScope, effectKey := "orders", "occurrence-1", "orders", "effect-1"
	kind, argsVersion, policyVersion, workerID := "email", "v1", "p1", "worker-1"
	recoveryGeneration, attemptGeneration := int64(0), int64(1)
	attemptsUsed := int32(1)
	return sqlcgen.ClaimPostgresJobsRow{
		LogicalJobID: &logicalJobID, ProducerScope: &producerScope, ProducerKey: &producerKey,
		OccurrenceScope: &occurrenceScope, OccurrenceID: &occurrenceID, EffectScope: &effectScope, EffectKey: &effectKey,
		Kind: &kind, ArgsVersion: &argsVersion, PolicyVersion: &policyVersion, Payload: []byte(`{"kind":"email"}`),
		RecoveryGeneration: &recoveryGeneration, AttemptGeneration: &attemptGeneration, AttemptsUsed: &attemptsUsed,
		BudgetStartedAt: testTimestamp(), AvailableAt: testTimestamp(), CurrentWorkerID: &workerID,
		StartedAt: testTimestamp(), LeaseExpiresAt: testTimestamp(),
	}
}

func testRescueCandidateRow() sqlcgen.ListExpiredPostgresJobAttemptsRow {
	workerID := "worker-1"
	return sqlcgen.ListExpiredPostgresJobAttemptsRow{
		LogicalJobID: "job-1", Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1", State: string(jobs.StateRunning),
		AttemptGeneration: 1, AttemptsUsed: 2, CurrentWorkerID: &workerID,
		BudgetStartedAt: testTimestamp(), StartedAt: testTimestamp(), LeaseExpiresAt: testTimestamp(), ObservedAt: testTimestamp(), ElapsedMilliseconds: 3,
	}
}
