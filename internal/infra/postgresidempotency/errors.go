package postgresidempotency

import "errors"

var (
	// ErrConfig reports an incomplete Store call before it can reach PostgreSQL.
	ErrConfig = errors.New("postgres idempotency store config")
	// ErrUnavailable means the writer could not make a safe classification.
	ErrUnavailable = errors.New("postgres idempotency store unavailable")
	// ErrIntegrityConflict means retained state cannot safely describe one attempt.
	ErrIntegrityConflict = errors.New("postgres idempotency integrity conflict")
	// ErrReservationLost means a caller no longer owns its provisional generation.
	ErrReservationLost = errors.New("postgres idempotency reservation lost")
	// ErrResultTooLarge rejects a result before the caller-owned transaction commits.
	ErrResultTooLarge = errors.New("postgres idempotency result too large")
	// ErrEpochLost means a completed row no longer has an exact physical commit time.
	ErrEpochLost = errors.New("postgres idempotency commit epoch lost")
)
