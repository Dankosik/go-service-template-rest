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
	// ponytail: the singleton is the commit-order safety boundary; replace it only
	// with a separately designed monotone authority if target contention exceeds budget.
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
