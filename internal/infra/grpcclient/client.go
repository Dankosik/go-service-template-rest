package grpcclient

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	maxHeaderListBytes     = 16 << 10
	maxReceiveMessageBytes = 4 << 20
	maxSendMessageBytes    = 4 << 20
)

// Options supplies the trust decision and the observability collaborators the
// composition root owns.
type Options struct {
	// TransportCredentials is required, and [New] refuses a nil value. Plaintext
	// is spelled insecure.NewCredentials() so that dialing without transport
	// security is a visible decision, as grpc.NewClient itself requires. The
	// server adapter's field of the same name is optional because it binds a
	// listener the deployment owns; this one chooses how far to trust a peer it
	// is about to send credentials to.
	TransportCredentials credentials.TransportCredentials
	// PerRPCCredentials optionally supplies one connection credential for both
	// application RPCs and any grpc-go control streams explicitly enabled by the
	// dependency owner.
	PerRPCCredentials credentials.PerRPCCredentials

	// Optional observability collaborators. Both fall back to their no-op
	// implementations, so a test can leave them unset.
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider
}

// New constructs a shareable ClientConn without performing network I/O. The
// caller owns Close and each operation owns its context deadline and retry
// semantics.
func New(target string, options Options) (*grpc.ClientConn, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, errors.New("build gRPC client: target is required")
	}
	if options.TransportCredentials == nil {
		return nil, errors.New("build gRPC client: transport credentials are required")
	}
	if options.MeterProvider == nil {
		options.MeterProvider = metricnoop.NewMeterProvider()
	}
	if options.TracerProvider == nil {
		options.TracerProvider = tracenoop.NewTracerProvider()
	}

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(options.TransportCredentials),
		// A dependency must explicitly reopen retry, health, or load-balancing
		// policy. Resolver addresses and reconnects remain native grpc-go behavior.
		grpc.WithDisableServiceConfig(),
		grpc.WithMaxHeaderListSize(maxHeaderListBytes),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxReceiveMessageBytes),
			grpc.MaxCallSendMsgSize(maxSendMessageBytes),
		),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithMeterProvider(options.MeterProvider),
			otelgrpc.WithTracerProvider(options.TracerProvider),
			otelgrpc.WithPropagators(propagation.TraceContext{}),
		)),
	}
	if options.PerRPCCredentials != nil {
		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(options.PerRPCCredentials))
	}

	connection, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("build gRPC client connection: %w", err)
	}
	return connection, nil
}
