package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineObserveCachesFreshnessAndClosesOnCompatibilityLoss(t *testing.T) {
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
	fresh := engine.telemetry.snapshot.fresh
	engine.telemetry.mu.RUnlock()
	if fresh {
		t.Fatal("failed observation remained fresh")
	}
}
