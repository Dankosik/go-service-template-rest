package postgreswebhook

import (
	"context"
	"time"
)

type maintenanceResult struct {
	reconciled        int64
	retired           int64
	cleaned           int64
	reconcileDuration time.Duration
	retireDuration    time.Duration
	cleanupDuration   time.Duration
}

func (worker *Worker) maintain(ctx context.Context) (maintenanceResult, error) {
	result := maintenanceResult{}
	started := time.Now()
	reconciled, err := worker.store.ReconcileExpired(ctx, worker.config.MaintenanceBatch)
	result.reconcileDuration = time.Since(started)
	if err != nil {
		return result, err
	}
	result.reconciled = int64(reconciled)
	started = time.Now()
	retired, err := worker.store.ResumeNamespaceRetirements(ctx, worker.config.MaintenanceBatch)
	result.retireDuration = time.Since(started)
	if err != nil {
		return result, err
	}
	result.retired = int64(retired)
	started = time.Now()
	cleaned, err := worker.store.CleanupRetention(ctx, worker.config.MaintenanceBatch)
	result.cleanupDuration = time.Since(started)
	result.cleaned = cleaned
	return result, err
}
