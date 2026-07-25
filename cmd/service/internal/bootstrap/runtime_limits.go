package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
)

// GOMAXPROCS is deliberately absent from this file. Since Go 1.25 the runtime
// derives it from the cgroup CPU bandwidth limit on Linux and re-reads it
// periodically as that limit changes; setting it here would switch both
// behaviors off. Memory is the half the toolchain still does not cover: the GC's
// soft limit defaults to math.MaxInt64, so a container is OOM-killed by the
// kernel rather than collected harder by its own runtime.
const (
	// cgroup v2 exposes one file holding either a byte count or "max".
	cgroupV2MemoryLimitPath = "/sys/fs/cgroup/memory.max"

	// cgroup v1 exposes a byte count, using a sentinel near the word size to
	// mean unlimited.
	cgroupV1MemoryLimitPath = "/sys/fs/cgroup/memory/memory.limit_in_bytes"

	cgroupUnlimitedLiteral = "max"

	// memoryLimitEnv is the runtime's own variable. A platform that already set
	// it owns the decision, and this must not override it.
	memoryLimitEnv = "GOMEMLIMIT"

	// cgroupUnlimitedFloor rejects the v1 unlimited sentinel, which is a value
	// close to the maximum int64 rather than a distinct literal. No real
	// container limit approaches it.
	cgroupUnlimitedFloor int64 = 1 << 62
)

// errNoMemoryLimit reports that no container memory limit was found. Running
// outside a limited cgroup is the normal local case, not a failure.
var errNoMemoryLimit = errors.New("no cgroup memory limit found")

// applyMemoryLimit publishes a fraction of the container memory limit to the
// garbage collector, and reports what it decided so the choice is visible in the
// startup log rather than inferred from an exit code.
//
// ratio <= 0 disables detection.
func applyMemoryLimit(log *slog.Logger, ratio float64) {
	if ratio <= 0 {
		log.Info(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "detection_disabled",
		)
		return
	}
	if raw, ok := os.LookupEnv(memoryLimitEnv); ok && strings.TrimSpace(raw) != "" {
		log.Info(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "already_set_by_platform",
			"env", memoryLimitEnv,
		)
		return
	}

	limit, err := detectCgroupMemoryLimit(os.DirFS("/"))
	if err != nil {
		// Finding no limit is the ordinary case outside a container; a malformed
		// limit is an operator-visible surprise and reports louder.
		level := slog.LevelInfo
		if !errors.Is(err, errNoMemoryLimit) {
			level = slog.LevelWarn
		}
		log.Log(
			context.Background(),
			level,
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "no_container_limit",
			"err", err,
		)
		return
	}

	applied := int64(float64(limit) * ratio)
	if applied <= 0 {
		log.Warn(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "ratio_rounds_to_zero",
			"limit.bytes", limit,
			"ratio", ratio,
		)
		return
	}

	debug.SetMemoryLimit(applied)
	log.Info(
		"runtime_memory_limit_applied",
		"component", "runtime_limits",
		"limit.bytes", limit,
		"ratio", ratio,
		"gomemlimit.bytes", applied,
	)
}

// detectCgroupMemoryLimit reads the container memory limit from root. It takes an
// fs.FS so the parsing is testable against fixtures rather than the host's own
// cgroup layout, which differs between developer machines and CI.
func detectCgroupMemoryLimit(root fs.FS) (int64, error) {
	for _, candidate := range []string{cgroupV2MemoryLimitPath, cgroupV1MemoryLimitPath} {
		limit, err := readCgroupMemoryLimit(root, candidate)
		if err == nil {
			return limit, nil
		}
		if !errors.Is(err, errNoMemoryLimit) && !errors.Is(err, fs.ErrNotExist) {
			return 0, err
		}
	}
	return 0, errNoMemoryLimit
}

func readCgroupMemoryLimit(root fs.FS, name string) (int64, error) {
	// fs.FS paths are always unrooted, unlike the absolute paths the constants
	// carry for readability at the point they are documented.
	raw, err := fs.ReadFile(root, path.Clean(strings.TrimPrefix(name, "/")))
	if err != nil {
		// The path is preserved so a caller can distinguish "no cgroup here" from
		// an unreadable one, and detectCgroupMemoryLimit matches on fs.ErrNotExist.
		return 0, fmt.Errorf("read cgroup memory limit %q: %w", name, err)
	}

	value := strings.TrimSpace(string(raw))
	if value == "" || value == cgroupUnlimitedLiteral {
		return 0, errNoMemoryLimit
	}
	limit, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse cgroup memory limit %q: invalid value", name)
	}
	if limit <= 0 || limit >= cgroupUnlimitedFloor {
		return 0, errNoMemoryLimit
	}
	return limit, nil
}
