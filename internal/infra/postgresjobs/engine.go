package postgresjobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

type engineStore interface {
	Claim(ctx context.Context, options ClaimOptions) (ClaimResult, error)
	ResolveClaims(ctx context.Context, attempts []AttemptIdentity) ([]ClaimResolution, error)
	Finalize(ctx context.Context, input FinalizeInput) (PersistedTransition, error)
	Renew(ctx context.Context, attempts []AttemptIdentity, leaseDuration time.Duration) ([]Renewal, error)
	RescueCandidates(ctx context.Context, options RescueCandidateOptions) ([]RescueCandidate, error)
	Rescue(ctx context.Context, input RescueInput) (PersistedTransition, error)
	Observe(ctx context.Context, revisions []jobs.Revision) (Observation, error)
}

type EngineConfig struct {
	WorkerID            string
	MaxConcurrency      int
	LeaseDuration       time.Duration
	ObservationInterval time.Duration
	ObservationMaxAge   time.Duration
	DrainTimeout        time.Duration
}

type EngineFacts struct {
	ClaimAdmissionOpen bool
	Compatible         bool
	InFlight           int
	Capacity           int
	ObservationFresh   bool
}

type inflightAttempt struct {
	cancel         context.CancelFunc
	renewAt        time.Time
	cancelObserved bool
}

// Engine owns the single coordinator's shared state. Stage-specific behavior
// stays with its engine_* file so later stages can extend this cycle directly.
type Engine struct {
	store    engineStore
	registry *jobs.Registry
	config   EngineConfig

	mu              sync.Mutex
	cycle           chan struct{}
	cycleCancel     context.CancelFunc
	attempts        sync.WaitGroup
	admission       bool
	draining        bool
	compatible      bool
	inflight        map[AttemptIdentity]inflightAttempt
	lastObservation time.Time
	terminal        chan error
	terminalErr     error
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
	if len(registry.Keys()) == 0 {
		return nil, fmt.Errorf("%w: at least one jobs definition is required", ErrConfig)
	}
	if config.MaxConcurrency < 1 || config.LeaseDuration <= 0 || config.ObservationInterval <= 0 || config.ObservationMaxAge < config.ObservationInterval || config.DrainTimeout <= 0 {
		return nil, fmt.Errorf("%w: positive concurrency, lease duration, observation interval, observation max age, and drain timeout are required", ErrConfig)
	}
	telemetry, err := NewTelemetry(nil)
	if err != nil {
		return nil, err
	}
	engine := &Engine{
		store: store, registry: registry, config: config,
		admission: true, compatible: true, inflight: make(map[AttemptIdentity]inflightAttempt),
		cycle: make(chan struct{}, 1), terminal: make(chan error, 1), telemetry: telemetry,
	}
	engine.cycle <- struct{}{}
	return engine, nil
}

// Run performs one serial coordinator cycle.
func (e *Engine) Run(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("%w: engine is required", ErrConfig)
	}
	if !e.lockCycle(ctx) {
		return fmt.Errorf("run jobs coordinator: %w", ctx.Err())
	}
	defer e.unlockCycle()
	cycleCtx, cancelCycle := context.WithCancel(ctx)
	defer cancelCycle()
	e.mu.Lock()
	if e.draining {
		e.mu.Unlock()
		return nil
	}
	e.cycleCancel = cancelCycle
	defer func() {
		e.mu.Lock()
		e.cycleCancel = nil
		e.mu.Unlock()
	}()
	terminalErr := e.terminalErr
	e.mu.Unlock()
	if terminalErr != nil {
		return terminalErr
	}
	if err := e.renew(cycleCtx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	if err := e.rescue(cycleCtx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	if err := e.renew(cycleCtx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	if err := e.claim(cycleCtx, ctx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	if err := e.renew(cycleCtx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	if err := e.observe(cycleCtx); err != nil {
		return e.failCycle(cycleCtx, err)
	}
	return nil
}

func (e *Engine) lockCycle(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-e.cycle:
		return true
	}
}

func (e *Engine) unlockCycle() { e.cycle <- struct{}{} }

func (e *Engine) Terminal() <-chan error {
	if e == nil {
		return nil
	}
	return e.terminal
}

func (e *Engine) Close() error {
	if e == nil || e.telemetry == nil {
		return nil
	}
	return e.telemetry.Unregister()
}

func (e *Engine) Facts() EngineFacts {
	if e == nil {
		return EngineFacts{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.factsLocked(time.Now())
}

func (e *Engine) factsLocked(now time.Time) EngineFacts {
	return EngineFacts{
		ClaimAdmissionOpen: e.admission,
		Compatible:         e.compatible,
		InFlight:           len(e.inflight),
		Capacity:           e.config.MaxConcurrency,
		ObservationFresh:   !e.lastObservation.IsZero() && now.Before(e.lastObservation.Add(e.config.ObservationMaxAge)),
	}
}

func (e *Engine) freeCapacityLocked() int { return e.config.MaxConcurrency - len(e.inflight) }

func (e *Engine) closeAdmissionLocked() { e.admission = false }

func (e *Engine) fail(ctx context.Context, err error) error {
	e.mu.Lock()
	e.closeAdmissionLocked()
	e.lastObservation = time.Time{}
	for _, attempt := range e.inflight {
		attempt.cancel()
	}
	firstFailure := e.terminalErr == nil
	if firstFailure {
		e.terminalErr = err
		e.terminal <- err
	}
	facts := e.factsLocked(time.Now())
	e.mu.Unlock()
	e.telemetry.MarkStale()
	e.telemetry.UpdateFacts(facts)
	if firstFailure {
		e.telemetry.RecordTerminalFailure(context.WithoutCancel(ctx))
	}
	return err
}

func (e *Engine) failCycle(ctx context.Context, err error) error {
	e.mu.Lock()
	draining := e.draining
	e.mu.Unlock()
	if draining && errors.Is(err, context.Canceled) {
		return nil
	}
	return e.fail(ctx, err)
}
