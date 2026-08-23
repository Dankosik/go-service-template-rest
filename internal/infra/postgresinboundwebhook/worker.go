// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

const (
	quarantineReasonInvalidJSON = "invalid_json"
	quarantineReasonRejected    = "schema_rejected"
	terminalSnooze              = time.Second
)

type storedReceipt struct {
	ReceiptID  string
	EndpointID string
	DeliveryID string
	SignedAt   time.Time
	ReceivedAt time.Time
	Payload    []byte
	Outcome    string
}

// Worker processes one inbound receipt job.
type Worker struct {
	river.WorkerDefaults[receiptJobArgs]

	store    receiptStore
	registry *inboundwebhook.Registry
	telem    telemetry
}

// NewWorker builds the River worker.
func newWorker(store receiptStore, registry *inboundwebhook.Registry, telem telemetry) (*Worker, error) {
	if store == nil || registry == nil {
		return nil, errors.New("inbound webhook store and registry are required")
	}
	return &Worker{store: store, registry: registry, telem: telem}, nil
}

// AddWorker registers the inbound worker on workers.
func AddWorker(workers *river.Workers, pool *pgxpool.Pool, registry *inboundwebhook.Registry, meter metric.MeterProvider, log *slog.Logger) error {
	if workers == nil || pool == nil {
		return errors.New("inbound webhook workers and postgres pool are required")
	}
	store, err := newPostgresStore(pool)
	if err != nil {
		return err
	}
	worker, err := newWorker(store, registry, newTelemetry(meter, log))
	if err != nil {
		return err
	}
	if err := river.AddWorkerSafely(workers, worker); err != nil {
		return fmt.Errorf("register inbound webhook worker: %w", err)
	}
	return nil
}

func (*Worker) Timeout(*river.Job[receiptJobArgs]) time.Duration { return 30 * time.Second }

//nolint:cyclop // One River lifecycle owner keeps retry and terminal state in one linear path.
func (w *Worker) Work(ctx context.Context, job *river.Job[receiptJobArgs]) (err error) {
	if w == nil || w.store == nil || w.registry == nil || job == nil {
		return errStorageUnavailable
	}
	receiptID := job.Args.ReceiptID
	if receiptID == "" {
		return errStorageUnavailable
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			w.telem.logFailure(ctx, receiptID, logClassPanicRecovered)
			w.telem.recordProcessing(ctx, "retrying")
			err = errPanicRecovered
		}
	}()

	receipt, err := w.store.loadByID(ctx, receiptID)
	if err != nil {
		w.telem.logFailure(ctx, receiptID, logClassStorageRetryable)
		w.telem.recordProcessing(ctx, "retrying")
		if job.Attempt >= job.MaxAttempts {
			return river.JobSnooze(terminalSnooze)
		}
		return errStorageUnavailable
	}
	if receipt.Outcome != "pending" {
		return nil
	}
	if !w.registry.HasBinding(receipt.EndpointID) {
		w.telem.logFailure(ctx, receiptID, logClassBindingUnavailable)
		w.telem.recordProcessing(ctx, "retrying")
		// ponytail: reuse the existing snooze; isolate a queue if binding drift becomes load.
		return river.JobSnooze(terminalSnooze)
	}
	if job.Attempt >= job.MaxAttempts {
		return w.finalize(ctx, receiptID)
	}

	delivery := inboundwebhook.VerifiedDelivery{
		EndpointID: receipt.EndpointID,
		DeliveryID: receipt.DeliveryID,
		SignedAt:   receipt.SignedAt,
		Body:       json.RawMessage(receipt.Payload),
		ReceivedAt: receipt.ReceivedAt,
	}

	dispatchErr := w.registry.Dispatch(ctx, delivery)
	switch {
	case dispatchErr == nil:
		updated, markErr := w.store.MarkHandled(ctx, receiptID)
		if markErr != nil {
			w.telem.logFailure(ctx, receiptID, logClassStorageRetryable)
			w.telem.recordProcessing(ctx, "retrying")
			return errStorageUnavailable
		}
		if updated {
			w.telem.recordProcessing(ctx, "handled")
		}
		return nil
	case inboundwebhook.IsDecodeError(dispatchErr) && errors.Is(dispatchErr, inboundwebhook.ErrDecodeRejected):
		reason := quarantineReasonRejected
		if !json.Valid(receipt.Payload) {
			reason = quarantineReasonInvalidJSON
		}
		updated, markErr := w.store.MarkQuarantined(ctx, receiptID, reason)
		if markErr != nil {
			w.telem.logFailure(ctx, receiptID, logClassStorageRetryable)
			w.telem.recordProcessing(ctx, "retrying")
			return errStorageUnavailable
		}
		if updated {
			w.telem.recordProcessing(ctx, "quarantined")
		}
		return nil
	case inboundwebhook.IsDecodeError(dispatchErr):
		w.telem.logFailure(ctx, receiptID, logClassDecoderInternal)
		w.telem.recordProcessing(ctx, "retrying")
		return errDecoderFailed
	default:
		w.telem.logFailure(ctx, receiptID, logClassHandlerRetryable)
		w.telem.recordProcessing(ctx, "retrying")
		return errHandlerFailed
	}
}

func (w *Worker) finalize(ctx context.Context, receiptID string) error {
	updated, err := w.store.MarkFailed(ctx, receiptID)
	if err != nil {
		w.telem.logFailure(ctx, receiptID, logClassTerminalizationRetryable)
		return river.JobSnooze(terminalSnooze)
	}
	if updated {
		w.telem.recordProcessing(ctx, "failed")
	}
	return nil
}

// profile:inbound-webhooks-standard:end
