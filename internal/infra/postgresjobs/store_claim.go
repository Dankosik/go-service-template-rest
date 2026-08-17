package postgresjobs

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AttemptIdentity struct {
	LogicalJobID       jobs.LogicalJobID
	AttemptGeneration  uint64
	RecoveryGeneration uint64
	WorkerID           string
}

type ClaimOptions struct {
	RegistryKeys  []jobs.Revision
	WorkerID      string
	Limit         int
	LeaseDuration time.Duration
}

type ClaimedAttempt struct {
	Identity        jobs.AcceptanceIdentity
	Revision        jobs.Revision
	Payload         []byte
	Attempt         AttemptIdentity
	AttemptNumber   uint32
	BudgetStartedAt time.Time
	BudgetElapsed   time.Duration
	QueueDelay      time.Duration
	StartedAt       time.Time
	LeaseExpiresAt  time.Time
}

type ClaimResult struct {
	ObservedAt      time.Time
	Paused          bool
	ScopeGeneration uint64
	Attempts        []ClaimedAttempt
}

type ClaimResolution struct {
	Attempt   AttemptIdentity
	Committed bool
}

func (s *Session) Claim(ctx context.Context, options ClaimOptions) (result ClaimResult, err error) {
	kinds, argsVersions, policyVersions, err := registryColumns(options.RegistryKeys)
	if err != nil {
		return ClaimResult{}, err
	}
	if err := validateStoreToken("worker_id", options.WorkerID); err != nil {
		return ClaimResult{}, err
	}
	if options.Limit < 1 || options.Limit > math.MaxInt32 {
		return ClaimResult{}, fmt.Errorf("%w: claim limit must be between 1 and %d", ErrConfig, math.MaxInt32)
	}
	leaseMicroseconds, err := durationMicroseconds("lease duration", options.LeaseDuration)
	if err != nil {
		return ClaimResult{}, err
	}

	err = s.withOperation(ctx, "claim", pgx.ReadWrite, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		rows, queryErr := queries.ClaimPostgresJobs(operationCtx, sqlcgen.ClaimPostgresJobsParams{
			Kinds: kinds, ArgsVersions: argsVersions, PolicyVersions: policyVersions,
			ClaimLimit: int32(options.Limit), WorkerID: options.WorkerID, // #nosec G115 -- claim limit is validated in [1, math.MaxInt32].
			LeaseMicroseconds: leaseMicroseconds,
		})
		if queryErr != nil {
			return fmt.Errorf("claim postgres jobs: %w", queryErr)
		}
		if len(rows) == 0 {
			return fmt.Errorf("%w: claim returned no scope row", ErrSchemaIncompatible)
		}
		observedAt, mapErr := requiredTime("claim observed_at", rows[0].ObservedAt)
		if mapErr != nil {
			return mapErr
		}
		if rows[0].ScopeGeneration < 0 {
			return fmt.Errorf("%w: negative scope generation", ErrUnknownVocabulary)
		}
		result = ClaimResult{
			ObservedAt: observedAt, Paused: rows[0].Paused,
			ScopeGeneration: uint64(rows[0].ScopeGeneration), // #nosec G115 -- negative scope generations are rejected before conversion.
		}
		if rows[0].Compatible == nil {
			return fmt.Errorf("%w: missing compatibility result", ErrUnknownVocabulary)
		}
		if !*rows[0].Compatible {
			missing, missingErr := revisionRows(
				rows[0].MissingKinds,
				rows[0].MissingArgsVersions,
				rows[0].MissingPolicyVersions,
			)
			if missingErr != nil {
				return missingErr
			}
			return unsupportedRevisions(missing)
		}
		for _, row := range rows {
			if row.LogicalJobID == nil {
				continue
			}
			claim, claimErr := claimedAttemptFromRow(row)
			if claimErr != nil {
				return claimErr
			}
			result.Attempts = append(result.Attempts, claim)
		}
		return nil
	})
	return result, err
}

func (s *Session) ResolveClaims(ctx context.Context, attempts []AttemptIdentity) (resolved []ClaimResolution, err error) {
	params, err := attemptParams(attempts)
	if err != nil {
		return nil, err
	}
	err = s.withOperation(ctx, "resolve_claims", pgx.ReadOnly, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		rows, queryErr := queries.ResolvePostgresJobsClaims(operationCtx, sqlcgen.ResolvePostgresJobsClaimsParams{
			LogicalJobIds: params.LogicalJobIDs, AttemptGenerations: params.AttemptGenerations,
			RecoveryGenerations: params.RecoveryGenerations, WorkerIds: params.WorkerIDs,
		})
		if queryErr != nil {
			return fmt.Errorf("resolve postgres jobs claims: %w", queryErr)
		}
		resolved = make([]ClaimResolution, 0, len(rows))
		for _, row := range rows {
			attempt, mapErr := attemptIdentity(row.LogicalJobID, row.AttemptGeneration, row.RecoveryGeneration, row.WorkerID)
			if mapErr != nil {
				return mapErr
			}
			resolved = append(resolved, ClaimResolution{Attempt: attempt, Committed: row.Committed})
		}
		return nil
	})
	return resolved, err
}

