package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineObserveCachesFreshnessAndClosesOnCompatibilityLoss(t *testing.T) {
	t.Parallel()
	now := time.Now()
	store := &engineStoreStub{observe: func(context.Context, []jobs.Revision) (Observation, error) {
		return Observation{ObservedAt: now, Compatible: false}, nil
	}}
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if facts := engine.Facts(); facts.ClaimAdmissionOpen || facts.Compatible || !facts.ObservationFresh {
		t.Fatalf("Facts() = %+v, want fresh incompatible closed", facts)
	}
}

func TestEngineObserveFailureMarksTelemetryStale(t *testing.T) {
	t.Parallel()
	store := &engineStoreStub{observe: func(context.Context, []jobs.Revision) (Observation, error) {
		return Observation{}, errors.New("database unavailable")
	}}
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.observe(context.Background()); err == nil {
		t.Fatal("observe() error = nil")
	}
	engine.telemetry.mu.RLock()
	freshUntil := engine.telemetry.snapshot.freshUntil
	engine.telemetry.mu.RUnlock()
	if !freshUntil.IsZero() {
		t.Fatal("failed observation remained fresh")
	}
}

func TestEngineObserveUsesLocalClockForCadenceAndFreshness(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		observedAt := []time.Time{
			time.Date(2200, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		}
		calls := 0
		store := &engineStoreStub{observe: func(context.Context, []jobs.Revision) (Observation, error) {
			result := Observation{ObservedAt: observedAt[calls], Compatible: true}
			calls++
			return result, nil
		}}
		cfg := engineConfig()
		cfg.ObservationInterval = 10 * time.Second
		cfg.ObservationMaxAge = 30 * time.Second
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), cfg)
		if err != nil {
			t.Fatal(err)
		}

		if err := engine.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		time.Sleep(cfg.ObservationInterval)
		synctest.Wait()
		if err := engine.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := engine.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("Observe calls = %d, want cadence independent of PostgreSQL clock skew", calls)
		}
		if facts := engine.Facts(); !facts.ObservationFresh {
			t.Fatalf("Facts() = %+v, want fresh local observation", facts)
		}

		time.Sleep(cfg.ObservationMaxAge)
		synctest.Wait()
		if facts := engine.Facts(); facts.ObservationFresh {
			t.Fatalf("Facts() = %+v, want stale observation after local max age", facts)
		}
	})
}
