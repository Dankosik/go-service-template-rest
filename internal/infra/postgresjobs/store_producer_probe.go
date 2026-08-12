package postgresjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// CheckProducerPath verifies the complete read-only schema and writer authority
// used by Stage and ResolveAcceptance. Pool saturation keeps the service's
// existing capacity-only readiness meaning.
func (s *Store) CheckProducerPath(ctx context.Context) error {
	session, err := s.AcquireSession(ctx)
	if errors.Is(err, postgres.ErrSaturated) {
		return nil
	}
	if err != nil {
		return err
	}
	defer session.Release(ctx)

	if err := session.CheckSchema(ctx); err != nil {
		return err
	}
	return session.checkProducerAuthority(ctx)
}

func (s *Session) checkProducerAuthority(ctx context.Context) error {
	// A bare BEGIN preserves the endpoint's default read-only state. READ ONLY
	// would hide writers; READ WRITE would mask a read-only primary session.
	return s.withOperation(ctx, pgx.TxAccessMode(""), func(ctx context.Context, queries *sqlcgen.Queries) error {
		authority, err := queries.CheckPostgresJobsProducerAuthority(ctx)
		if err != nil {
			return fmt.Errorf("check postgres jobs producer authority: %w", err)
		}
		if !authority.WriterPrimary {
			return errors.New("postgres jobs producer path is not a writable primary")
		}
		if !authority.ProducerPrivileges {
			return errors.New("postgres jobs producer path lacks SELECT or INSERT privilege")
		}
		return nil
	})
}
