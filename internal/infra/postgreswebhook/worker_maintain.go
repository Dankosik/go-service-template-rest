package postgreswebhook

import "context"

func (worker *Worker) maintain(ctx context.Context) error {
	if _, err := worker.store.ReconcileExpired(ctx, worker.config.MaintenanceBatch); err != nil {
		return err
	}
	_, err := worker.store.CleanupRetention(ctx, worker.config.MaintenanceBatch)
	return err
}
