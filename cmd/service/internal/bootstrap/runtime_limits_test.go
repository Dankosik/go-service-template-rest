package bootstrap

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestApplyMemoryLimitSkipsWhenDisabled(t *testing.T) {
	restoreMemoryLimit(t)

	var logged bytes.Buffer
	applyMemoryLimit(slog.New(slog.NewJSONHandler(&logged, nil)), 0)

	if got := debug.SetMemoryLimit(-1); got != math.MaxInt64 {
		t.Fatalf("memory limit = %d, want it left at the default", got)
	}
	if !strings.Contains(logged.String(), "detection_disabled") {
		t.Fatalf("log = %q, want a disabled-detection record", logged.String())
	}
}

// TestApplyMemoryLimitDefersToPlatform keeps the template from overriding a limit
// the deployment already chose.
func TestApplyMemoryLimitDefersToPlatform(t *testing.T) {
	restoreMemoryLimit(t)
	t.Setenv(memoryLimitEnv, "512MiB")

	var logged bytes.Buffer
	applyMemoryLimit(slog.New(slog.NewJSONHandler(&logged, nil)), 0.9)

	if got := debug.SetMemoryLimit(-1); got != math.MaxInt64 {
		t.Fatalf("memory limit = %d, want the platform's own setting left alone", got)
	}
	if !strings.Contains(logged.String(), "already_set_by_platform") {
		t.Fatalf("log = %q, want a deferral record", logged.String())
	}
}

// TestApplyMemoryLimitReportsItsDecision is what makes this debuggable: an
// operator seeing exit 137 needs the resolved number in the boot log, not an
// inference from a restart count.
func TestApplyMemoryLimitReportsItsDecision(t *testing.T) {
	restoreMemoryLimit(t)
	t.Setenv(memoryLimitEnv, "")

	var logged bytes.Buffer
	const detected = uint64(512 << 20)
	got := applyMemoryLimitFrom(
		slog.New(slog.NewJSONHandler(&logged, nil)),
		0.9,
		func() (uint64, error) { return detected, nil },
	)

	line := logged.String()
	if got != int64(detected) {
		t.Fatalf("detected limit = %d, want %d", got, detected)
	}
	if got, want := debug.SetMemoryLimit(-1), int64(detected*9/10); got != want {
		t.Fatalf("memory limit = %d, want %d", got, want)
	}
	if !strings.Contains(line, "runtime_memory_limit_applied") {
		t.Fatalf("log = %q, want an applied record", line)
	}
	if !strings.Contains(line, `"component":"runtime_limits"`) {
		t.Fatalf("log = %q, want the runtime_limits component", line)
	}
}

// restoreMemoryLimit puts the process-wide GC limit back after a test touches it.
// SetMemoryLimit(-1) reads the current value without changing it.
func restoreMemoryLimit(tb testing.TB) {
	tb.Helper()

	previous := debug.SetMemoryLimit(-1)
	tb.Cleanup(func() { debug.SetMemoryLimit(previous) })
}

// TestReportRequestBufferBudget covers the arithmetic nothing else performs:
// http.max_in_flight and http.max_body_bytes are validated independently, so
// their product against the container's memory limit is a number no layer
// computes until the kernel does it.
func TestReportRequestBufferBudget(t *testing.T) {
	t.Parallel()

	const gibibyte = int64(1) << 30

	budgetConfig := func(maxInFlight int, maxBodyBytes int64) config.Config {
		return config.Config{
			HTTP:    config.HTTPConfig{MaxInFlight: maxInFlight, MaxBodyBytes: maxBodyBytes},
			Runtime: config.RuntimeConfig{MemoryLimitRatio: 0.9},
		}
	}

	tests := []struct {
		name  string
		cfg   config.Config
		limit int64
		want  bool
	}{
		{
			// The shipped defaults: 256 x 1 MiB is 256 MiB against a
			// 1 GiB container, which is over a quarter of the GC's budget.
			name:  "shipped defaults on a small container are reported",
			cfg:   budgetConfig(256, 1<<20),
			limit: gibibyte,
			want:  true,
		},
		{
			name:  "shipped defaults on a roomy container are silent",
			cfg:   budgetConfig(256, 1<<20),
			limit: 8 * gibibyte,
		},
		{
			// The week-two change: raising the body limit for uploads without
			// touching the concurrency ceiling.
			name:  "raising the body limit for uploads is reported",
			cfg:   budgetConfig(256, 10<<20),
			limit: 8 * gibibyte,
			want:  true,
		},
		{
			name:  "no container limit leaves nothing to relate to",
			cfg:   budgetConfig(256, 10<<20),
			limit: 0,
		},
		{
			// Shedding disabled means the ceiling is however many requests
			// arrive, so there is no product to compare.
			name:  "disabled shedding has no bounded product",
			cfg:   budgetConfig(0, 10<<20),
			limit: gibibyte,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var logged bytes.Buffer
			reportRequestBufferBudget(slog.New(slog.NewJSONHandler(&logged, nil)), tc.cfg, tc.limit)

			got := strings.Contains(logged.String(), "runtime_request_buffer_budget_exceeded")
			if got != tc.want {
				t.Fatalf("reported = %v, want %v; log = %s", got, tc.want, logged.String())
			}
		})
	}
}