type attemptQueryParams struct {
	LogicalJobIDs       []string
	AttemptGenerations  []int64
	RecoveryGenerations []int64
	WorkerIDs           []string
}

func attemptParams(attempts []AttemptIdentity) (attemptQueryParams, error) {
	params := attemptQueryParams{
		LogicalJobIDs: make([]string, len(attempts)), AttemptGenerations: make([]int64, len(attempts)),
		RecoveryGenerations: make([]int64, len(attempts)), WorkerIDs: make([]string, len(attempts)),
	}
	for index, attempt := range attempts {
		if err := validateAttemptIdentity(attempt); err != nil {
			return attemptQueryParams{}, err
		}
		params.LogicalJobIDs[index] = string(attempt.LogicalJobID)
		params.AttemptGenerations[index] = int64(attempt.AttemptGeneration)   // #nosec G115 -- validateAttemptIdentity rejects generations above math.MaxInt64.
		params.RecoveryGenerations[index] = int64(attempt.RecoveryGeneration) // #nosec G115 -- validateAttemptIdentity rejects generations above math.MaxInt64.
		params.WorkerIDs[index] = attempt.WorkerID
	}
	return params, nil
}

//nolint:cyclop // A generated nullable row needs explicit validation before it becomes domain data.
func claimedAttemptFromRow(row sqlcgen.ClaimPostgresJobsRow) (ClaimedAttempt, error) {
	if row.LogicalJobID == nil || row.ProducerScope == nil || row.ProducerKey == nil ||
		row.OccurrenceScope == nil || row.OccurrenceID == nil || row.EffectScope == nil || row.EffectKey == nil ||
		row.Kind == nil || row.ArgsVersion == nil || row.PolicyVersion == nil || row.RecoveryGeneration == nil ||
		row.AttemptGeneration == nil || row.AttemptsUsed == nil || row.CurrentWorkerID == nil {
		return ClaimedAttempt{}, fmt.Errorf("%w: incomplete claimed row", ErrUnknownVocabulary)
	}
	identity := jobs.AcceptanceIdentity{
		LogicalJobID: jobs.LogicalJobID(*row.LogicalJobID), ProducerScope: jobs.ProducerScope(*row.ProducerScope),
		ProducerKey: jobs.ProducerKey(*row.ProducerKey), OccurrenceScope: jobs.OccurrenceScope(*row.OccurrenceScope),
		OccurrenceID: jobs.OccurrenceID(*row.OccurrenceID), EffectScope: jobs.EffectScope(*row.EffectScope),
		EffectKey: jobs.EffectKey(*row.EffectKey),
	}
	if err := identity.Validate(); err != nil {
		return ClaimedAttempt{}, fmt.Errorf("validate claimed identity: %w", err)
	}
	revision := jobs.Revision{Kind: *row.Kind, ArgsVersion: *row.ArgsVersion, PolicyVersion: *row.PolicyVersion}
	if err := revision.Validate(); err != nil {
		return ClaimedAttempt{}, fmt.Errorf("validate claimed revision: %w", err)
	}
	attempt, err := attemptIdentity(*row.LogicalJobID, *row.AttemptGeneration, *row.RecoveryGeneration, *row.CurrentWorkerID)
	if err != nil {
		return ClaimedAttempt{}, err
	}
	if *row.AttemptsUsed < 1 {
		return ClaimedAttempt{}, fmt.Errorf("%w: invalid attempt number", ErrUnknownVocabulary)
	}
	budgetStartedAt, err := requiredTime("claim budget_started_at", row.BudgetStartedAt)
	if err != nil {
		return ClaimedAttempt{}, err
	}
	availableAt, err := requiredTime("claim available_at", row.AvailableAt)
	if err != nil {
		return ClaimedAttempt{}, err
	}
	startedAt, err := requiredTime("claim started_at", row.StartedAt)
	if err != nil {
		return ClaimedAttempt{}, err
	}
	leaseExpiresAt, err := requiredTime("claim lease_expires_at", row.LeaseExpiresAt)
	if err != nil {
		return ClaimedAttempt{}, err
	}
	return ClaimedAttempt{
		Identity: identity, Revision: revision, Payload: append([]byte(nil), row.Payload...),
		Attempt: attempt, AttemptNumber: uint32(*row.AttemptsUsed), BudgetStartedAt: budgetStartedAt,
		BudgetElapsed: max(time.Duration(0), startedAt.Sub(budgetStartedAt)),
		QueueDelay:    max(time.Duration(0), startedAt.Sub(availableAt)),
		StartedAt:     startedAt, LeaseExpiresAt: leaseExpiresAt,
	}, nil
}

