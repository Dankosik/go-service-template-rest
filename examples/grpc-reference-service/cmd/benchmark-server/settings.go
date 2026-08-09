package main

import (
	"errors"

	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
)

type benchmarkServerSettings struct {
	transport      grpcx.Config
	maxConnections int
}

// settingsFromDefaults is this example's crossing from loaded configuration to
// the transport adapter's bounds, and the third place that crossing is written
// out: cmd/service/internal/bootstrap.grpcServerConfig is the production one and
// serverConfigFromRuntime in internal/infra/grpc/config_parity_test.go is that
// package's oracle. Neither is importable from here — one is internal to another
// binary, the other is a test — so a bound added to grpcx.Config has to be added
// in all three. TestBenchmarkServerProcessLifecycle asks the target-side
// question that makes a bound missed here fail rather than quietly change what
// the benchmark measures.
func settingsFromDefaults(defaults config.GRPCServerConfig) (benchmarkServerSettings, error) {
	if defaults.MaxConnections <= 0 {
		return benchmarkServerSettings{}, errors.New("build gRPC benchmark server: max connections must be positive")
	}
	// The remaining transport bounds are proven by grpcx.NewServer, which owns
	// the one range check every composition root would otherwise repeat.
	return benchmarkServerSettings{
		transport: grpcx.Config{
			MaxConcurrentRPCs:          defaults.MaxConcurrentRPCs,
			MaxConcurrentHealthRPCs:    defaults.MaxConnections,
			MaxConcurrentStreams:       defaults.MaxConcurrentStreams,
			MaxHeaderListBytes:         defaults.MaxHeaderListBytes,
			MaxReceiveMessageBytes:     defaults.MaxReceiveMessageBytes,
			MaxSendMessageBytes:        defaults.MaxSendMessageBytes,
			AccessLogHealthChecks:      defaults.AccessLogHealthChecks,
			AccessLogSuccessSampleRate: defaults.AccessLogSuccessSampleRate,
			AccessLogSlowThreshold:     defaults.AccessLogSlowThreshold,
			TelemetryHealthChecks:      defaults.TelemetryHealthChecks,
			UnaryTimeout:               defaults.UnaryTimeout,
			StreamTimeout:              defaults.StreamTimeout,
			MaxConnectionIdle:          defaults.MaxConnectionIdle,
			ServerPingInterval:         defaults.ServerPingInterval,
			ServerPingTimeout:          defaults.ServerPingTimeout,
			MinClientPingInterval:      defaults.MinClientPingInterval,
			PermitPingWithoutStream:    defaults.PermitPingWithoutStream,
			MaxConnectionAge:           defaults.MaxConnectionAge,
			MaxConnectionAgeGrace:      defaults.MaxConnectionAgeGrace,
		},
		maxConnections: defaults.MaxConnections,
	}, nil
}
