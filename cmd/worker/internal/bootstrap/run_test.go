package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
)

func TestMessagingCompositionRejectsEmptyHandlerBeforeConfig(t *testing.T) {
	if err := run(t.Context(), []string{"--config", "/does/not/exist"}, nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(nil handler builder) error = %v, want ErrRejected", err)
	}
}

func TestMessagingCompositionRejectsDisabledTransportWithRegisteredHandler(t *testing.T) {
	configtest.IsolateEnv(t)
	// profile:authn-oidc-jwt:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	// profile:authn-oidc-jwt:end
	built := false
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*natsjs.Registry, func(context.Context), error) {
		built = true
		return nil, nil, nil
	})
	if !errors.Is(err, natsjs.ErrRejected) || !strings.Contains(err.Error(), "messaging must be enabled for worker") {
		t.Fatalf("run(disabled messaging) error = %v, want disabled ErrRejected", err)
	}
	if built {
		t.Fatal("worker built the feature handler while messaging was disabled")
	}
}

func TestMessagingCompositionRejectsShutdownBudgetWithoutCleanupTail(t *testing.T) {
	configtest.IsolateEnv(t)
	// profile:authn-oidc-jwt:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	// profile:authn-oidc-jwt:end
	t.Setenv("APP__HTTP__GRACE_PERIOD", "25s")
	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "0s")

	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*natsjs.Registry, func(context.Context), error) {
		return nil, nil, nil
	})
	if !errors.Is(err, config.ErrValidate) || !strings.Contains(err.Error(), "post-drain teardown budget") {
		t.Fatalf("run(unreserved shutdown budget) error = %v, want ErrValidate naming the cleanup tail", err)
	}
}

func TestMessagingCompositionParsesWorkerBounds(t *testing.T) {
	cfg := config.MessagingConfig{
		MaxPayloadBytes: 256 << 10,
		Worker: config.MessagingWorkerConfig{
			Consumer: "events-worker", FilterSubject: "events.>", DeadLetterSubject: "dead.events",
			MaxConcurrency: 8,
		},
	}
	got, err := messagingWorkerConfig(cfg)
	if err != nil {
		t.Fatalf("messagingWorkerConfig() error = %v", err)
	}
	if got.MaxDeliveryBytes != cfg.MaxPayloadBytes+natsjs.HeaderLimitBytes || got.HandlerTimeout != 30*time.Second || len(got.RetryDelays) != 4 {
		t.Fatalf("worker defaults = %#v", got)
	}
}

func TestMessagingCompositionRejectsInvalidWorkerBeforeConnection(t *testing.T) {
	cfg := config.MessagingConfig{MaxPayloadBytes: 256 << 10}
	if _, err := messagingWorkerConfig(cfg); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("messagingWorkerConfig(invalid) error = %v, want ErrRejected", err)
	}
}

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestWorkerLoggerCorrelatesRecords(t *testing.T) {
	telemetrytest.InstallSpanRecorder(t)

	var out bytes.Buffer
	log := runtimeopts.Logger(&out, config.Config{
		App:           config.AppConfig{Env: "test", Version: "v1"},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{ServiceName: "worker"}},
	})
	ctx, span := otel.Tracer("test").Start(context.Background(), "worker-log-test")
	log.InfoContext(ctx, "worker_message")
	span.End()

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal worker log %q: %v", out.String(), err)
	}
	for _, key := range []string{"trace_id", "span_id"} {
		if value, ok := record[key].(string); !ok || value == "" {
			t.Fatalf("worker log %v is missing %s", record, key)
		}
	}
}

//nolint:paralleltest // Telemetry setup installs process-wide providers.
func TestWorkerTelemetrySetupCanBeCleanedWithinCallerBudget(t *testing.T) {
	telemetrytest.RestoreGlobals(t)
	telemetrytest.ClearAmbientExporterEnv(t)

	cleanup, err := runtimeopts.InstallTelemetry(t.Context(), config.Config{
		App: config.AppConfig{
			Env: "test", Version: "v1", Commit: "test-commit", InstanceID: "worker-test",
		},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{
			ServiceName: "worker", TracesSampler: "always_off",
		}},
	}, telemetry.New(), slog.New(slog.DiscardHandler), "worker")
	if err != nil {
		t.Fatalf("runtimeopts.InstallTelemetry() error = %v", err)
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = cleanup(cleanupCtx)
}

func TestWorkerCompositionHelpers(t *testing.T) {
	options, err := parseLoadOptions([]string{
		"--config", "config.yaml",
		"--config-overlay", "first.yaml",
		"--config-overlay", "second.yaml",
	})
	if err != nil {
		t.Fatalf("parseLoadOptions() error = %v", err)
	}
	if options.ConfigPath != "config.yaml" || len(options.ConfigOverlays) != 2 || options.ConfigOverlays[1] != "second.yaml" {
		t.Fatalf("parseLoadOptions() = %+v", options)
	}
	for _, args := range [][]string{
		{"--config", ""},
		{"--config-overlay", ""},
		{"unexpected"},
	} {
		if _, err := parseLoadOptions(args); err == nil {
			t.Fatalf("parseLoadOptions(%q) error = nil", args)
		}
	}

	clientCfg := runtimeopts.Messaging(config.MessagingConfig{
		URLs: " nats://one:4222, nats://two:4222 ", Stream: "EVENTS", MaxPayloadBytes: 1024,
	})
	if len(clientCfg.URLs) != 2 || clientCfg.URLs[0] != "nats://one:4222" || clientCfg.URLs[1] != "nats://two:4222" {
		t.Fatalf("runtimeopts.Messaging() URLs = %q", clientCfg.URLs)
	}
	if clientCfg.Stream != "EVENTS" || clientCfg.MaxPayloadBytes != 1024 {
		t.Fatalf("runtimeopts.Messaging() = %+v", clientCfg)
	}
}

