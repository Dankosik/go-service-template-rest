package postgresjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

type engineStore interface {
	Claim(context.Context, ClaimOptions) (ClaimResult, error)
	ResolveClaims(context.Context, []AttemptIdentity) ([]ClaimResolution, error)
	Finalize(context.Context, FinalizeInput) (PersistedTransition, error)
	Renew(context.Context, []AttemptIdentity, time.Duration) ([]Renewal, error)
	RescueCandidates(context.Context, int) ([]RescueCandidate, error)
	Rescue(context.Context, RescueInput) (PersistedTransition, error)
	Observe(context.Context, []jobs.Revision) (Observation, error)
}

type EngineConfig struct {
	WorkerID            string
	MaxConcurrency      int
	LeaseDuration       time.Duration
	ObservationInterval time.Duration
	DrainTimeout        time.Duration
}

type EngineFacts struct {
	ClaimAdmissionOpen bool
	Compatible         bool
	InFlight           int
	Capacity           int
	ObservationFresh   bool
}

// Engine owns the single coordinator's shared state. Stage-specific behavior
// stays with its engine_* file so later stages can extend this cycle directly.
type Engine struct {
	store    engineStore
	registry *jobs.Registry
	config   EngineConfig

	mu              sync.Mutex
	cycleMu         sync.Mutex
	attempts        sync.WaitGroup
	admission       bool
	compatible      bool
	inflight        map[AttemptIdentity]context.CancelFunc
	lastLease       time.Time
	lastObservation time.Time
	telemetry       *Telemetry
}

func NewEngine(session *Session, registry *jobs.Registry, config EngineConfig) (*Engine, error) {
	if session == nil || !session.valid() {
		return nil, fmt.Errorf("%w: usable Session is required", ErrConfig)
	}
	return newEngine(session, registry, config)
}

func newEngine(store engineStore, registry *jobs.Registry, config EngineConfig) (*Engine, error) {
	if store == nil || registry == nil {
		return nil, fmt.Errorf("%w: engine store and registry are required", ErrConfig)
	}
	if err := validateStoreToken("worker_id", config.WorkerID); err != nil {
		return nil, err
	}
	if config.MaxConcurrency < 1 || config.LeaseDuration <= 0 || config.ObservationInterval <= 0 || config.DrainTimeout <= 0 {
		return nil, fmt.Errorf("%w: positive concurrency, lease duration, observation interval, and drain timeout are required", ErrConfig)
	}
	telemetry, err := NewTelemetry(nil)
	if err != nil {
		return nil, err
	}
	return &Engine{
		store: store, registry: registry, config: config,
		admission: true, compatible: true, inflight: make(map[AttemptIdentity]context.CancelFunc), telemetry: telemetry,
	}, nil
}

// Run performs one serial coordinator cycle.
func (e *Engine) Run(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("%w: engine is required", ErrConfig)
	}
	e.cycleMu.Lock()
	defer e.cycleMu.Unlock()
	if err := e.renew(ctx); err != nil {
		return e.fail(err)
	}
	if err := e.rescue(ctx); err != nil {
		return e.fail(err)
	}
	if err := e.renew(ctx); err != nil {
		return e.fail(err)
	}
	if err := e.claim(ctx); err != nil {
		return e.fail(err)
	}
	if err := e.renew(ctx); err != nil {
		return e.fail(err)
	}
	if err := e.observe(ctx); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Engine) Facts() EngineFacts {
	if e == nil {
		return EngineFacts{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return EngineFacts{
		ClaimAdmissionOpen: e.admission,
		Compatible:         e.compatible,
		InFlight:           len(e.inflight),
		Capacity:           e.config.MaxConcurrency,
		ObservationFresh:   !e.lastObservation.IsZero(),
	}
}

func (e *Engine) freeCapacityLocked() int { return e.config.MaxConcurrency - len(e.inflight) }

func (e *Engine) closeAdmissionLocked() { e.admission = false }

func (e *Engine) fail(err error) error {
	e.mu.Lock()
	e.closeAdmissionLocked()
	e.lastObservation = time.Time{}
	for _, cancel := range e.inflight {
		cancel()
	}
	facts := EngineFacts{ClaimAdmissionOpen: e.admission, Compatible: e.compatible, InFlight: len(e.inflight), Capacity: e.config.MaxConcurrency}
	e.mu.Unlock()
	e.telemetry.MarkStale()
	e.telemetry.UpdateFacts(facts)
	return err
}
