package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestRunReturnsParseErrorForInvalidFlags(t *testing.T) {
	t.Parallel()

	err := runWithRuntime([]string{"--unknown-flag"}, testRuntimeWiring())
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
		budget.start(context.Background())
		defer budget.close()

		if got := budget.clamp(4 * time.Second); got != 4*time.Second {
			t.Fatalf("clamp(4s) with the whole period left = %s, want 4s", got)
		}

		time.Sleep(9 * time.Second)
		if got := budget.clamp(4 * time.Second); got != time.Second {
			t.Fatalf("clamp(4s) with 1s left = %s, want 1s", got)
		}

		// Past the deadline no stage may extend the process-wide grace period.
		time.Sleep(2 * time.Second)
		if got := budget.clamp(4 * time.Second); got != 0 {
			t.Fatalf("clamp(4s) past the deadline = %s, want zero", got)
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
		budget.start(context.Background())
		defer budget.close()
		// A second caller must not restart it: several points can each be the
		// first to observe that serving ended.
		budget.start(context.Background())

		if got := budget.clamp(time.Hour); got != 10*time.Second {
			t.Fatalf("clamp() after an hour of serving = %s, want the full grace period", got)
		}
	})
}

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
	// profile:authn-bearer:start
	// Authentication trust has no executable defaults. This test is about the
	// shipped shutdown budgets, so it supplies the unrelated required policy.
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "service-api")
	// profile:authn-bearer:end
	// profile:object-storage:start
	setObjectStorageBootstrapTestEnv(t)
	// profile:object-storage:end
}

// profile:object-storage:start
//
//nolint:paralleltest // This test mutates process-global environment or working directory.
func setObjectStorageBootstrapTestEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"APP__OBJECT_STORAGE__PROVIDER":              "amazon_s3",
		"APP__OBJECT_STORAGE__REGION":                "us-east-1",
		"APP__OBJECT_STORAGE__BUCKET":                "examplebucket",
		"APP__OBJECT_STORAGE__EXPECTED_BUCKET_OWNER": "123456789012",
		"APP__OBJECT_STORAGE__CREDENTIAL_SOURCE":     "aws_default",
		"APP__OBJECT_STORAGE__MAX_OBJECT_BYTES":      "10485760",
	} {
		t.Setenv(key, value)
	}
}

// profile:object-storage:end

func testRuntimeWiring() runtimeWiring {
	wiring := productionRuntimeWiring()
	// profile:object-storage:start
	wiring.initObjectStorage = func(context.Context, config.ObjectStorageConfig) (objectStorageRuntime, error) {
		return &countingObjectStorageRuntime{}, nil
	}
	// profile:object-storage:end
	return wiring
}
