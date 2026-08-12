package postgresidempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// Complete writes the bounded terminal result as the final application
// statement in the caller-owned transaction.
func (s *Store) Complete(
	ctx context.Context,
	tx pgx.Tx,
	contract httpidempotency.Contract,
	reservation httpidempotency.Reservation,
	result httpidempotency.Result,
) error {
	if !s.valid() || tx == nil || !validReservation(reservation) {
		return fmt.Errorf("%w: store, transaction, and reservation are required", ErrConfig)
	}
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("%w: invalid contract", ErrConfig)
	}
	encoded, err := httpidempotency.EncodeResult(contract, result)
	if err != nil {
		return fmt.Errorf("%w: encode result", ErrResultTooLarge)
	}
	var duplicateRiskNanos *int64
	if !contract.DuplicateRisk.Permanent {
		duration := int64(contract.DuplicateRisk.Duration)
		duplicateRiskNanos = &duration
	}
	_, err = sqlcgen.New(tx).CompleteHTTPIdempotencyReservation(ctx, sqlcgen.CompleteHTTPIdempotencyReservationParams{
		FingerprintVersion:     reservation.Attempt.Fingerprint.Version,
		Fingerprint:            fingerprintBytes(reservation.Attempt.Fingerprint),
		Result:                 encoded,
		ResultMaxBytes:         int64(contract.ResultMaxBytes),
		ReplayNanos:            int64(contract.ReplayTTL),
		DuplicateRiskNanos:     duplicateRiskNanos,
		DuplicateRiskPermanent: contract.DuplicateRisk.Permanent,
		IdentityToken:          identityBytes(reservation.Attempt),
		Generation:             reservation.Generation,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReservationLost
	}
	if err != nil {
		return unavailable(ctx, "complete reservation")
	}
	return nil
}
