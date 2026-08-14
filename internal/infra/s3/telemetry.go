package s3

import (
	"context"
	"sync"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/objectstorage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const telemetryScope = "service.object_storage.s3"

type telemetry struct {
	tracer trace.Tracer

	operations metric.Int64Counter
	duration   metric.Float64Histogram
	active     metric.Int64UpDownCounter
	admitted   metric.Int64Counter
	rejected   metric.Int64Counter
	bytes      metric.Int64Counter
	integrity  metric.Int64Counter
	issuances  metric.Int64Counter
}

func newTelemetry() (*telemetry, error) {
	return newTelemetryWithMeter(otel.GetMeterProvider().Meter(telemetryScope), otel.GetTracerProvider().Tracer(telemetryScope))
}

func newTelemetryWithMeter(meter metric.Meter, tracer trace.Tracer) (*telemetry, error) {
	set := infratelemetry.NewInstrumentSet(meter)
	t := &telemetry{tracer: tracer}
	set.Int64Counter(&t.operations, "object_storage.operations")
	set.Float64Histogram(&t.duration, "object_storage.operation.duration", metric.WithUnit("s"))
	set.Int64UpDownCounter(&t.active, "object_storage.active")
	set.Int64Counter(&t.admitted, "object_storage.admitted")
	set.Int64Counter(&t.rejected, "object_storage.rejected")
	set.Int64Counter(&t.bytes, "object_storage.completed.bytes", metric.WithUnit("By"))
	set.Int64Counter(&t.integrity, "object_storage.integrity.failures")
	set.Int64Counter(&t.issuances, "object_storage.presign.issuances")
	if err := set.Err(); err != nil {
		return nil, err
	}
	return t, nil
}

type operationName string

const (
	telemetryOperationUpload   operationName = "upload"
	telemetryOperationDownload operationName = "download"
	telemetryOperationMetadata operationName = "metadata"
	telemetryOperationDelete   operationName = "delete"
	telemetryOperationPresign  operationName = "presign"
)

type operationCall struct {
	telemetry *telemetry
	name      operationName
	started   time.Time
	span      trace.Span
	path      string
	cleanup   string

	once sync.Once
}

//nolint:contextcheck,spancheck // The caller context parents the span and operationCall.finish owns its single terminal End.
func (t *telemetry) begin(ctx context.Context, name operationName) *operationCall {
	if ctx == nil {
		ctx = context.Background()
	}
	_, span := t.tracer.Start(ctx, "object_storage."+string(name))
	return &operationCall{telemetry: t, name: name, started: time.Now(), span: span, path: "none", cleanup: "none"}
}

func (c *operationCall) admitted() {
	c.telemetry.admitted.Add(context.Background(), 1)
	c.telemetry.active.Add(context.Background(), 1)
}

func (c *operationCall) rejected() {
	c.telemetry.rejected.Add(context.Background(), 1)
}

func (c *operationCall) released() {
	c.telemetry.active.Add(context.Background(), -1)
}

func (c *operationCall) finish(err error, bytes int64) {
	c.once.Do(func() {
		result := "success"
		phase := "complete"
		if err != nil {
			result = boundedResult(objectstorage.Kind(err))
			phase = failurePhase(result)
		}
		attrs := metric.WithAttributes(
			attribute.String("object_storage.operation", string(c.name)),
			attribute.String("object_storage.result", result),
			attribute.String("object_storage.transfer_path", boundedPath(c.path)),
			attribute.String("object_storage.cleanup", boundedCleanup(c.cleanup)),
		)
		c.telemetry.operations.Add(context.Background(), 1, attrs)
		c.telemetry.duration.Record(context.Background(), time.Since(c.started).Seconds(), attrs)
		if bytes > 0 && err == nil && (c.name == telemetryOperationUpload || c.name == telemetryOperationDownload) {
			c.telemetry.bytes.Add(context.Background(), bytes, metric.WithAttributes(attribute.String("object_storage.operation", string(c.name))))
		}
		if result == string(objectstorage.KindIntegrityFailed) {
			c.telemetry.integrity.Add(context.Background(), 1, metric.WithAttributes(attribute.String("object_storage.operation", string(c.name))))
		}
		if c.name == telemetryOperationPresign && err == nil {
			c.telemetry.issuances.Add(context.Background(), 1)
		}
		c.span.SetAttributes(
			attribute.String("object_storage.operation", string(c.name)),
			attribute.String("object_storage.result", result),
			attribute.String("object_storage.phase", phase),
			attribute.String("object_storage.transfer_path", boundedPath(c.path)),
			attribute.String("object_storage.cleanup", boundedCleanup(c.cleanup)),
		)
		c.span.End()
	})
}

func boundedResult(kind objectstorage.ErrorKind) string {
	switch kind {
	case objectstorage.KindInvalid, objectstorage.KindTooLarge, objectstorage.KindBusy, objectstorage.KindNotFound,
		objectstorage.KindPreconditionFailed, objectstorage.KindDenied, objectstorage.KindTemporary,
		objectstorage.KindIntegrityFailed, objectstorage.KindCancelled, objectstorage.KindDeadlineExceeded,
		objectstorage.KindOutcomeUnknown, objectstorage.KindInternal:
		return string(kind)
	default:
		return "unknown"
	}
}

func boundedPath(path string) string {
	if path == "single" || path == "multipart" || path == "none" {
		return path
	}
	return "unknown"
}

func boundedCleanup(cleanup string) string {
	if cleanup == string(objectstorage.CleanupNone) || cleanup == string(objectstorage.CleanupComplete) || cleanup == string(objectstorage.CleanupPending) {
		return cleanup
	}
	return "unknown"
}

func failurePhase(result string) string {
	switch result {
	case "invalid":
		return "validation"
	case "busy":
		return "admission"
	case "cancelled", "deadline_exceeded":
		return "context"
	case "integrity_failed":
		return "integrity"
	default:
		return "response"
	}
}
