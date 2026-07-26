package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestDetectCgroupMemoryLimit(t *testing.T) {
	t.Parallel()

	const v2 = "sys/fs/cgroup/memory.max"
	const v1 = "sys/fs/cgroup/memory/memory.limit_in_bytes"

	testCases := []struct {
		name    string
		files   fstest.MapFS
		want    int64
		wantErr error
	}{
		{
			name:  "cgroup v2 byte count",
			files: fstest.MapFS{v2: &fstest.MapFile{Data: []byte("536870912\n")}},
			want:  536870912,
		},
		{
			name:    "cgroup v2 unlimited literal falls through",
			files:   fstest.MapFS{v2: &fstest.MapFile{Data: []byte("max\n")}},
			wantErr: errNoMemoryLimit,
		},
		{
			name: "cgroup v2 unlimited falls back to v1",
			files: fstest.MapFS{
				v2: &fstest.MapFile{Data: []byte("max\n")},
				v1: &fstest.MapFile{Data: []byte("268435456")},
			},
			want: 268435456,
		},
		{
			name:  "cgroup v1 byte count",
			files: fstest.MapFS{v1: &fstest.MapFile{Data: []byte("268435456")}},
			want:  268435456,
		},
		{
			// cgroup v1 signals "unlimited" with a value near the word size
			// rather than a distinct literal, so a naive parse would hand the GC
			// a limit of ~8 exabytes and look like it worked.
			name:    "cgroup v1 unlimited sentinel is rejected",
			files:   fstest.MapFS{v1: &fstest.MapFile{Data: []byte("9223372036854771712")}},
			wantErr: errNoMemoryLimit,
		},
		{
			name:    "no cgroup files",
			files:   fstest.MapFS{},
			wantErr: errNoMemoryLimit,
		},
		{
			name:    "empty file",
			files:   fstest.MapFS{v2: &fstest.MapFile{Data: []byte("  \n")}},
			wantErr: errNoMemoryLimit,
		},
		{
			name:    "zero is not a usable limit",
			files:   fstest.MapFS{v2: &fstest.MapFile{Data: []byte("0")}},
			wantErr: errNoMemoryLimit,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectCgroupMemoryLimit(tc.files)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("detectCgroupMemoryLimit() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("detectCgroupMemoryLimit() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("detectCgroupMemoryLimit() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestDetectCgroupMemoryLimitReportsMalformedValues keeps a corrupt limit from
// being reported as "no container limit", which would hide a real misconfiguration.
func TestDetectCgroupMemoryLimitReportsMalformedValues(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"sys/fs/cgroup/memory.max": &fstest.MapFile{Data: []byte("not-a-number")}}

	_, err := detectCgroupMemoryLimit(files)
	if err == nil {
		t.Fatal("detectCgroupMemoryLimit() error = nil, want a parse failure")
	}
	if errors.Is(err, errNoMemoryLimit) || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("detectCgroupMemoryLimit() error = %v, want it distinguished from a missing limit", err)
	}
	// The value can carry deployment detail, so the message names the file only.
	if strings.Contains(err.Error(), "not-a-number") {
		t.Fatalf("error = %q, want the raw value withheld", err.Error())
	}
}

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

	var logged bytes.Buffer
	// The host is unlikely to be in a limited cgroup, so this exercises the
	// reporting path rather than the applied one. Both must say what happened.
	applyMemoryLimit(slog.New(slog.NewJSONHandler(&logged, nil)), 0.9)

	line := logged.String()
	if !strings.Contains(line, "runtime_memory_limit_applied") && !strings.Contains(line, "runtime_memory_limit_skipped") {
		t.Fatalf("log = %q, want an applied or skipped record", line)
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
			// The shipped defaults: 256 x 1 MiB x 2 copies is 512 MiB against a
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
				<-release
				concurrent.Add(-1)
				w.WriteHeader(http.StatusOK)
			}),
		}
		go func() { _ = server.Serve(listener) }()
		t.Cleanup(func() { _ = server.Close() })

		var wg sync.WaitGroup
		for range limit * 3 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := &http.Client{Timeout: 5 * time.Second}
				request, reqErr := http.NewRequestWithContext(
					context.Background(), http.MethodGet, "http://"+base.Addr().String()+"/", nil)
				if reqErr != nil {
					return
				}
				response, doErr := client.Do(request)
				if doErr != nil {
					return
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			}()
		}

		// Give the excess callers time to reach the accept queue, then let the
		// admitted ones finish. Without the cap, every one of them would have
		// been accepted and be sitting in the handler by now.
		time.Sleep(200 * time.Millisecond)
		observedPeak := peak.Load()
		close(release)
		wg.Wait()

		if observedPeak > limit {
			t.Fatalf("peak concurrent connections = %d, want <= %d", observedPeak, limit)
		}
		if observedPeak == 0 {
			t.Fatal("no connection was served; the test proved nothing")
		}
	})
}
