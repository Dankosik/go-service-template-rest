// Package grpcclient owns construction of bounded, instrumented native gRPC
// client connections.
//
// [New] returns a shareable grpc.ClientConn and performs no network I/O, so one
// connection per dependency built at startup is the intended shape. The caller
// owns Close, and each call owns its own deadline and retry policy — this
// package sets neither. [DefaultConfig] supplies conservative per-call transport
// bounds that a feature raises only with matching workload and memory evidence,
// and [Options] carries the one dependency [New] refuses to invent: an explicit
// transport-credentials choice, plaintext included.
//
// # The trust boundary
//
// What this package adds over a bare grpc.NewClient is a guarantee about what
// leaves the process: exactly the correlation values [PropagationPolicy]
// selected, and nothing else. Three sources the caller does not fully control
// can otherwise put a value under a reserved key on the wire — metadata already
// on the outgoing context, metadata returned by a per-RPC credential, and
// address metadata attached by a resolver — so each has its own strip seam,
// applied before OpenTelemetry injects the selected allowlist. propagation.go
// owns the first two and the reserved key set; resolver.go owns the third.
// [New] closes the two remaining routes by which a peer could introduce a
// fourth, refusing both proxies and resolver-supplied service config. It still
// supplies a default service config of its own, carrying the
// [LoadBalancingPolicy] and optional standard health policy; that is this
// client's decision rather than a peer's, which is why the two compose.
//
// # Extending it
//
// [Config.LoadBalancing] selects how RPCs spread across the addresses a target
// resolves to. Round robin is the zero value, so a connection nobody configured
// distributes rather than pins; its own doc owns why. [DefaultConfig] enables
// standard health for round-robin backends, while [Config.HealthCheck] can turn
// it off for a dependency that does not publish whole-process health. [Config]
// also carries the complete interval/timeout pair that explicitly opts into
// keepalive pings on an idle connection; the default sends none.
//
// Selecting the propagation policy is the one decision most callers make, once
// per dependency rather than per call. [PropagationNone] is the zero value on
// purpose: a client nobody configured emits no remote correlation metadata at
// all, while still recording local client spans and metrics.
// [PropagationTraceContext] adds the W3C trace context, and
// [PropagationTrustedService] additionally sends the accepted request ID — the
// name says what it costs, because that identifier is only meaningful to a hop
// that will not leak or forge it.
//
// Sending a further correlation value to a trusted service is a change in three
// places, and any two of them without the third is a defect rather than a
// partial feature: reservedCorrelationMetadataKeys claims the key so no other
// source may set it, policyPropagator.Inject emits it under the selected
// policy, and policyPropagator.Fields declares it so the injecting carrier
// reserves room. Claiming without emitting silently drops a caller's value;
// emitting without claiming lets a caller forge one.
//
// Per-RPC credentials can be attached to the whole connection through
// [Options.PerRPCCredentials], which also reaches grpc-go control streams, or
// to one application call. Both forms pass through unchanged apart from the
// reserved keys, which are removed from whatever the credential returns. The
// credential's own errors and its RequireTransportSecurity answer are
// preserved exactly, because gRPC turns them into the authentication failure
// the caller sees.
//
// See docs/grpc.md for the service-to-service contract and
// docs/repo-architecture.md for the repository's extension seams.
package grpcclient
