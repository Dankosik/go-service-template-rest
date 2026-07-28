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
	"github.com/example/go-service-template-rest/internal/config"
)

// GOMAXPROCS is deliberately absent from this file. Since Go 1.25 the runtime
// derives it from the cgroup CPU bandwidth limit on Linux and re-reads it
// periodically as that limit changes; setting it here would switch both
// behaviors off. Memory is the half the toolchain still does not cover: the GC's
// soft limit defaults to math.MaxInt64, so a container is OOM-killed by the
// kernel rather than collected harder by its own runtime.
const (
	// memoryLimitEnv is the runtime's own variable. A platform that already set
	// it owns the decision, and this must not override it.
	memoryLimitEnv = "GOMEMLIMIT"
)

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

// requestBufferCopiesPerRequest is how many full copies of one request body the
// request path can hold at once.
//
// One: the OpenAPI validator reads the body and puts back a copy.
const requestBufferCopiesPerRequest = 1

// requestBufferBudgetRatio is the share of the GC's own limit that admitted
// request bodies may account for before the arithmetic is worth reporting.
//
// It is a fraction rather than the whole limit because the buffers are not what
// the process is for: the same heap holds the connection pool, the metric
// registry, and whatever the handlers allocate on top of the body they were
// given.
const requestBufferBudgetRatio = 0.25

// reportRequestBufferBudget relates http.max_in_flight and http.max_body_bytes
// to the memory limit the GC was just given.
//
// Nothing else does. The two settings are validated independently — one must be
// positive, the other must fit a range — so their product is a number no layer
// ever computes, and it is the number that decides whether the process survives
// its own concurrency. The shipped defaults are modest; raising
// http.max_body_bytes for file uploads is a one-line change that multiplies it
// by whatever the new limit is, and the first evidence of that is the GC burning
// CPU against a limit live data has already passed, then an OOM kill. Readiness
// reports healthy throughout, because a cached dependency probe still succeeds.
//
// It reports rather than rejects. A service whose handlers stream instead of
// buffering is legitimately over the estimate, and refusing to start would be
// wrong for it; a number in the startup log is what the other case needs.
//
// limit is the detected container limit, and zero — no cgroup limit, or
// detection disabled — skips the check, because there is nothing to relate to.
func reportRequestBufferBudget(log *slog.Logger, cfg config.Config, limit int64) {
	if limit <= 0 || cfg.Runtime.MemoryLimitRatio <= 0 {
		return
	}
	// Zero disables shedding, which means the ceiling is however many requests
	// arrive at once. That is unbounded, so there is no product to compare.
	if cfg.HTTP.MaxInFlight <= 0 {
		return
	}

	worstCase := int64(cfg.HTTP.MaxInFlight) * cfg.HTTP.MaxBodyBytes * requestBufferCopiesPerRequest
	budget := int64(float64(limit) * cfg.Runtime.MemoryLimitRatio * requestBufferBudgetRatio)
	if worstCase <= budget {
		return
	}

	log.Warn(
		"runtime_request_buffer_budget_exceeded",
		"component", "runtime_limits",
		"reason", "max_in_flight_times_max_body_bytes_exceeds_memory_budget",
		"http.max_in_flight", cfg.HTTP.MaxInFlight,
		"http.max_body_bytes", cfg.HTTP.MaxBodyBytes,
		"request_buffers.worst_case_bytes", worstCase,
		"request_buffers.budget_bytes", budget,
		"limit.bytes", limit,
	)
}
