package grpclimits

import (
	"math"
	"testing"
	"time"
)

func TestAccessLogValidationRejectsUnsafeSampling(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value AccessLog
		field string
	}{
		{name: "nan", value: AccessLog{SuccessSampleRate: math.NaN(), SlowThreshold: time.Second}, field: "access_log_success_sample_rate"},
		{name: "infinite", value: AccessLog{SuccessSampleRate: math.Inf(1), SlowThreshold: time.Second}, field: "access_log_success_sample_rate"},
		{name: "negative rate", value: AccessLog{SuccessSampleRate: -0.1, SlowThreshold: time.Second}, field: "access_log_success_sample_rate"},
		{name: "rate above one", value: AccessLog{SuccessSampleRate: 1.1, SlowThreshold: time.Second}, field: "access_log_success_sample_rate"},
		{name: "negative slow threshold", value: AccessLog{SuccessSampleRate: 0.5, SlowThreshold: -time.Second}, field: "access_log_slow_threshold"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violation := ValidateAccessLog(tc.value)
			if violation == nil || violation.Field != tc.field {
				t.Fatalf("ValidateAccessLog() = %#v, want field %q", violation, tc.field)
			}
		})
	}
	if violation := ValidateAccessLog(AccessLog{SuccessSampleRate: 0.5}); violation != nil {
		t.Fatalf("ValidateAccessLog(valid) = %#v, want nil", violation)
	}
}

func TestLifetimeValidationPreservesRotationBounds(t *testing.T) {
	t.Parallel()

	valid := Lifetime{
		UnaryTimeout: time.Second, StreamTimeout: time.Second, MaxConnectionIdle: time.Second,
		MaxConnectionAge: 5 * time.Second, MaxConnectionAgeGrace: 2 * time.Second,
		ServerPingInterval: time.Second, ServerPingTimeout: time.Second, MinClientPingInterval: time.Second,
	}
	for _, tc := range []struct {
		name  string
		set   func(*Lifetime)
		field string
	}{
		{name: "negative unary timeout", set: func(cfg *Lifetime) { cfg.UnaryTimeout = -time.Second }, field: "unary_timeout"},
		{name: "idle must be positive", set: func(cfg *Lifetime) { cfg.MaxConnectionIdle = 0 }, field: "max_connection_idle"},
		{name: "missing rotation grace", set: func(cfg *Lifetime) { cfg.MaxConnectionAgeGrace = 0 }, field: "max_connection_age_grace"},
		{name: "grace below unary timeout", set: func(cfg *Lifetime) { cfg.MaxConnectionAgeGrace = time.Nanosecond }, field: "max_connection_age_grace"},
		{name: "stream can outlive rotation", set: func(cfg *Lifetime) { cfg.StreamTimeout = cfg.MaxConnectionAge }, field: "stream_timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			tc.set(&cfg)
			violation := ValidateLifetime(cfg)
			if violation == nil || violation.Field != tc.field {
				t.Fatalf("ValidateLifetime() = %#v, want field %q", violation, tc.field)
			}
		})
	}
	if violation := ValidateLifetime(valid); violation != nil {
		t.Fatalf("ValidateLifetime(valid) = %#v, want nil", violation)
	}
	if got := (Violation{Field: "max_connection_age"}).FieldWords(); got != "max connection age" {
		t.Fatalf("FieldWords() = %q, want rendered field", got)
	}
}
