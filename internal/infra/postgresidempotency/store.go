package postgresidempotency

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

// FingerprintResolver reconstructs the current request under a retained
// fingerprint version. The Store never persists request input to do that work.
type FingerprintResolver func(string) (httpidempotency.Fingerprint, error)

// Store is the concrete PostgreSQL idempotency adapter.
type Store struct {
	pool          *postgres.Pool
	recoveryDelay time.Duration
	flights       publicationGroup
}

// NewStore constructs a Store with the deployment-owned owner-recovery delay.
func NewStore(pool *postgres.Pool, recoveryDelay time.Duration) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if recoveryDelay <= 0 {
		return nil, fmt.Errorf("%w: recovery delay must be positive", ErrConfig)
	}
	return &Store{
		pool:          pool,
		recoveryDelay: recoveryDelay,
		flights:       publicationGroup{waiting: make(map[[32]byte]chan struct{})},
	}, nil
}

func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.pool.PGX() != nil && s.recoveryDelay > 0
}

func validReservation(reservation httpidempotency.Reservation) bool {
	if reservation.Generation <= 0 || reservation.Attempt.Fingerprint.Version == "" {
		return false
	}
	switch reservation.Recovery {
	case httpidempotency.ReservationRecoveryNone,
		httpidempotency.ReservationRecoveryDue,
		httpidempotency.ReservationRecoveryReconciled:
		return true
	default:
		return false
	}
}

func validateAttempt(contract httpidempotency.Contract, attempt httpidempotency.Attempt, resolve FingerprintResolver) error {
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("%w: invalid contract", ErrConfig)
	}
	if err := attempt.Scope.Validate(); err != nil {
		return fmt.Errorf("%w: invalid scope", ErrConfig)
	}
	if attempt.Fingerprint.Version == "" || !slices.Contains(contract.FingerprintVersions, attempt.Fingerprint.Version) {
		return fmt.Errorf("%w: current fingerprint version is not declared", ErrConfig)
	}
	if resolve == nil {
		return fmt.Errorf("%w: fingerprint resolver is required", ErrConfig)
	}
	return nil
}

func identityBytes(attempt httpidempotency.Attempt) []byte {
	return attempt.Identity[:]
}

func fingerprintBytes(fingerprint httpidempotency.Fingerprint) []byte {
	return fingerprint.Digest[:]
}

func sameFingerprint(version string, digest []byte, candidate httpidempotency.Fingerprint) bool {
	return version == candidate.Version && len(digest) == len(candidate.Digest) &&
		subtle.ConstantTimeCompare(digest, candidate.Digest[:]) == 1
}

func resolveFingerprint(resolve FingerprintResolver, version string) (httpidempotency.Fingerprint, error) {
	fingerprint, err := resolve(version)
	if err != nil || fingerprint.Version != version {
		return httpidempotency.Fingerprint{}, fmt.Errorf("%w: retained fingerprint version is unavailable", ErrUnavailable)
	}
	return fingerprint, nil
}

func classificationContext(parent context.Context, wait time.Duration) (context.Context, context.CancelFunc, bool) {
	if deadline, ok := parent.Deadline(); ok && time.Until(deadline) <= wait {
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel, false
	}
	ctx, cancel := context.WithTimeout(parent, wait)
	return ctx, cancel, true
}

func classificationError(parent, budget context.Context, ownBudget bool, err error) error {
	if ownBudget && budget.Err() != nil && parent.Err() == nil {
		return fmt.Errorf("%w: in-progress classification budget exhausted", ErrUnavailable)
	}
	return err
}

func unavailable(ctx context.Context, stage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, stage)
}

func decisionForClassificationError(err error) (httpidempotency.Decision, bool) {
	switch {
	case err == nil:
		return httpidempotency.Decision{}, false
	case errors.Is(err, ErrEpochLost), errors.Is(err, ErrUnavailable):
		return httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, true
	case errors.Is(err, ErrIntegrityConflict):
		return httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, true
	default:
		return httpidempotency.Decision{}, false
	}
}
