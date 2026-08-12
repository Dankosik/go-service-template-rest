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
}

type EngineConfig struct {
	WorkerID       string
	MaxConcurrency int
	LeaseDuration  time.Duration
}

type EngineFacts struct {
	ClaimAdmissionOpen bool
	Compatible         bool
	InFlight           int
}

// Engine owns the single coordinator's shared state. Stage-specific behavior
// stays with its engine_* file so later stages can extend this cycle directly.
type Engine struct {
	store    engineStore
	registry *jobs.Registry
	config   EngineConfig

	mu         sync.Mutex
	admission  bool
	compatible bool
	inflight   map[AttemptIdentity]context.CancelFunc
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
	if config.MaxConcurrency < 1 || config.LeaseDuration <= 0 {
		return nil, fmt.Errorf("%w: positive concurrency and lease duration are required", ErrConfig)
	}
	return &Engine{
		store: store, registry: registry, config: config,
		admission: true, compatible: true, inflight: make(map[AttemptIdentity]context.CancelFunc),
	}, nil
}

// Run performs one serial coordinator cycle. T7 extends this cycle with the
// due renewal, rescue, and observation stages.
func (e *Engine) Run(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("%w: engine is required", ErrConfig)
	}
	return e.claim(ctx)
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
	}
}

func (e *Engine) freeCapacityLocked() int { return e.config.MaxConcurrency - len(e.inflight) }

func (e *Engine) closeAdmissionLocked() { e.admission = false }
