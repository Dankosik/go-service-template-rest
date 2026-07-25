package bootstrap

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func recordStartupRejection(bootstrapSpan trace.Span, errorType, failedStage string, err error) {
	if err != nil {
		bootstrapSpan.RecordError(err)
	}
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", errorType),
		attribute.String("failed.stage", failedStage),
	)
}
