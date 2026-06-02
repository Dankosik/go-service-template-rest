package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/config"
	"github.com/Dankosik/billing-service/internal/infra/telemetry"
)

func TestOverlayPathsFlagSetAndString(t *testing.T) {
	t.Parallel()

	var f overlayPathsFlag
	if err := f.Set("  a.yaml  "); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if err := f.Set("b.yaml"); err != nil {
		t.Fatalf("Set() second error = %v, want nil", err)
	}
	if got := f.String(); got != "a.yaml,b.yaml" {
		t.Fatalf("String() = %q, want %q", got, "a.yaml,b.yaml")
	}
	if err := f.Set("   "); err == nil {
		t.Fatal("Set(empty) error = nil, want non-nil")
	}
}

func TestParseLoadOptions(t *testing.T) {
	t.Parallel()

	t.Run("parses flags", func(t *testing.T) {
		t.Parallel()

		opts, err := parseLoadOptions([]string{
			"--config", "/tmp/base.yaml",
			"--config-overlay", "/tmp/o1.yaml",
			"--config-overlay", "/tmp/o2.yaml",
			"--config-strict",
		})
		if err != nil {
			t.Fatalf("parseLoadOptions() error = %v, want nil", err)
		}
		if opts.ConfigPath != "/tmp/base.yaml" {
			t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "/tmp/base.yaml")
		}
		if !opts.Strict {
			t.Fatal("Strict = false, want true")
		}
		if len(opts.ConfigOverlays) != 2 {
			t.Fatalf("ConfigOverlays len = %d, want 2", len(opts.ConfigOverlays))
		}
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--unknown-flag"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
	})

	t.Run("fails on positional arguments", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config", "/tmp/base.yaml", "serve"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
		if !strings.Contains(err.Error(), "serve") {
			t.Fatalf("parseLoadOptions() err = %v, want unexpected positional argument detail", err)
		}
	})
}

func TestRunReturnsParseErrorForInvalidFlags(t *testing.T) {
	t.Parallel()

	err := Run([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Fatalf("Run() err = %v, want parse flags context", err)
	}
}

func TestBuildBillingAuthorityHandlerFailsClosedUntilRuntimeIsReady(t *testing.T) {
	t.Parallel()

	handler, err := buildBillingAuthorityHandler(config.Config{}, nil)
	if err != nil {
		t.Fatalf("buildBillingAuthorityHandler(disabled) error = %v, want nil", err)
	}
	if handler != nil {
		t.Fatalf("buildBillingAuthorityHandler(disabled) handler = %T, want nil fail-closed handler", handler)
	}

	_, err = buildBillingAuthorityHandler(config.Config{
		Authority: config.AuthorityConfig{Enabled: true},
	}, nil)
	if err == nil {
		t.Fatal("buildBillingAuthorityHandler(enabled,nil pool) error = nil, want fail-closed startup error")
	}
	if !strings.Contains(err.Error(), "postgres pool is required") {
		t.Fatalf("buildBillingAuthorityHandler(enabled,nil pool) error = %v, want postgres pool context", err)
	}
}

func TestRuntimeIngressAdmissionGuardEnforcesPolicyAtReadiness(t *testing.T) {
	t.Parallel()

	var nilGuard *runtimeIngressAdmissionGuard
	if err := nilGuard.Check(context.Background()); err != nil {
		t.Fatalf("nil guard Check() error = %v, want nil", err)
	}
	if err := newRuntimeIngressAdmissionGuard(networkPolicy{}).Check(context.Background()); err != nil {
		t.Fatalf("permissive guard Check() error = %v, want nil", err)
	}

	guard := newRuntimeIngressAdmissionGuard(networkPolicy{ingressDeclarationRequired: true})
	err := guard.Check(context.Background())
	if err == nil {
		t.Fatal("guard Check() error = nil, want ingress policy rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("guard Check() error = %v, want dependency init rejection", err)
	}
	if !strings.Contains(err.Error(), envNetworkPublicIngressEnabled) {
		t.Fatalf("guard Check() error = %v, want public ingress env context", err)
	}
}

//nolint:paralleltest // bootstrapLoggerStage mutates the process-wide slog default.
func TestBootstrapStagesReportSafeStartupState(t *testing.T) {
	ctx := context.Background()
	tracer, bootstrapCtx, bootstrapSpan := bootstrapTraceStage(ctx)
	defer bootstrapSpan.End()

	if startupBootstrapContext(ctx, bootstrapSpan) == ctx {
		t.Fatal("startupBootstrapContext() returned original context, want span-bound context")
	}

	cfg := config.Config{
		App: config.AppConfig{
			Env:     "local",
			Version: "test",
		},
		HTTP: config.HTTPConfig{
			Addr: ":8080",
		},
		Log: config.LogConfig{Level: slog.LevelInfo},
		Observability: config.ObservabilityConfig{
			OTel: config.OTelConfig{ServiceName: "billing-service-test"},
		},
		Redis: config.RedisConfig{Mode: config.RedisModeCache},
	}
	log := bootstrapLoggerStage(cfg)
	if log == nil {
		t.Fatal("bootstrapLoggerStage() returned nil logger")
	}
	bootstrapReportStage(
		bootstrapCtx,
		tracer,
		slog.New(slog.DiscardHandler),
		cfg,
		config.LoadOptions{
			ConfigPath:     "config.yaml",
			ConfigOverlays: []string{"overlay.yaml"},
			Strict:         true,
		},
		config.LoadReport{
			LoadDuration:         time.Millisecond,
			LoadDefaultsDuration: time.Millisecond,
			LoadFileDuration:     time.Millisecond,
			LoadEnvDuration:      time.Millisecond,
			ParseDuration:        time.Millisecond,
			ValidateDuration:     time.Millisecond,
			UnknownKeyWarnings:   []string{"unknown.safe_key"},
		},
		errors.New("setup tracing: telemetry egress target denied"),
	)
}

func TestBootstrapRuntimeReturnsCanceledConfigLoadBeforeDependencies(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bootstrap, err := bootstrapRuntime(ctx, config.LoadOptions{}, telemetry.New())
	if err == nil {
		t.Fatalf("bootstrapRuntime(canceled) bootstrap = %+v, error = nil; want cancellation", bootstrap)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("bootstrapRuntime(canceled) error = %v, want context.Canceled", err)
	}
}

func TestReleaseSignalNotificationOnDoneReleasesOnceAfterCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var stopCalls atomic.Int32
	stopCalled := make(chan struct{})

	release := releaseSignalNotificationOnDone(ctx, func() {
		if stopCalls.Add(1) == 1 {
			close(stopCalled)
		}
	})
	defer release()

	cancel()

	select {
	case <-stopCalled:
	case <-time.After(time.Second):
		t.Fatal("stop callback was not called after context cancellation")
	}

	release()
	release()
	if got := stopCalls.Load(); got != 1 {
		t.Fatalf("stop callback calls = %d, want 1", got)
	}
}
