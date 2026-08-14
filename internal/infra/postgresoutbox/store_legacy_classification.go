package postgresoutbox

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// ClassifyLegacyUncertainty converts one bounded batch of pre-upgrade NULL
// rows into the authoritative sticky/non-sticky state before relay startup.
//
// It is the one Store method that runs outside the delivery cycle and outside
// the maintenance timer: a rollout transition a service performs once against
// rows an older schema left behind, driven by its own bootstrap step rather
// than by Relay.runDueMaintenance. Keeping it beside the periodic observation
// and cleanup would put a one-shot upgrade path under a name that promises a
// timer.
func (s *Store) ClassifyLegacyUncertainty(ctx context.Context, maxAttempts, batchSize int) (classified int, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "classify_legacy", started, err) }()
	if !s.valid() {
		return 0, errStoreRequired()
	}
	if maxAttempts < 1 || maxAttempts > math.MaxInt32 || batchSize < 1 || batchSize > math.MaxInt32 {
		return 0, fmt.Errorf("%w: max attempts and classification batch size must be positive", ErrConfig)
	}
	var count int32
	err = s.pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var classifyErr error
		count, classifyErr = sqlcgen.New(tx).ClassifyLegacyOutboxUncertainty(ctx, sqlcgen.ClassifyLegacyOutboxUncertaintyParams{
			MaxAttempts: int32(maxAttempts), // #nosec G115 -- range checked above.
			BatchSize:   int32(batchSize),   // #nosec G115 -- range checked above.
		})
		if classifyErr != nil {
			return fmt.Errorf("classify legacy outbox uncertainty statement: %w", classifyErr)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("classify legacy outbox uncertainty: %w", err)
	}
	return int(count), nil
}
