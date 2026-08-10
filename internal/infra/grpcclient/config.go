package grpcclient

import "time"

// The shape a dependency owner fills in, and the conservative values it starts
// from. What proves a filled-in Config is legal stays in client.go beside [New]:
// unlike the server adapter in internal/infra/grpc, half of what has to be
// checked here — the transport credentials and the propagation policy — lives on
// [Options] rather than on Config, so the check belongs to the construction that
// reads both and not to either type alone.

// Config contains the fixed target and finite per-call transport bounds.
//
// MaxHeaderListBytes is the uint32 grpc-go asks for, where the server adapter in
// internal/infra/grpc takes the same bound as an int: that Config is filled from
// a configuration file and proves the conversion on the way in, while these come
// from [DefaultConfig] in Go source, where the narrower type is the check.
type Config struct {
	Target                 string
	MaxHeaderListBytes     uint32
	MaxReceiveMessageBytes int
	MaxSendMessageBytes    int

	// LoadBalancing selects how RPCs spread across the addresses Target
	// resolves to. Its own doc owns why it is a Config field.
	LoadBalancing LoadBalancingPolicy
	// HealthCheck makes supported round-robin backends eligible only while the
	// standard health service reports SERVING for the whole process.
	HealthCheck bool

	// KeepalivePingInterval and KeepalivePingTimeout opt into idle keepalive as
	// one complete positive pair. grpc-go raises an interval below ten seconds to
	// ten; the dependency owner must choose values its peer and intermediaries
	// accept.
	KeepalivePingInterval time.Duration
	KeepalivePingTimeout  time.Duration
}

// DefaultConfig returns the template's conservative transport constraints for
// target. A feature raises them only with matching workload and memory proof.
func DefaultConfig(target string) Config {
	return Config{
		Target:                 target,
		MaxHeaderListBytes:     16 << 10,
		MaxReceiveMessageBytes: 4 << 20,
		MaxSendMessageBytes:    4 << 20,
		LoadBalancing:          LoadBalancingRoundRobin,
		HealthCheck:            true,
	}
}
