package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
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
	if err := worker.Run(ctx); err != nil {
		return fmt.Errorf("run billing worker: %w", err)
	}
	return nil
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
