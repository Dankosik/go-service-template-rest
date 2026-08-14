package postgresidempotency

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

// FingerprintResolver reconstructs the current request under a retained
// fingerprint version. The Store never persists request input to do that work.
type FingerprintResolver func(string) (httpidempotency.Fingerprint, error)

// StoreOptions are the Store-owned runtime safety bounds. None has a template
// default; bootstrap supplies one validated active-registration value.
type StoreOptions struct {
	OwnerRecoveryDelay     time.Duration
	CleanupBatchSize       int
	MaxMaintenanceLag      time.Duration
	MaxRelationBytes       int64
	AdmissionHeadroomBytes int64
}

func (o StoreOptions) validate() error {
	if o.OwnerRecoveryDelay <= 0 {
		return fmt.Errorf("%w: owner_recovery_delay must be positive", ErrConfig)
	}
	if o.CleanupBatchSize <= 0 {
		return fmt.Errorf("%w: cleanup_batch_size must be positive", ErrConfig)
	}
	if o.MaxMaintenanceLag <= 0 {
		return fmt.Errorf("%w: max_maintenance_lag must be positive", ErrConfig)
	}
	if o.MaxRelationBytes <= 0 {
		return fmt.Errorf("%w: max_relation_bytes must be positive", ErrConfig)
	}
	if o.AdmissionHeadroomBytes <= 0 {
		return fmt.Errorf("%w: admission_headroom_bytes must be positive", ErrConfig)
	}
	if o.AdmissionHeadroomBytes >= o.MaxRelationBytes {
		return fmt.Errorf("%w: admission_headroom_bytes must be less than max_relation_bytes", ErrConfig)
	}
	return nil
}

// Store is the concrete PostgreSQL idempotency adapter.
type Store struct {
	pool              *postgres.Pool
	options           StoreOptions
	recoveryDelay     time.Duration
	flights           publicationGroup
	maintenanceMu     sync.Mutex
	maintenanceFailed bool
	safety            atomic.Pointer[maintenanceSnapshot]
	terminalErrors    chan error
	telemetry         *storeTelemetry
}

// NewStore constructs a Store with its deployment-owned runtime safety bounds.
func NewStore(pool *postgres.Pool, options StoreOptions) (*Store, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	store := &Store{
		pool:           pool,
		options:        options,
		recoveryDelay:  options.OwnerRecoveryDelay,
		flights:        publicationGroup{waiting: make(map[[32]byte]chan struct{})},
		terminalErrors: make(chan error, 1),
	}
	store.safety.Store(&maintenanceSnapshot{})
	telemetry, err := newStoreTelemetry(store)
	if err != nil {
		return nil, err
	}
	store.telemetry = telemetry
	return store, nil
}

// TerminalErrors reports the first request-discovered integrity failure to the
// existing maintenance supervisor. Store safety remains the readiness authority.
func (s *Store) TerminalErrors() <-chan error {
	if s == nil {
		return nil
	}
	return s.terminalErrors
}

// ObserveTerminal records the one final outcome owned by an admitted request.
func (s *Store) ObserveTerminal(ctx context.Context, decision httpidempotency.Decision, err error) {
	if s == nil {
		return
	}
	if s.telemetry != nil {
		s.telemetry.recordTerminal(ctx, terminalOutcome(decision, err))
	}
	if errors.Is(err, ErrEpochLost) || errors.Is(err, ErrIntegrityConflict) {
		s.markTerminal(err)
	}
}

func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.pool.PGX() != nil && s.options.validate() == nil
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
		return fmt.Errorf("%s: %w", stage, err)
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
