package grpcclient

// LoadBalancingPolicy selects how one connection spreads RPCs across the
// addresses its target resolves to.
//
// It lives on [Config] beside Target rather than on [Options] because it is a
// property of the target's shape rather than a trust decision: one address makes
// both policies equivalent, and many make the choice matter. [Options] stays the
// trust and observability boundary.
type LoadBalancingPolicy uint8

const (
	// LoadBalancingRoundRobin connects to every resolved address and rotates
	// through them.
	//
	// It is the zero value so a connection nobody configured distributes rather
	// than pins. Against a mesh sidecar or any single-address target it is
	// equivalent to picking the first; against a headless DNS name it is the
	// only answer that reaches more than one backend, and grpc-go's own default
	// would send every RPC from every replica to one of them.
	LoadBalancingRoundRobin LoadBalancingPolicy = iota

	// LoadBalancingPickFirst sends every RPC to the first address that connects.
	// Choose it when the fan-out of one subchannel per backend is not wanted.
	LoadBalancingPickFirst
)

func (p LoadBalancingPolicy) valid() bool {
	return p <= LoadBalancingPickFirst
}

// serviceConfig renders the policy as this client's own default service config.
//
// Supplying it through grpc.WithDefaultServiceConfig composes with the
// grpc.WithDisableServiceConfig [New] already sets: that option refuses only
// what a *resolver* supplies, and grpc-go documents that a default service
// config is still used. So address selection stays this client's own decision
// and the trust boundary propagation.go describes is unchanged — nothing a peer
// controls reaches the balancer.
//
// Both arms are spelled out rather than leaving one to grpc-go's default, so the
// selected policy is visible in the same shape either way.
func (p LoadBalancingPolicy) serviceConfig() string {
	if p == LoadBalancingPickFirst {
		return `{"loadBalancingConfig":[{"pick_first":{}}]}`
	}
	return `{"loadBalancingConfig":[{"round_robin":{}}]}`
}
