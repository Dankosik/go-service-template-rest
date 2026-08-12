package postgresidempotency

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
)

// Release removes only the generation that definitely rolled back. A successor
// can therefore never be released by an older owner.
func (s *Store) Release(ctx context.Context, reservation httpidempotency.Reservation) error {
	if !s.valid() || !validReservation(reservation) {
		return fmt.Errorf("%w: store and reservation are required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return unavailable(ctx, "release connection")
	}
	defer conn.Release()
	queries := sqlcgen.New(conn)
	writer, err := queries.CheckHTTPIdempotencyWriter(ctx)
	if err != nil {
		return unavailable(ctx, "check release writer")
	}
	if !writer {
		return ErrUnavailable
	}
	if _, err := queries.ReleaseHTTPIdempotencyReservation(ctx, sqlcgen.ReleaseHTTPIdempotencyReservationParams{
		IdentityToken: identityBytes(reservation.Attempt),
		Generation:    reservation.Generation,
	}); err != nil {
		return unavailable(ctx, "release reservation")
	}
	return nil
}
