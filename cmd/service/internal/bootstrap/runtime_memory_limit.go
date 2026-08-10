package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"strings"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

// GOMAXPROCS is deliberately absent from this file. Since Go 1.25 the runtime
// derives it from the cgroup CPU bandwidth limit on Linux and re-reads it
// periodically as that limit changes; setting it here would switch both
// behaviors off. Memory is the half the toolchain still does not cover: the GC's
// soft limit defaults to math.MaxInt64, so a container is OOM-killed by the
// kernel rather than collected harder by its own runtime.
const memoryLimitEnv = "GOMEMLIMIT"

// applyMemoryLimit publishes a fraction of the container memory limit to the
// garbage collector, and reports what it decided so the choice is visible in the
// startup log rather than inferred from an exit code.
//
// It returns the detected container limit, or zero when there is none, so the
// caller can relate the process's other byte budgets to the same number.
//
// ratio <= 0 disables detection.
func applyMemoryLimit(log *slog.Logger, ratio float64) int64 {
	return applyMemoryLimitFrom(log, ratio, memlimit.FromCgroup)
}

func applyMemoryLimitFrom(log *slog.Logger, ratio float64, detect func() (uint64, error)) int64 {
	if ratio <= 0 {
		log.Info(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "detection_disabled",
		)
		return 0
	}
	if raw, ok := os.LookupEnv(memoryLimitEnv); ok && strings.TrimSpace(raw) != "" {
		log.Info(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "already_set_by_platform",
			"env", memoryLimitEnv,
		)
		return 0
	}

	detected, err := detect()
	if err != nil {
		// Finding no limit is the ordinary case outside a container; a malformed
		// limit is an operator-visible surprise and reports louder.
		level := slog.LevelInfo
		if !errors.Is(err, memlimit.ErrNoLimit) &&
			!errors.Is(err, memlimit.ErrNoCgroup) &&
			!errors.Is(err, memlimit.ErrCgroupsNotSupported) {
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
		return 0
	}
	if detected == 0 || detected > math.MaxInt64 {
		log.Warn(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "unusable_container_limit",
			"limit.bytes", detected,
		)
		return 0
	}
	limit := int64(detected)

	applied := int64(float64(limit) * ratio)
	if applied <= 0 {
		log.Warn(
			"runtime_memory_limit_skipped",
			"component", "runtime_limits",
			"reason", "ratio_rounds_to_zero",
			"limit.bytes", limit,
			"ratio", ratio,
		)
		return 0
	}

	debug.SetMemoryLimit(applied)
	log.Info(
		"runtime_memory_limit_applied",
		"component", "runtime_limits",
		"limit.bytes", limit,
		"ratio", ratio,
		"gomemlimit.bytes", applied,
	)
	return limit
}
