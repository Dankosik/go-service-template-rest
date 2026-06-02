package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dankosik/billing-service/internal/app/microleaseworker"
	"github.com/Dankosik/billing-service/internal/config"
)

type overlayPathsFlag []string

func (f *overlayPathsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *overlayPathsFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("config overlay path cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

func Run(args []string) error {
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, _, err := config.LoadDetailedWithContext(ctx, loadOptions)
	if err != nil {
		return fmt.Errorf("load billing worker config: %w", err)
	}
	if !cfg.Microlease.WorkerEnabled {
		slog.Info("billing worker disabled by config")
		return nil
	}

	worker, cleanup, err := buildWorkerRuntime(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build billing worker runtime: %w", err)
	}
	defer cleanup()
	healthServer := newWorkerHealthServer(cfg.HTTP, worker)
	healthErrCh := make(chan error, 1)
	go func() {
		slog.Info("billing worker health server started", "addr", cfg.HTTP.Addr)
		err := healthServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			healthErrCh <- err
			return
		}
		healthErrCh <- nil
	}()
	defer shutdownWorkerHealthServer(healthServer, cfg.HTTP.ShutdownTimeout)

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- worker.Run(ctx)
	}()

	select {
	case err := <-healthErrCh:
		if err != nil {
			stop()
			<-workerErrCh
			return fmt.Errorf("serve billing worker health: %w", err)
		}
		return nil
	case err := <-workerErrCh:
		if err != nil {
			return fmt.Errorf("run billing worker: %w", err)
		}
		return nil
	}
}

type workerReadyChecker interface {
	Ready(context.Context) error
}

func newWorkerHealthServer(cfg config.HTTPConfig, checker workerReadyChecker) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           workerHealthHandler(cfg, checker),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}

func workerHealthHandler(cfg config.HTTPConfig, checker workerReadyChecker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if checker == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		readyCtx, cancel := context.WithTimeout(r.Context(), cfg.ReadinessTimeout)
		defer cancel()
		if err := checker.Ready(readyCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func shutdownWorkerHealthServer(srv *http.Server, timeout time.Duration) {
	if srv == nil {
		return
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("billing worker health server shutdown failed", "err", err)
	}
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var overlays overlayPathsFlag
	flags := flag.NewFlagSet("billing-worker", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "path to base config file")
	flags.Var(&overlays, "config-overlay", "path to config overlay file (repeatable)")
	configStrict := flags.Bool("config-strict", false, "enable strict unknown-key validation")
	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse args: %w", err)
	}
	return config.LoadOptions{
		ConfigPath:     strings.TrimSpace(*configPath),
		ConfigOverlays: []string(overlays),
		Strict:         *configStrict,
		LoadBudget:     10 * time.Second,
		ValidateBudget: 2 * time.Second,
	}, nil
}

func workerConfig(cfg config.Config) microleaseworker.Config {
	return microleaseworker.Config{
		ReadinessTimeout: cfg.Redpanda.HealthcheckTimeout,
		DefaultInterval:  cfg.Microlease.AdmissionControlRenewalInterval,
		ShutdownTimeout:  cfg.HTTP.ShutdownTimeout,
	}
}
