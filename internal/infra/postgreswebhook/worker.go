package postgreswebhook

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type WorkerConfig struct {
	WorkerID              string
	ClaimScanPage         int
	PollInterval          time.Duration
	ObservationInterval   time.Duration
	AttemptTimeout        time.Duration
	StoreOperationTimeout time.Duration
	DrainTimeout          time.Duration
	MaintenanceInterval   time.Duration
	MaintenanceBatch      int
}

type WorkerResult struct {
	Err           error
	CleanupUnsafe bool
}

type Worker struct {
	store     *Store
	manifest  *SecretManifest
	resolver  *net.Resolver
	config    WorkerConfig
	ready     readinessState
	attempts  sync.WaitGroup
	slots     chan struct{}
	telemetry *Telemetry
}

func NewWorker(store *Store, manifest *SecretManifest, config WorkerConfig) (*Worker, error) {
	if store == nil || !store.valid() || manifest == nil || manifest.Revision() != store.options.ManifestRevision ||
		validateToken("worker_id", config.WorkerID) != nil || config.ClaimScanPage < 1 || config.ClaimScanPage > MaxClaimScanPage ||
		config.PollInterval <= 0 || config.ObservationInterval <= 0 || config.AttemptTimeout <= 0 ||
		config.StoreOperationTimeout <= 0 || config.DrainTimeout <= config.AttemptTimeout ||
		config.AttemptTimeout != store.options.AttemptTimeout || config.StoreOperationTimeout != store.options.OperationTimeout || config.DrainTimeout != store.options.DrainTimeout ||
		config.MaintenanceInterval <= 0 || config.MaintenanceBatch < 1 || config.MaintenanceBatch > 1000 {
		return nil, fmt.Errorf("%w: worker bounds are invalid", ErrConfig)
	}
	telemetry, err := NewTelemetry(nil)
	if err != nil {
		return nil, err
	}
	worker := &Worker{store: store, manifest: manifest, resolver: &net.Resolver{PreferGo: true}, config: config, slots: make(chan struct{}, store.options.GlobalConcurrency), telemetry: telemetry}
	worker.ready.interval = config.ObservationInterval
	return worker, nil
}

func (worker *Worker) Ready() bool { return worker != nil && worker.ready.ready() }

func (worker *Worker) CloseTelemetry() error {
	if worker == nil {
		return nil
	}
	return worker.telemetry.Unregister()
}
