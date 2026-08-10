package bootstrap

import (
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
)

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
// Nothing else does. The two are validated independently, so their product is a
// number no layer computes — and it is the number that decides whether the
// process survives its own concurrency. Raising http.max_body_bytes for file
// uploads is a one-line change that multiplies it, and the first evidence is the
// GC burning CPU against a limit live data has already passed, then an OOM kill,
// with readiness healthy throughout because a cached dependency probe still
// succeeds.
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
