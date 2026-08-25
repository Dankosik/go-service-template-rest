package config

import (
	"fmt"
	"strings"
	"time"
)

type HTTPConfig struct {
	Addr string `koanf:"addr"`
	// GracePeriod is the total time the platform allows between SIGTERM and
	// SIGKILL — terminationGracePeriodSeconds on Kubernetes, drainingSeconds on
	// Railway. Every teardown stage draws from it, so it is the one number that
	// bounds shutdown as a whole. ShutdownTimeout bounds only the HTTP drain;
	// closing diagnostics, joining background work, releasing pooled
	// dependencies, and flushing telemetry all still run after it.
	GracePeriod time.Duration `koanf:"grace_period"`
	// ShutdownTimeout bounds the HTTP drain, including the readiness
	// propagation delay in front of it. It is validated against GracePeriod so
	// the stages after the drain still have budget left.
	ShutdownTimeout           time.Duration `koanf:"shutdown_timeout"`
	ReadinessTimeout          time.Duration `koanf:"readiness_timeout"`
	ReadinessPropagationDelay time.Duration `koanf:"readiness_propagation_delay"`
	ReadHeaderTimeout         time.Duration `koanf:"read_header_timeout"`
	ReadTimeout               time.Duration `koanf:"read_timeout"`
	// RequestTimeout is the per-request handler budget. The net/http server
	// timeouts are connection deadlines that never cancel the request context,
	// so this is the only bound on how long one handler may hold a goroutine
	// and its pooled resources.
	RequestTimeout time.Duration `koanf:"request_timeout"`
	WriteTimeout   time.Duration `koanf:"write_timeout"`
	IdleTimeout    time.Duration `koanf:"idle_timeout"`
	MaxHeaderBytes int           `koanf:"max_header_bytes"`
	MaxBodyBytes   int64         `koanf:"max_body_bytes"`
	// MaxInFlight bounds concurrent handler execution. Nothing else in the
	// request path does: RequestTimeout bounds how long one request may run,
	// not how many may run, so an arrival spike otherwise queues every request
	// on a finite resource until all of them spend the full budget waiting.
	// Zero disables shedding.
	MaxInFlight int `koanf:"max_in_flight"`
	// MaxConnections bounds how many connections the API listener accepts at
	// once. MaxInFlight does not: it bounds handler execution, and every
	// accepted connection already costs a goroutine and its read and write
	// buffers before any middleware runs. Excess callers wait in the kernel
	// backlog, where they cost nothing. Validated against MaxInFlight so the cap
	// can never shed below the concurrency the service advertises. Zero accepts
	// without a bound.
	MaxConnections int `koanf:"max_connections"`
	// AccessLogHealthProbes re-enables access logging for /health/live and
	// /health/ready. It defaults to false because orchestrator probes generate
	// continuous no-signal log volume.
	AccessLogHealthProbes bool `koanf:"access_log_health_probes"`
}

// httpTerminalResponseReserve leaves the connection write deadline enough time
// to carry the small Problem response emitted after a request budget expires.
const httpTerminalResponseReserve = time.Second

func httpDefaults() map[string]any {
	return map[string]any{
		"http.addr": ":8080",
		// grace_period matches railway.toml's drainingSeconds, the platform this
		// repository ships a deployment profile for. Kubernetes grants 30s by
		// default, so a deployment there sets both this and shutdown_timeout;
		// leaving them mismatched is what validateShutdownGraceBudget rejects at
		// startup rather than letting a SIGKILL discover it.
		"http.grace_period": "45s",
		// shutdown_timeout bounds the drain only. It is 25s rather than the whole
		// grace period because the teardown after the drain — diagnostics,
		// background join, pool release, telemetry flush — needs the remaining 17s,
		// and the flush that records how all three went is what runs last.
		"http.shutdown_timeout":  "25s",
		"http.readiness_timeout": "4s",
		// readiness_propagation_delay holds the drain open long enough for a load
		// balancer to notice /health/ready failing before connections stop being
		// accepted. It is production-shaped on purpose, so running the binary
		// directly waits it out on SIGTERM; env/.env.example sets 0s for local
		// iteration, and docs/railway-deployment-profile.md does the arithmetic
		// against the platform's draining window.
		"http.readiness_propagation_delay": "15s",
		"http.read_header_timeout":         "5s",
		"http.read_timeout":                "5s",
		// request_timeout stays below write_timeout so the budget expires while
		// the connection can still carry the 504 that reports it.
		"http.request_timeout":  "8s",
		"http.write_timeout":    "10s",
		"http.idle_timeout":     "60s",
		"http.max_header_bytes": 16 << 10,
		"http.max_body_bytes":   int64(1 << 20),
		// max_in_flight is deliberately larger than postgres.max_open_conns:
		// shedding must engage after the pool is saturated and requests start
		// queueing, not before it is used.
		"http.max_in_flight": 256,
		// max_connections is well above max_in_flight, because the two bound
		// different things: shedding answers 503 with a Retry-After, while this
		// leaves a caller in the kernel backlog with no answer at all. The
		// headroom is what keeps the informative rejection the common one, and
		// this the backstop against a connection flood costing a goroutine and
		// two buffers apiece until the process is OOM-killed.
		"http.max_connections":          4096,
		"http.access_log_health_probes": false,
	}
}

