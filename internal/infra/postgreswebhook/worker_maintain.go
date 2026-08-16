package postgreswebhook

import "context"

func (worker *Worker) maintain(ctx context.Context) error {
	reconciled, err := worker.store.ReconcileExpired(ctx, worker.config.MaintenanceBatch)
	if err != nil {
		return err
	}
	worker.telemetry.RecordMaintenance(ctx, "reconcile", int64(reconciled))
	finalized, err := worker.store.FinalizeExpiredCycles(ctx, worker.config.MaintenanceBatch)
	if err != nil {
		return err
	}
	worker.telemetry.RecordMaintenance(ctx, "deadline", finalized)
	cleaned, err := worker.store.CleanupRetention(ctx, worker.config.MaintenanceBatch)
	worker.telemetry.RecordMaintenance(ctx, "cleanup", cleaned)
	return err
}