func TestHandlerCleanupSafetyTracksWorkerExit(t *testing.T) {
	stopped := make(chan struct{})
	close(stopped)
	if !handlerStoppedBeforeReturn(nil, stopped) {
		t.Fatal("completed graceful worker was not safe to clean up")
	}
	if !handlerStoppedBeforeReturn(errors.New("forced"), stopped) {
		t.Fatal("completed forced worker was not safe to clean up")
	}
	if handlerStoppedBeforeReturn(errors.New("forced"), make(chan struct{})) {
		t.Fatal("running forced handler was marked safe to clean up")
	}
}

// TestWorkerRunLoopPanicIsRecovered covers the loop this process exists to run.
// It ran bare, so a panic in the fetch or ack bookkeeping took the process down
// before the drain, the handler join, and the telemetry flush that records why.
//
// Both cases also pin the send-then-close order the drain depends on: it decides
// cleanup safety from done and only then looks for a result that has not been
// read, so a done that closed first would let a real exit reason go unreported.
func TestWorkerRunLoopPanicIsRecovered(t *testing.T) {
	const poison = "worker-panic-marker-8c23"
	stopped := errors.New("worker stopped")

	for _, testCase := range []struct {
		name string
		run  func(context.Context) error
		want error
	}{
		{name: "panic", run: func(context.Context) error { panic(poison) }, want: errWorkerPanic},
		{name: "return", run: func(context.Context) error { return stopped }, want: stopped},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var records strings.Builder
			result := make(chan error, 1)
			done := make(chan struct{})

			go superviseWorkerRun(
				t.Context(), slog.New(slog.NewJSONHandler(&records, nil)), testCase.run, result, done,
			)

			<-done
			var got error
			select {
			case got = <-result:
			default:
				t.Fatal("superviseWorkerRun() closed done before reporting a result")
			}
			if !errors.Is(got, testCase.want) {
				t.Fatalf("superviseWorkerRun() err = %v, want %v", got, testCase.want)
			}
			select {
			case second := <-result:
				t.Fatalf("superviseWorkerRun() reported a second result %v", second)
			default:
			}

			// The panic's value reaches neither the reported error nor the record:
			// only its type, its class, and the stack do. The shared panic-attribute
			// constructor owns that argument.
			logged := records.String()
			if strings.Contains(got.Error(), poison) || strings.Contains(logged, poison) {
				t.Fatalf("panic value leaked: err=%v records=%s", got, logged)
			}
			if errors.Is(testCase.want, errWorkerPanic) && !strings.Contains(logged, "worker_run_loop_panic") {
				t.Fatalf("recovered panic was not recorded: %s", logged)
			}
		})
	}
}

//nolint:paralleltest // Worker setup installs process-wide telemetry providers.
func TestMessagingCompositionDoesNotBuildHandlerBeforeBrokerAdmission(t *testing.T) {
	telemetrytest.RestoreGlobals(t)
	telemetrytest.ClearAmbientExporterEnv(t)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable worker endpoint: %v", err)
	}
	url := "nats://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close unavailable worker endpoint: %v", err)
	}
	setWorkerTestEnvironment(t, url, "127.0.0.1:0")
	built := false
	err = run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*natsjs.Registry, func(context.Context), error) {
		built = true
		return testRegistry(t, "test", func(context.Context, string) error { return nil }), nil, nil
	})
	if err == nil {
		t.Fatal("run(unavailable broker) error = nil")
	}
	if built {
		t.Fatal("worker built the feature handler before broker topology admission")
	}
}

func TestMessagingCompositionRejectsMissingDiagnosticsBeforeConnection(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for worker connection sentinel: %v", err)
	}
	accepted := make(chan struct{}, 1)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = connection.Close()
			accepted <- struct{}{}
		}
	}()
	setWorkerTestEnvironment(t, "nats://"+listener.Addr().String(), "")
	err = run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*natsjs.Registry, func(context.Context), error) {
		return testRegistry(t, "test", func(context.Context, string) error { return nil }), nil, nil
	})
	if !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(missing diagnostics) error = %v, want ErrRejected", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close worker connection sentinel: %v", err)
	}
	<-acceptDone
	select {
	case <-accepted:
		t.Fatal("worker connected before rejecting missing diagnostics address")
	default:
	}
}

func setWorkerTestEnvironment(t *testing.T, messagingURL, diagnosticsAddr string) {
	t.Helper()
	configtest.IsolateEnv(t)
	for key, value := range map[string]string{
		// profile:authn-oidc-jwt:start
		"APP__AUTHN__ISSUER":   "https://issuer.example.com",
		"APP__AUTHN__AUDIENCE": "https://api.example.com",
		// profile:authn-oidc-jwt:end
		"APP__MESSAGING__URLS":                        messagingURL,
		"APP__MESSAGING__ALLOW_PLAINTEXT":             "true",
		"APP__MESSAGING__ALLOW_UNAUTHENTICATED":       "true",
		"APP__MESSAGING__STREAM":                      "EVENTS",
		"APP__MESSAGING__WORKER__CONSUMER":            "missing-diagnostics-worker",
		"APP__MESSAGING__WORKER__FILTER_SUBJECT":      "events.test",
		"APP__MESSAGING__WORKER__DEAD_LETTER_SUBJECT": "dead.events.test",
		"APP__OBSERVABILITY__METRICS__ADDR":           diagnosticsAddr,
	} {
		t.Setenv(key, value)
	}
}