func validateHTTPConfig(cfg *HTTPConfig) error {
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	if cfg.Addr == "" {
		return fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("http.grace_period", cfg.GracePeriod, time.Second, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.shutdown_timeout", cfg.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return err
	}
	// The drain is one stage inside the grace period, never the whole of it. How
	// much the stages after it need is owned by the composition root, which knows
	// their ceilings; see validateShutdownGraceBudget.
	if cfg.ShutdownTimeout > cfg.GracePeriod {
		return fmt.Errorf(
			"%w: http.shutdown_timeout must be <= http.grace_period (%s)",
			ErrValidate,
			cfg.GracePeriod,
		)
	}
	if err := validateDurationRange("http.readiness_timeout", cfg.ReadinessTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("http.readiness_propagation_delay", cfg.ReadinessPropagationDelay, 0, cfg.ShutdownTimeout); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_header_timeout", cfg.ReadHeaderTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_timeout", cfg.ReadTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.request_timeout", cfg.RequestTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.write_timeout", cfg.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return err
	}
	if err := validateHTTPReadinessWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPRequestWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPShutdownBudget(*cfg); err != nil {
		return err
	}
	if err := validateHTTPCapacityBounds(*cfg); err != nil {
		return err
	}
	return nil
}

// validateHTTPCapacityBounds owns the four settings that decide how much work
// and how many callers one process admits. They are grouped because they are
// read together — see validateHTTPConnectionCeiling for the one relationship
// between them that a range check cannot express.
func validateHTTPCapacityBounds(cfg HTTPConfig) error {
	if cfg.MaxHeaderBytes <= 0 {
		return fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	if err := validateIntRange("http.max_in_flight", cfg.MaxInFlight, 0, 100_000); err != nil {
		return err
	}
	if err := validateIntRange("http.max_connections", cfg.MaxConnections, 0, 1_000_000); err != nil {
		return err
	}
	return validateHTTPConnectionCeiling(cfg)
}

// validateHTTPConnectionCeiling keeps the accept cap above the concurrency the
// service advertises.
//
// The two bound different things and only make sense together. max_in_flight is
// how many requests may execute a handler; max_connections is how many callers
// may be attached at all. Setting the second below the first means the service
// cannot reach its own concurrency ceiling, and the excess callers do not get
// the 503 with a Retry-After that shedding would have given them — they wait in
// the kernel backlog and time out at connect, which is a worse signal for a
// client and an invisible one for an operator.
func validateHTTPConnectionCeiling(cfg HTTPConfig) error {
	if cfg.MaxConnections == 0 || cfg.MaxConnections >= cfg.MaxInFlight {
		return nil
	}
	return fmt.Errorf(
		"%w: http.max_connections must be >= http.max_in_flight (%d)",
		ErrValidate,
		cfg.MaxInFlight,
	)
}

func validateHTTPShutdownBudget(cfg HTTPConfig) error {
	effectiveDrainBudget := cfg.ShutdownTimeout - cfg.ReadinessPropagationDelay
	if effectiveDrainBudget <= 0 {
		return fmt.Errorf("%w: http.readiness_propagation_delay must be less than http.shutdown_timeout", ErrValidate)
	}
	if cfg.WriteTimeout > effectiveDrainBudget {
		return fmt.Errorf(
			"%w: http.write_timeout must be <= effective drain budget after readiness propagation (%s)",
			ErrValidate,
			effectiveDrainBudget,
		)
	}
	return nil
}

func validateHTTPReadinessWriteTimeout(cfg HTTPConfig) error {
	if cfg.ReadinessTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.readiness_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}

// validateHTTPRequestWriteTimeout keeps the handler budget plus the terminal
// response reserve inside the connection write deadline.
func validateHTTPRequestWriteTimeout(cfg HTTPConfig) error {
	if cfg.RequestTimeout+httpTerminalResponseReserve > cfg.WriteTimeout {
		return fmt.Errorf(
			"%w: http.request_timeout must leave at least %s before http.write_timeout for the terminal response",
			ErrValidate,
			httpTerminalResponseReserve,
		)
	}
	return nil
}