func registryColumns(keys []jobs.Revision) ([]string, []string, []string, error) {
	kinds := make([]string, len(keys))
	argsVersions := make([]string, len(keys))
	policyVersions := make([]string, len(keys))
	for index, key := range keys {
		if err := key.Validate(); err != nil {
			return nil, nil, nil, fmt.Errorf("validate registry revision: %w", err)
		}
		if index > 0 && !revisionLess(keys[index-1], key) {
			return nil, nil, nil, fmt.Errorf("%w: registry keys must be strictly sorted", jobs.ErrInvalidDefinition)
		}
		kinds[index], argsVersions[index], policyVersions[index] = key.Kind, key.ArgsVersion, key.PolicyVersion
	}
	return kinds, argsVersions, policyVersions, nil
}

func revisionRows(kinds, argsVersions, policyVersions []string) ([]jobs.Revision, error) {
	if len(kinds) != len(argsVersions) || len(kinds) != len(policyVersions) {
		return nil, fmt.Errorf("%w: revision column length mismatch", ErrUnknownVocabulary)
	}
	revisions := make([]jobs.Revision, len(kinds))
	for index := range kinds {
		revisions[index] = jobs.Revision{Kind: kinds[index], ArgsVersion: argsVersions[index], PolicyVersion: policyVersions[index]}
		if err := revisions[index].Validate(); err != nil {
			return nil, fmt.Errorf("validate retained revision: %w", err)
		}
	}
	return revisions, nil
}

func unsupportedRevisions(revisions []jobs.Revision) error {
	values := make([]string, len(revisions))
	for index, revision := range revisions {
		values[index] = revision.Kind + "/" + revision.ArgsVersion + "/" + revision.PolicyVersion
	}
	return fmt.Errorf("%w: retained revisions %s", jobs.ErrUnsupportedRevision, strings.Join(values, ","))
}

func revisionLess(left, right jobs.Revision) bool {
	return cmp.Or(
		cmp.Compare(left.Kind, right.Kind),
		cmp.Compare(left.ArgsVersion, right.ArgsVersion),
		cmp.Compare(left.PolicyVersion, right.PolicyVersion),
	) < 0
}

func attemptIdentity(logicalJobID string, attemptGeneration, recoveryGeneration int64, workerID string) (AttemptIdentity, error) {
	if attemptGeneration < 1 || recoveryGeneration < 0 {
		return AttemptIdentity{}, fmt.Errorf("%w: invalid attempt generation", ErrUnknownVocabulary)
	}
	attempt := AttemptIdentity{
		LogicalJobID: jobs.LogicalJobID(logicalJobID), AttemptGeneration: uint64(attemptGeneration),
		RecoveryGeneration: uint64(recoveryGeneration), WorkerID: workerID,
	}
	return attempt, validateAttemptIdentity(attempt)
}

func validateAttemptIdentity(attempt AttemptIdentity) error {
	if err := validateStoreToken("logical_job_id", string(attempt.LogicalJobID)); err != nil {
		return err
	}
	if attempt.AttemptGeneration == 0 || attempt.AttemptGeneration > math.MaxInt64 || attempt.RecoveryGeneration > math.MaxInt64 {
		return fmt.Errorf("%w: invalid attempt generation", ErrConfig)
	}
	return validateStoreToken("worker_id", attempt.WorkerID)
}

func validateStoreToken(name, value string) error {
	if !utf8.ValidString(value) || len(value) < 1 || len(value) > 256 {
		return fmt.Errorf("%w: %s must be 1-256 UTF-8 bytes", ErrConfig, name)
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%w: %s contains a control character", ErrConfig, name)
	}
	return nil
}

func durationMicroseconds(name string, duration time.Duration) (int64, error) {
	if duration <= 0 {
		return 0, fmt.Errorf("%w: %s must be positive", ErrConfig, name)
	}
	microseconds := int64(duration / time.Microsecond)
	if duration%time.Microsecond != 0 {
		microseconds++
	}
	return microseconds, nil
}

func requiredTime(name string, value pgtype.Timestamptz) (time.Time, error) {
	if !value.Valid || value.InfinityModifier != pgtype.Finite {
		return time.Time{}, fmt.Errorf("%w: invalid %s", ErrUnknownVocabulary, name)
	}
	return value.Time, nil
}
