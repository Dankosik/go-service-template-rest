package postgresjobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Stage stores one prepared job in the caller's transaction. It never begins,
// commits, rolls back, or retries that transaction.
func (s *Store) Stage(ctx context.Context, tx pgx.Tx, prepared jobs.Prepared) (result jobs.StageResult, err error) {
	rejected := jobs.StageResult{Outcome: jobs.StageRejected}
	result = rejected
	defer func() { s.recordAcceptance(ctx, result, err) }()
	if !s.valid() {
		return result, fmt.Errorf("%w: Store is required", ErrConfig)
	}
	if tx == nil {
		return result, fmt.Errorf("%w: transaction is required", ErrConfig)
	}
	if err := prepared.Validate(); err != nil {
		return result, fmt.Errorf("validate prepared job: %w", err)
	}

	identity := prepared.Identity()
	revision := prepared.Revision()
	fingerprint := prepared.Fingerprint()
	queries := sqlcgen.New(tx)
	logicalJobID, err := queries.InsertPostgresJobAcceptance(ctx, sqlcgen.InsertPostgresJobAcceptanceParams{
		LogicalJobID:      string(identity.LogicalJobID),
		ProducerScope:     string(identity.ProducerScope),
		ProducerKey:       string(identity.ProducerKey),
		OccurrenceScope:   string(identity.OccurrenceScope),
		OccurrenceID:      string(identity.OccurrenceID),
		EffectScope:       string(identity.EffectScope),
		EffectKey:         string(identity.EffectKey),
		IntentFingerprint: fingerprint[:],
		Kind:              revision.Kind,
		ArgsVersion:       revision.ArgsVersion,
		PolicyVersion:     revision.PolicyVersion,
		Payload:           prepared.Payload(),
		AvailableAt:       pgtype.Timestamptz{Time: prepared.AvailableAt(), Valid: true},
	})
	if err == nil {
		return jobs.StageResult{Outcome: jobs.StageNew, LogicalJobID: jobs.LogicalJobID(logicalJobID)}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return result, fmt.Errorf("insert postgres jobs acceptance: %w", err)
	}

	conflicts, err := queries.ListPostgresJobsAcceptanceConflicts(ctx, sqlcgen.ListPostgresJobsAcceptanceConflictsParams{
		LogicalJobID:    string(identity.LogicalJobID),
		ProducerScope:   string(identity.ProducerScope),
		ProducerKey:     string(identity.ProducerKey),
		OccurrenceScope: string(identity.OccurrenceScope),
		OccurrenceID:    string(identity.OccurrenceID),
		EffectScope:     string(identity.EffectScope),
		EffectKey:       string(identity.EffectKey),
	})
	if err != nil {
		return result, fmt.Errorf("read postgres jobs acceptance conflict: %w", err)
	}
	if len(conflicts) == 0 {
		return result, errors.New("insert postgres jobs acceptance reported a conflict without retained authority")
	}
	if len(conflicts) == 1 && bytes.Equal(conflicts[0].IntentFingerprint, fingerprint[:]) {
		return jobs.StageResult{Outcome: jobs.StageExisting, LogicalJobID: jobs.LogicalJobID(conflicts[0].LogicalJobID)}, nil
	}
	return jobs.StageResult{Outcome: jobs.StageConflict, LogicalJobID: jobs.LogicalJobID(conflicts[0].LogicalJobID)}, nil
}

// ResolveAcceptance classifies a lost commit result from the configured writer.
// Only a successful writer read may make absence decisive.
func (s *Store) ResolveAcceptance(ctx context.Context, expected jobs.ReadbackExpectation) (jobs.ReadbackResult, error) {
	unknown := jobs.ReadbackResult{Outcome: jobs.ReadbackUnknown}
	if !s.valid() {
		return unknown, fmt.Errorf("%w: Store is required", ErrConfig)
	}
	if err := expected.Validate(); err != nil {
		return unknown, fmt.Errorf("validate acceptance readback: %w", err)
	}
	identity := expected.Identity()

	readbackCtx, cancel := context.WithTimeout(ctx, min(s.operationTimeout, s.statementTimeout))
	defer cancel()
	conn, err := s.pool.Acquire(readbackCtx)
	if err != nil {
		return unknown, fmt.Errorf("acquire postgres jobs acceptance readback: %w", err)
	}
	defer conn.Release()
	row, err := sqlcgen.New(conn).ReadPostgresJobsAcceptance(readbackCtx, sqlcgen.ReadPostgresJobsAcceptanceParams{
		ProducerScope: string(identity.ProducerScope),
		ProducerKey:   string(identity.ProducerKey),
	})
	if err != nil {
		return unknown, fmt.Errorf("read postgres jobs acceptance: %w", err)
	}
	if !row.WriterPrimary {
		return unknown, errors.New("read postgres jobs acceptance: PostgreSQL endpoint is not a writable primary")
	}
	if !row.RowExists {
		return jobs.ReadbackResult{Outcome: jobs.ReadbackNotAccepted}, nil
	}
	stored, ok := acceptanceIdentityFromReadback(row)
	if !ok {
		return unknown, errors.New("read postgres jobs acceptance: retained identity is incomplete")
	}
	expectedFingerprint := expected.Fingerprint()
	if stored == identity && bytes.Equal(row.IntentFingerprint, expectedFingerprint[:]) {
		return jobs.ReadbackResult{Outcome: jobs.ReadbackAccepted, LogicalJobID: stored.LogicalJobID}, nil
	}
	return jobs.ReadbackResult{Outcome: jobs.ReadbackConflict, LogicalJobID: stored.LogicalJobID}, nil
}
