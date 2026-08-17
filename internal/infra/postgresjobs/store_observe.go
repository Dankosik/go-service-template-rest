package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

type StateObservation struct {
	State             jobs.State
	Count             uint64
	OldestAvailableAt time.Time
}

type Observation struct {
	ObservedAt        time.Time
	Compatible        bool
	RequiredRevisions []jobs.Revision
	States            []StateObservation
}

func (s *Session) Observe(ctx context.Context, registryKeys []jobs.Revision) (observation Observation, err error) {
	kinds, argsVersions, policyVersions, err := registryColumns(registryKeys)
	if err != nil {
		return Observation{}, err
	}
	err = s.withOperation(ctx, "observe", pgx.ReadOnly, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		row, queryErr := queries.ObservePostgresJobs(operationCtx, sqlcgen.ObservePostgresJobsParams{
			Kinds: kinds, ArgsVersions: argsVersions, PolicyVersions: policyVersions,
		})
		if queryErr != nil {
			return fmt.Errorf("observe postgres jobs: %w", queryErr)
		}
		if row.Compatible == nil {
			return fmt.Errorf("%w: missing observation compatibility", ErrUnknownVocabulary)
		}
		observedAt, mapErr := requiredTime("observation observed_at", row.ObservedAt)
		if mapErr != nil {
			return mapErr
		}
		revisions, mapErr := revisionRows(row.RequiredKinds, row.RequiredArgsVersions, row.RequiredPolicyVersions)
		if mapErr != nil {
			return mapErr
		}
		if len(row.States) != len(row.StateCounts) || len(row.States) != len(row.OldestAvailableAt) {
			return fmt.Errorf("%w: observation column length mismatch", ErrUnknownVocabulary)
		}
		states := make([]StateObservation, len(row.States))
		for index, stateValue := range row.States {
			if row.StateCounts[index] < 0 {
				return fmt.Errorf("%w: negative state count", ErrUnknownVocabulary)
			}
			state, stateErr := stateFromDatabase(stateValue)
			if stateErr != nil {
				return stateErr
			}
			oldest, stateErr := requiredTime("observation oldest_available_at", row.OldestAvailableAt[index])
			if stateErr != nil {
				return stateErr
			}
			states[index] = StateObservation{State: state, Count: uint64(row.StateCounts[index]), OldestAvailableAt: oldest}
		}
		observation = Observation{
			ObservedAt: observedAt, Compatible: *row.Compatible,
			RequiredRevisions: revisions, States: states,
		}
		return nil
	})
	return observation, err
}
