package bootstrap

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"math"
	"runtime/debug"
	"strings"
	"testing"
	"testing/fstest"
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
