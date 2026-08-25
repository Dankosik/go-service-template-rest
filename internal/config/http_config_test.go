package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestShutdownTimeoutCanBeTunedWhenDrainBudgetIsValid(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "20s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for tuned shutdown timeout", err)
	}
	if cfg.HTTP.ShutdownTimeout != 45*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 45s", cfg.HTTP.ShutdownTimeout)
	}
}

func TestShutdownTimeoutMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "500ms")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "0s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "100ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected validation error for shutdown timeout range")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.shutdown_timeout must be in range") {
		t.Fatalf("error = %v, want shutdown timeout range policy", err)
	}
}

func TestHTTPShutdownBudgetMustLeaveWriteDrainTime(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "20s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected validation error for write timeout beyond drain budget")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.write_timeout must be <= effective drain budget") {
		t.Fatalf("error = %v, want explicit drain budget policy", err)
	}
}

func TestReadinessTimeoutMustNotExceedWriteTimeout(t *testing.T) {
	t.Run("greater readiness timeout rejects", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__READINESS_TIMEOUT", "6s")
		t.Setenv("APP__HTTP__REQUEST_TIMEOUT", "4s")
		t.Setenv("APP__HTTP__WRITE_TIMEOUT", "5s")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatal("LoadDetailed() expected validation error for readiness timeout beyond write timeout")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.readiness_timeout must be <= http.write_timeout") {
			t.Fatalf("error = %v, want readiness/write timeout compatibility policy", err)
		}
	})

	for _, tc := range []struct {
		name             string
		readinessTimeout string
		writeTimeout     string
	}{
		{name: "equal timeout allows", readinessTimeout: "5s", writeTimeout: "5s"},
		{name: "lower readiness timeout allows", readinessTimeout: "4s", writeTimeout: "5s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", tc.readinessTimeout)
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", "4s")
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

// The readiness/health-check relationship is owned by
// bootstrap.validateStartupBudgetCompatibility, which enforces it with the
// startup headroom this package cannot see. It is proved there.
// The request budget must expire while the connection still has enough write
// budget to carry the 504 that reports it.
//
//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestRequestTimeoutLeavesTerminalResponseReserve(t *testing.T) {
	for _, tc := range []struct {
		name           string
		requestTimeout string
		writeTimeout   string
	}{
		{name: "request outlasts writer", requestTimeout: "6s", writeTimeout: "5s"},
		{name: "equal deadlines", requestTimeout: "5s", writeTimeout: "5s"},
		{name: "subsecond reserve", requestTimeout: "4500ms", writeTimeout: "5s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", "1s")
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.requestTimeout)
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() expected terminal response reserve validation error")
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), "must leave at least 1s") {
				t.Fatalf("error = %v, want terminal response reserve policy", err)
			}
		})
	}

	for _, tc := range []struct {
		name           string
		requestTimeout string
		writeTimeout   string
	}{
		{name: "exact reserve", requestTimeout: "4s", writeTimeout: "5s"},
		{name: "default headroom", requestTimeout: "8s", writeTimeout: "10s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", "1s")
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.requestTimeout)
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestRequestTimeoutRejectsOutOfRangeValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "below lower bound", value: "50ms"},
		{name: "above upper bound", value: "11m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.value)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatalf("LoadDetailed() expected validation error for http.request_timeout = %q", tc.value)
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), "http.request_timeout must be in range") {
				t.Fatalf("error = %v, want request timeout range policy", err)
			}
		})
	}
}

func TestMaxInFlightBounds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "default accepted"},
		{name: "negative", value: "-1", wantErr: true},
		{name: "above ceiling", value: "100001", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.value != "" {
				t.Setenv("APP__HTTP__MAX_IN_FLIGHT", tc.value)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestHTTPMaxInFlightMayDisableShedding(t *testing.T) {
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	cfg.HTTP.MaxInFlight = 0
	if err := validateHTTPConfig(&cfg.HTTP); err != nil {
		t.Fatalf("validateHTTPConfig() error = %v, want zero to disable shedding", err)
	}
}

func TestMaxConnectionsBounds(t *testing.T) {
	for _, tc := range []struct {
		name        string
		connections string
		inFlight    string
		wantErr     bool
	}{
		{name: "default accepted"},
		{name: "zero accepts without a bound", connections: "0"},
		{name: "negative", connections: "-1", wantErr: true},
		{name: "above ceiling", connections: "1000001", wantErr: true},
		{name: "equal to in flight accepted", connections: "256", inFlight: "256"},
		// A cap under the advertised concurrency means excess callers never get
		// the 503 with a Retry-After that shedding would have given them; they
		// wait in the kernel backlog and time out at connect instead.
		{name: "below in flight", connections: "128", inFlight: "256", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.connections != "" {
				t.Setenv("APP__HTTP__MAX_CONNECTIONS", tc.connections)
			}
			if tc.inFlight != "" {
				t.Setenv("APP__HTTP__MAX_IN_FLIGHT", tc.inFlight)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}
