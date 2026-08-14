package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

func advanceClock(ctx context.Context, queries *sqlcgen.Queries) (time.Time, error) {
	value, err := queries.AdvanceWebhookClock(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrClockRegression
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("advance webhook clock: %w", err)
	}
	if !value.Valid {
		return time.Time{}, fmt.Errorf("%w: webhook clock returned no time", ErrConflict)
	}
	return value.Time.UTC(), nil
}

type ClockObservation struct {
	HighWater  time.Time
	Regression bool
	ObservedAt time.Time
}

func (s *Store) ObserveClock(ctx context.Context) (ClockObservation, error) {
	var observation ClockObservation
	err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).ObserveWebhookClock(ctx)
		if err != nil {
			return fmt.Errorf("observe webhook clock: %w", err)
		}
		observation = ClockObservation{HighWater: row.HighWater.Time.UTC(), Regression: row.Regression, ObservedAt: row.ObservedAt.Time.UTC()}
		return nil
	})
	return observation, err
}
