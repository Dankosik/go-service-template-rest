package postgreswebhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type Observation struct {
	Scheduled       int64
	InFlight        int64
	Terminal        int64
	Disabled        int64
	LeasedSlots     int64
	TotalSlots      int64
	ClockHighWater  time.Time
	ClockRegression bool
}

//nolint:gocognit // One observation must validate the complete database/secret snapshot atomically.
func (s *Store) ObserveReadiness(ctx context.Context, manifest *SecretManifest) (Observation, error) {
	if !s.valid() || manifest == nil {
		return Observation{}, fmt.Errorf("%w: store and manifest are required", ErrConfig)
	}
	var observation Observation
	err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		clock, err := queries.ObserveWebhookClock(ctx)
		if err != nil {
			return fmt.Errorf("observe webhook clock: %w", err)
		}
		capacity, err := queries.ReadWebhookCapacity(ctx)
		if err != nil {
			return fmt.Errorf("observe webhook capacity: %w", err)
		}
		bindings, err := queries.ListWebhookSecretBindings(ctx)
		if err != nil {
			return fmt.Errorf("observe webhook secret bindings: %w", err)
		}
		counts, err := queries.ObservePostgresWebhooks(ctx)
		if err != nil {
			return fmt.Errorf("observe postgres webhooks: %w", err)
		}
		observation = Observation{Scheduled: counts.Scheduled, InFlight: counts.InFlight, Terminal: counts.Terminal, Disabled: counts.Disabled, LeasedSlots: counts.LeasedSlots, TotalSlots: counts.TotalSlots, ClockHighWater: clock.HighWater.Time.UTC(), ClockRegression: clock.Regression}
		if capacity.RevisionCount != 1 || capacity.CapacityRevision != s.options.CapacityRevision || int(capacity.SlotCount) != s.options.GlobalConcurrency {
			return fmt.Errorf("%w: capacity revision/count conflict", ErrConfig)
		}
		for _, binding := range bindings {
			if binding.RequiredSecretRevision > manifest.Revision() {
				return fmt.Errorf("%w: secret manifest revision is stale", ErrConfig)
			}
			if _, err := manifest.Resolve(binding.OwnerScope, binding.DestinationID, binding.ActiveKeyReference); err != nil {
				return fmt.Errorf("%w: active secret binding is missing", ErrConfig)
			}
			if binding.PredecessorKeyReference != nil && binding.PredecessorValidUntil.Valid && clock.HighWater.Time.Before(binding.PredecessorValidUntil.Time) {
				if _, err := manifest.Resolve(binding.OwnerScope, binding.DestinationID, *binding.PredecessorKeyReference); err != nil {
					return fmt.Errorf("%w: predecessor secret binding is missing", ErrConfig)
				}
			}
		}
		return nil
	})
	if err != nil {
		return observation, err
	}
	if observation.ClockRegression {
		return observation, ErrClockRegression
	}
	return observation, nil
}

type readinessState struct {
	mu              sync.RWMutex
	admission       bool
	maintenanceLive bool
	lastObservation time.Time
	interval        time.Duration
}

func (state *readinessState) update() {
	state.mu.Lock()
	state.admission = true
	state.maintenanceLive = true
	state.lastObservation = time.Now()
	state.mu.Unlock()
}

func (state *readinessState) close() {
	state.mu.Lock()
	state.admission = false
	state.mu.Unlock()
}

func (state *readinessState) ready() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.admission && state.maintenanceLive && !state.lastObservation.IsZero() && time.Since(state.lastObservation) <= 2*state.interval
}
