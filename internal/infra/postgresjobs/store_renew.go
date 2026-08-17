package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type Renewal struct {
	Attempt         AttemptIdentity
	ObservedAt      time.Time
	LeaseExpiresAt  time.Time
	CancelRequested bool
}

func (s *Session) Renew(
	ctx context.Context,
	attempts []AttemptIdentity,
	leaseDuration time.Duration,
) (renewed []Renewal, err error) {
	params, err := attemptParams(attempts)
	if err != nil {
		return nil, err
	}
	leaseMicroseconds, err := durationMicroseconds("lease duration", leaseDuration)
	if err != nil {
		return nil, err
	}

	err = s.withOperation(ctx, "renew", pgx.ReadWrite, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		rows, queryErr := queries.RenewPostgresJobAttempts(operationCtx, sqlcgen.RenewPostgresJobAttemptsParams{
			LogicalJobIds: params.LogicalJobIDs, AttemptGenerations: params.AttemptGenerations,
			RecoveryGenerations: params.RecoveryGenerations, WorkerIds: params.WorkerIDs,
			LeaseMicroseconds: leaseMicroseconds,
		})
		if queryErr != nil {
			return fmt.Errorf("renew postgres job attempts: %w", queryErr)
		}
		renewed = make([]Renewal, 0, len(rows))
		for _, row := range rows {
			if row.WorkerID == nil {
				return fmt.Errorf("%w: renewed attempt has no worker", ErrUnknownVocabulary)
			}
			attempt, mapErr := attemptIdentity(row.LogicalJobID, row.AttemptGeneration, row.RecoveryGeneration, *row.WorkerID)
			if mapErr != nil {
				return mapErr
			}
			observedAt, mapErr := requiredTime("renew observed_at", row.ObservedAt)
			if mapErr != nil {
				return mapErr
			}
			leaseExpiresAt, mapErr := requiredTime("renew lease_expires_at", row.LeaseExpiresAt)
			if mapErr != nil {
				return mapErr
			}
			renewed = append(renewed, Renewal{
				Attempt: attempt, ObservedAt: observedAt, LeaseExpiresAt: leaseExpiresAt,
				CancelRequested: row.CancelRequested,
			})
		}
		return nil
	})
	return renewed, err
}
