// Package grpclimits owns the gRPC server bounds that internal/config and
// internal/infra/grpc must answer identically.
//
// It is a leaf for the reason internal/authntrust is: the depguard rule
// config_no_runtime_owners stops internal/config from importing the adapter, and
// both owners must refuse the same value — the loader so the process stops, the
// adapter so a server built any other way is refused too. Restating the rules in
// both is what let one side be tightened alone, breaking nothing anyone ran.
//
// Only the rules that are genuinely one policy live here. Capacity bounds do not:
// internal/config is deliberately the tighter owner there, and
// internal/infra/grpc/config_parity_test.go proves the containment that makes the
// difference safe rather than checking the two for equality.
//
// A rule returns a [Violation] rather than an error because the two owners answer
// different audiences. internal/config names the configuration leaf an operator
// edits; the adapter names the same bound in prose, because a direct package user
// never saw a configuration file.
package grpclimits

// profile:grpc:start

import (
	"math"
	"strings"
	"time"
)

// Violation is one broken bound: the configuration leaf that broke it and the
// rule it broke, worded to follow the rendered field name.
type Violation struct {
	Field string
	Rule  string
}

// FieldWords renders Field the way a reader with no configuration file sees it.
func (v Violation) FieldWords() string {
	return strings.ReplaceAll(v.Field, "_", " ")
}

// AccessLog is the access-log policy both owners bound identically.
type AccessLog struct {
	SuccessSampleRate float64
	SlowThreshold     time.Duration
}

// ValidateAccessLog reports the first broken access-log rule, or nil.
//
// The rate is refused rather than clamped: a NaN read as "sample nothing" would
// silently turn the success log off for the life of a deployment.
func ValidateAccessLog(cfg AccessLog) *Violation {
	if math.IsNaN(cfg.SuccessSampleRate) ||
		math.IsInf(cfg.SuccessSampleRate, 0) ||
		cfg.SuccessSampleRate < 0 ||
		cfg.SuccessSampleRate > 1 {
		return &Violation{Field: "access_log_success_sample_rate", Rule: "must be finite and in range [0,1]"}
	}
	if cfg.SlowThreshold < 0 {
		return &Violation{Field: "access_log_slow_threshold", Rule: "must be non-negative"}
	}
	return nil
}

// Lifetime is the connection and call time policy both owners bound identically.
type Lifetime struct {
	UnaryTimeout          time.Duration
	StreamTimeout         time.Duration
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	ServerPingInterval    time.Duration
	ServerPingTimeout     time.Duration
	MinClientPingInterval time.Duration
}

// ValidateLifetime reports the first broken time bound, or nil.
//
// Every duration is non-negative because a negative one is not a third meaning:
// grpc-go normalizes only zero to infinity, so a negative maximum age arms its
// timer immediately and rotates every connection at once — the opposite of the
// off-by-default this repository chose. The three conditional rules exist only
// when rotation is on, which is why they are checked after the flat bounds.
//
// The flat bounds are slices rather than maps so that a configuration breaking
// more than one of them is refused with the same message every time. Map
// iteration would pick one at random, and an operator would see a different
// reason on each restart of the same deployment.
func ValidateLifetime(cfg Lifetime) *Violation {
	nonNegative := []bound{
		{field: "unary_timeout", value: cfg.UnaryTimeout},
		{field: "stream_timeout", value: cfg.StreamTimeout},
		{field: "max_connection_age", value: cfg.MaxConnectionAge},
		{field: "max_connection_age_grace", value: cfg.MaxConnectionAgeGrace},
	}
	for _, bound := range nonNegative {
		if bound.value < 0 {
			return &Violation{Field: bound.field, Rule: "must be non-negative"}
		}
	}
	positive := []bound{
		{field: "max_connection_idle", value: cfg.MaxConnectionIdle},
		{field: "server_ping_interval", value: cfg.ServerPingInterval},
		{field: "server_ping_timeout", value: cfg.ServerPingTimeout},
		{field: "min_client_ping_interval", value: cfg.MinClientPingInterval},
	}
	for _, bound := range positive {
		if bound.value <= 0 {
			return &Violation{Field: bound.field, Rule: "must be positive"}
		}
	}

	if cfg.MaxConnectionAge == 0 {
		return nil
	}
	if cfg.MaxConnectionAgeGrace <= 0 {
		return &Violation{
			Field: "max_connection_age_grace",
			Rule:  "must be positive when a max connection age is set",
		}
	}
	if cfg.MaxConnectionAgeGrace < cfg.UnaryTimeout {
		return &Violation{
			Field: "max_connection_age_grace",
			Rule:  "must be at least the unary timeout",
		}
	}
	if cfg.StreamTimeout > 0 && cfg.StreamTimeout >= cfg.MaxConnectionAge {
		return &Violation{
			Field: "stream_timeout",
			Rule:  "must be below the max connection age, or rotation decides first",
		}
	}
	return nil
}

// bound is one named time bound, in the order [ValidateLifetime] checks them.
type bound struct {
	field string
	value time.Duration
}

// profile:grpc:end
