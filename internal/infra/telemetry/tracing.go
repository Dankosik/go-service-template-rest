package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type TracingConfig struct {
	ServiceName      string
	DeploymentEnv    string
	TracesSampler    string
	TracesSamplerArg float64
}

func SetupTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "service"
	}
	deploymentEnv := strings.TrimSpace(cfg.DeploymentEnv)
	if deploymentEnv == "" {
		deploymentEnv = "unknown"
	}

	sampler, err := buildTraceSampler(cfg.TracesSampler, cfg.TracesSamplerArg)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment.name", deploymentEnv),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}

	if isOTLPTraceExporterConfigured() {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

func buildTraceSampler(name string, arg float64) (sdktrace.Sampler, error) {
	samplerName := strings.ToLower(strings.TrimSpace(name))
	if samplerName == "" {
		samplerName = "parentbased_traceidratio"
	}

	ratio := clampRatio(arg)

	switch samplerName {
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio), nil
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), nil
	default:
		return nil, fmt.Errorf("unsupported trace sampler %q", name)
	}
}

func clampRatio(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func isOTLPTraceExporterConfigured() bool {
	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) != "" ||
		strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")) != ""
}