// TestBoundedAPIListenerCapsAcceptedConnections covers the half of overload the
// middleware chain cannot see: MaxInFlight sheds inside a handler, so every
// connection beyond its limit has already cost a goroutine and its buffers by
// the time it is rejected.
func TestBoundedAPIListenerCapsAcceptedConnections(t *testing.T) {
	t.Parallel()

	t.Run("zero leaves the listener unwrapped", func(t *testing.T) {
		t.Parallel()

		var listenConfig net.ListenConfig
		base, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
		t.Cleanup(func() { _ = base.Close() })

		if got := boundedAPIListener(base, 0); got != base {
			t.Fatal("boundedAPIListener(0) wrapped the listener; an unbounded accept must cost nothing")
		}
	})

	t.Run("the cap bounds concurrently served connections", func(t *testing.T) {
		t.Parallel()

		const limit = 2

		var listenConfig net.ListenConfig
		base, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
		listener := boundedAPIListener(base, limit)
		t.Cleanup(func() { _ = listener.Close() })

		release := make(chan struct{})
		releaseAll := sync.OnceFunc(func() { close(release) })
		defer releaseAll()
		limitReached := make(chan struct{})
		signalLimitReached := sync.OnceFunc(func() { close(limitReached) })
		var concurrent atomic.Int64
		var peak atomic.Int64
		server := &http.Server{
			ReadHeaderTimeout: time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				current := concurrent.Add(1)
				for {
					observed := peak.Load()
					if current <= observed || peak.CompareAndSwap(observed, current) {
						break
					}
				}
				if peak.Load() == limit {
					signalLimitReached()
				}
				<-release
				concurrent.Add(-1)
				w.WriteHeader(http.StatusOK)
			}),
		}
		go func() { _ = server.Serve(listener) }()
		t.Cleanup(func() { _ = server.Close() })

		var wg sync.WaitGroup
		client := &http.Client{Timeout: 5 * time.Second}
		for range limit * 3 {
			wg.Go(func() {
				request, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+base.Addr().String()+"/", http.NoBody)
				if reqErr != nil {
					t.Errorf("NewRequestWithContext() error = %v", reqErr)
					return
				}
				// LimitListener releases capacity when the connection closes, so
				// keeping an idle test connection alive would make queued requests
				// wait for the client's diagnostic timeout rather than exercise
				// the next admitted connection.
				request.Close = true
				response, doErr := client.Do(request)
				if doErr != nil {
					t.Errorf("client.Do() error = %v", doErr)
					return
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			})
		}

		// Release the admitted requests only after the listener reaches its cap.
		// The timeout diagnoses a broken accept path; it is not synchronization.
		waittest.ReceiveSignal(t, limitReached, 5*time.Second, "listener to reach its connection limit")
		observedPeak := peak.Load()
		releaseAll()
		wg.Wait()

		if observedPeak > limit {
			t.Fatalf("peak concurrent connections = %d, want <= %d", observedPeak, limit)
		}
		if observedPeak == 0 {
			t.Fatal("no connection was served; the test proved nothing")
		}
	})
}
