package grpcx

import (
	"errors"
	"fmt"
	"math"

	"github.com/example/go-service-template-rest/internal/grpclimits"
)

// validateConfig proves the bounds NewServer is about to hand grpc-go.
//
// The access-log and lifetime rules come from internal/grpclimits, which
// internal/config applies to the same values on the way in; this file owns only
// the capacity bounds, where the two owners are deliberately different.
// config_parity_test.go proves that what the loader accepts still builds a
// server.
func validateConfig(cfg Config) error {
	if cfg.MaxConcurrentRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent RPCs must be positive")
	}
	if cfg.MaxConcurrentHealthRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent health RPCs must be positive")
	}
	if err := validateUint32Bound("max concurrent streams", cfg.MaxConcurrentStreams); err != nil {
		return err
	}
	if err := validateUint32Bound("max header list bytes", cfg.MaxHeaderListBytes); err != nil {
		return err
	}
	if cfg.MaxReceiveMessageBytes <= 0 {
		return errors.New("build gRPC server: max receive message bytes must be positive")
	}
	if cfg.MaxSendMessageBytes <= 0 {
		return errors.New("build gRPC server: max send message bytes must be positive")
	}
	if violation := grpclimits.ValidateAccessLog(grpclimits.AccessLog{
		SuccessSampleRate: cfg.AccessLogSuccessSampleRate,
		SlowThreshold:     cfg.AccessLogSlowThreshold,
	}); violation != nil {
		return refuse(violation)
	}
	if violation := grpclimits.ValidateLifetime(grpclimits.Lifetime{
		UnaryTimeout:          cfg.UnaryTimeout,
		StreamTimeout:         cfg.StreamTimeout,
		MaxConnectionIdle:     cfg.MaxConnectionIdle,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
		ServerPingInterval:    cfg.ServerPingInterval,
		ServerPingTimeout:     cfg.ServerPingTimeout,
		MinClientPingInterval: cfg.MinClientPingInterval,
	}); violation != nil {
		return refuse(violation)
	}
	return nil
}

// refuse renders a shared bound violation for a direct package user, who never
// saw a configuration file and so is told the bound in prose rather than by its
// configuration leaf.
func refuse(violation *grpclimits.Violation) error {
	return fmt.Errorf("build gRPC server: %s %s", violation.FieldWords(), violation.Rule)
}

// validateUint32Bound owns the one place a caller-supplied transport bound is
// proven to fit the uint32 grpc-go asks for; [Config] owns why those fields are
// int.
func validateUint32Bound(name string, value int) error {
	if value <= 0 || uint64(value) > math.MaxUint32 {
		return fmt.Errorf(
			"build gRPC server: %s must be in range [1,%d]",
			name,
			uint64(math.MaxUint32),
		)
	}
	return nil
}
