package config

import (
	"fmt"
	"strings"
	"time"
)

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

// validateHTTPRequestWriteTimeout keeps the handler budget inside the response
// write deadline. A request budget larger than http.write_timeout expires only
// after the connection can no longer carry a response, so the timeout would be
// reported to the client as a dropped connection instead of a 504.
func validateHTTPRequestWriteTimeout(cfg HTTPConfig) error {
	if cfg.RequestTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.request_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}
