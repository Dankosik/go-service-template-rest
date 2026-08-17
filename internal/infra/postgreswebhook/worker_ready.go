package postgreswebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type Observation struct {
	Ready             int64
	Scheduled         int64
	InFlight          int64
	Terminal          int64
	Suspended         int64
	Quarantined       int64
	Disabled          int64
	OldestDueAge      time.Duration
	HTTPAccepted      int64
	HTTPRejected      int64
	LocallyDenied     int64
	OutcomeUnknown    int64
	AttemptsExhausted int64
	RedriveExhausted  int64
	LeasedSlots       int64
	TotalSlots        int64
	RetentionBackfill int64
	PrivacyPending    int64
	ClockHighWater    time.Time
	ClockRegression   bool
}

//nolint:gocognit,cyclop // One observation must validate the complete database/secret snapshot atomically.
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
		observation = Observation{
			Ready: counts.Ready, Scheduled: counts.Scheduled, InFlight: counts.InFlight,
			Terminal: counts.Terminal, Suspended: counts.Suspended, Quarantined: counts.Quarantined,
			Disabled: counts.Disabled, OldestDueAge: time.Duration(counts.OldestDueAgeSeconds) * time.Second,
			HTTPAccepted: counts.HttpAccepted, HTTPRejected: counts.HttpRejected, LocallyDenied: counts.LocallyDenied,
			OutcomeUnknown: counts.OutcomeUnknown, AttemptsExhausted: counts.AttemptsExhausted,
			RedriveExhausted: counts.RedriveExhausted, LeasedSlots: counts.LeasedSlots,
			TotalSlots: counts.TotalSlots, RetentionBackfill: counts.RetentionBackfillPending, PrivacyPending: counts.PrivacyPending,
			ClockHighWater: clock.HighWater.Time.UTC(), ClockRegression: clock.Regression,
		}
		if observation.RetentionBackfill != 0 {
			return fmt.Errorf("%w: webhook retention backfill is incomplete", ErrConfig)
		}
		if capacity.RevisionCount != 1 || capacity.CapacityRevision != s.options.CapacityRevision || int(capacity.SlotCount) != s.options.GlobalConcurrency {
			return fmt.Errorf("%w: capacity revision/count conflict", ErrConfig)
		}
		for _, binding := range bindings {
			var policy DeliveryPolicy
			if err := json.Unmarshal(binding.Policy, &policy); err != nil || policy.validate() != nil || !s.admits(policy) {
				return fmt.Errorf("%w: retained destination policy is incompatible", ErrConfig)
			}
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

func (state *readinessState) observed() {
	state.mu.Lock()
	state.admission = true
	state.lastObservation = time.Now()
	state.mu.Unlock()
}

func (state *readinessState) maintained() {
	state.mu.Lock()
	state.maintenanceLive = true
	state.mu.Unlock()
}

func (state *readinessState) close() {
	state.mu.Lock()
	state.admission = false
	state.maintenanceLive = false
	state.mu.Unlock()
}

func (state *readinessState) ready() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.admission && state.maintenanceLive && !state.lastObservation.IsZero() && time.Since(state.lastObservation) <= 2*state.interval
}
