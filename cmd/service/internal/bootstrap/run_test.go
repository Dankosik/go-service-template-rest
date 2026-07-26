package bootstrap

import (
	"errors"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestParseLoadOptions(t *testing.T) {
	t.Parallel()

	t.Run("parses flags", func(t *testing.T) {
		t.Parallel()

		opts, err := parseLoadOptions([]string{
			"--config", "/tmp/base.yaml",
			"--config-overlay", "  /tmp/o1.yaml  ",
			"--config-overlay", "/tmp/o2.yaml",
		})
		if err != nil {
			t.Fatalf("parseLoadOptions() error = %v, want nil", err)
		}
		if opts.ConfigPath != "/tmp/base.yaml" {
			t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "/tmp/base.yaml")
		}
		if len(opts.ConfigOverlays) != 2 || opts.ConfigOverlays[0] != "/tmp/o1.yaml" || opts.ConfigOverlays[1] != "/tmp/o2.yaml" {
			t.Fatalf("ConfigOverlays = %v, want trimmed [/tmp/o1.yaml /tmp/o2.yaml]", opts.ConfigOverlays)
		}
	})

	t.Run("fails on empty overlay path", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config-overlay", "   "})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
	})

	t.Run("fails on empty base path", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config", "   "})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
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

// testShutdownBudget is the teardown deadline for tests that exercise one stage
// in isolation. The grace period is generous on purpose: these tests assert what
// a stage does with its own ceiling, not what the clamp does to it.
func testShutdownBudget() *shutdownBudget {
	return newShutdownBudget(10 * time.Minute)
}

// TestShippedDefaultsFitTheGracePeriod is the arithmetic that used to be nobody's
// job.
//
// http.shutdown_timeout bounded the drain, and four hard-coded constants bounded
// everything after it, so the worst-case teardown was their sum — 47s against the
// 45s railway.toml grants and the 30s Kubernetes grants by default. Nothing
// related the two, so the overrun was only ever observable as a SIGKILL that took
// the shutdown telemetry with it.
func TestShippedDefaultsFitTheGracePeriod(t *testing.T) {
	resetShutdownConfigEnv(t)

	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		t.Fatalf("config.LoadDetailed() error = %v", err)
	}
	if err := validateShutdownGraceBudget(cfg); err != nil {
		t.Fatalf("shipped defaults do not fit their own grace period: %v", err)
	}
}

func TestValidateShutdownGraceBudgetRejectsADrainThatCannotFit(t *testing.T) {
	t.Parallel()

	// A Kubernetes deployment left at the default 30s grace period, with a drain
	// budget that leaves nothing for the teardown behind it.
	cfg := config.Config{HTTP: config.HTTPConfig{
		GracePeriod:     30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}}

	err := validateShutdownGraceBudget(cfg)
	if err == nil {
		t.Fatal("validateShutdownGraceBudget() error = nil, want the overrun rejected at startup")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want config.ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.grace_period") {
		t.Fatalf("error = %v, want the setting an operator can edit named", err)
	}
}

// TestShutdownBudgetClampsStagesToTheRemainingGracePeriod is what makes the
// ordering worth anything: a stage that asks for more than the grace period has
// left gets what is left, so the stages behind it still run.
func TestShutdownBudgetClampsStagesToTheRemainingGracePeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		budget := newShutdownBudget(10 * time.Second)
		budget.start()

		if got := budget.clamp(4 * time.Second); got != 4*time.Second {
			t.Fatalf("clamp(4s) with the whole period left = %s, want 4s", got)
		}

		time.Sleep(9 * time.Second)
		if got := budget.clamp(4 * time.Second); got != time.Second {
			t.Fatalf("clamp(4s) with 1s left = %s, want 1s", got)
		}

		// Past the deadline a stage still gets a floor. The process is about to be
		// killed either way, and a stage given nothing cannot report that it was
		// cut short.
		time.Sleep(2 * time.Second)
		if got := budget.clamp(4 * time.Second); got != shutdownStageFloor {
			t.Fatalf("clamp(4s) past the deadline = %s, want the floor %s", got, shutdownStageFloor)
		}
	})
}

// TestShutdownBudgetStartsWhenTeardownBegins keeps the clock off the process
// lifetime. A deadline taken at startup would be spent before the first request
// was ever served.
func TestShutdownBudgetStartsWhenTeardownBegins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		budget := newShutdownBudget(10 * time.Second)

		time.Sleep(time.Hour)
		budget.start()
		// A second caller must not restart it: several points can each be the
		// first to observe that serving ended.
		budget.start()

		if got := budget.clamp(time.Hour); got != 10*time.Second {
			t.Fatalf("clamp() after an hour of serving = %s, want the full grace period", got)
		}
	})
}

// resetShutdownConfigEnv clears the ambient APP__ variables a developer shell may
// carry, so the defaults under test are the shipped ones.
func resetShutdownConfigEnv(t *testing.T) {
	t.Helper()

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "APP__") {
			t.Setenv(name, "")
			if err := os.Unsetenv(name); err != nil {
				t.Fatalf("unset %s: %v", name, err)
			}
		}
	}
}
