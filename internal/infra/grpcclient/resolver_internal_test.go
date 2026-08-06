// Resolver guard, white-box: does the wrapper in resolver.go strip
// Address.Metadata everywhere it can appear, while leaving the wrapped
// resolver's lifecycle, optional authority override, and build error intact?
//
// It drives the wrapper types directly, so it needs no network and no
// subprocess. The two black-box halves answer different questions:
// resolver_selection_test.go asks which builder grpc-go actually selects, and
// resolver_live_test.go asks whether the guard survives a real TLS connection
// with proxy environment set.

//nolint:staticcheck // This file proves grpc-go's deprecated but live resolver metadata path is sanitized.
package grpcclient

import (
	"errors"
	"testing"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

func TestResolverWrapperSanitizesEveryAddressAndPreservesLifecycle(t *testing.T) {
	t.Parallel()

	baseResolver := &recordingResolver{}
	baseBuilder := &recordingResolverBuilder{
		scheme:   "recording",
		resolver: baseResolver,
	}
	downstream := &recordingResolverClientConn{}
	wrapped := wrapResolverBuilder(baseBuilder)

	gotResolver, err := wrapped.Build(resolver.Target{}, downstream, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if gotResolver != baseResolver {
		t.Fatalf("Build() resolver = %T, want original resolver", gotResolver)
	}

	addressAttributes := attributes.New("address", "top")
	balancerAttributes := attributes.New("balancer", "top")
	endpointAttributes := attributes.New("endpoint", "one")
	endpointAddressAttributes := attributes.New("address", "endpoint")
	endpointBalancerAttributes := attributes.New("balancer", "endpoint")
	stateAttributes := attributes.New("state", "retained")
	serviceConfig := &serviceconfig.ParseResult{Err: errors.New("opaque parsed service config")}
	state := resolver.State{
		Addresses: []resolver.Address{{
			Addr:               "127.0.0.1:9000",
			ServerName:         "service.internal",
			Attributes:         addressAttributes,
			BalancerAttributes: balancerAttributes,
			Metadata:           map[string]string{"traceparent": "stale"},
		}},
		Endpoints: []resolver.Endpoint{{
			Attributes: endpointAttributes,
			Addresses: []resolver.Address{{
				Addr:               "127.0.0.1:9001",
				ServerName:         "endpoint.service.internal",
				Attributes:         endpointAddressAttributes,
				BalancerAttributes: endpointBalancerAttributes,
				Metadata:           map[string]string{"x-request-id": "stale"},
			}},
		}},
		ServiceConfig: serviceConfig,
		Attributes:    stateAttributes,
	}
	if err := baseBuilder.connection.UpdateState(state); err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}
	baseBuilder.connection.NewAddress(state.Addresses)

	assertResolverMetadataRemoved(t, downstream.state.Addresses)
	assertResolverMetadataRemoved(t, downstream.state.Endpoints[0].Addresses)
	assertResolverMetadataRemoved(t, downstream.addresses)
	if state.Addresses[0].Metadata == nil || state.Endpoints[0].Addresses[0].Metadata == nil {
		t.Fatal("caller-owned resolver state was mutated")
	}
	if downstream.updateCalls != 1 || downstream.newAddressCalls != 1 {
		t.Fatalf(
			"resolver callbacks = UpdateState %d NewAddress %d, want 1 each",
			downstream.updateCalls,
			downstream.newAddressCalls,
		)
	}
	if downstream.state.Addresses[0].Addr != state.Addresses[0].Addr ||
		downstream.state.Addresses[0].ServerName != state.Addresses[0].ServerName ||
		downstream.state.Addresses[0].Attributes != addressAttributes ||
		downstream.state.Addresses[0].BalancerAttributes != balancerAttributes {
		t.Fatalf("top-level address fields were not preserved: %+v", downstream.state.Addresses[0])
	}
	if downstream.state.Endpoints[0].Attributes != endpointAttributes ||
		downstream.state.Endpoints[0].Addresses[0].Addr !=
			state.Endpoints[0].Addresses[0].Addr ||
		downstream.state.Endpoints[0].Addresses[0].ServerName !=
			state.Endpoints[0].Addresses[0].ServerName ||
		downstream.state.Endpoints[0].Addresses[0].Attributes != endpointAddressAttributes ||
		downstream.state.Endpoints[0].Addresses[0].BalancerAttributes !=
			endpointBalancerAttributes {
		t.Fatalf("endpoint fields were not preserved: %+v", downstream.state.Endpoints[0])
	}
	if downstream.state.ServiceConfig != serviceConfig || downstream.state.Attributes != stateAttributes {
		t.Fatalf("state-owned fields were not preserved: %+v", downstream.state)
	}
	if downstream.addresses[0].Addr != state.Addresses[0].Addr ||
		downstream.addresses[0].ServerName != state.Addresses[0].ServerName ||
		downstream.addresses[0].Attributes != addressAttributes ||
		downstream.addresses[0].BalancerAttributes != balancerAttributes {
		t.Fatalf("NewAddress fields were not preserved: %+v", downstream.addresses[0])
	}
	downstream.state.Addresses[0].Addr = "changed"
	if state.Addresses[0].Addr != "127.0.0.1:9000" {
		t.Fatal("resolver address slices still alias caller state")
	}
	downstream.state.Endpoints[0].Addresses[0].Addr = "changed"
	if state.Endpoints[0].Addresses[0].Addr != "127.0.0.1:9001" {
		t.Fatal("resolver endpoint address slices still alias caller state")
	}
	downstream.addresses[0].Addr = "changed"
	if state.Addresses[0].Addr != "127.0.0.1:9000" {
		t.Fatal("deprecated NewAddress slice still aliases caller state")
	}

	updateErr := errors.New("resolver state rejected")
	downstream.updateErr = updateErr
	if err := baseBuilder.connection.UpdateState(state); !errors.Is(err, updateErr) {
		t.Fatalf("UpdateState() error = %v, want sentinel", err)
	}
	if downstream.updateCalls != 2 {
		t.Fatalf("UpdateState() calls = %d, want 2", downstream.updateCalls)
	}

	reportErr := errors.New("resolver lookup failed")
	baseBuilder.connection.ReportError(reportErr)
	if downstream.reportCalls != 1 || !errors.Is(downstream.reportErr, reportErr) {
		t.Fatalf(
			"ReportError() = calls %d error %v, want one sentinel callback",
			downstream.reportCalls,
			downstream.reportErr,
		)
	}

	parseResult := &serviceconfig.ParseResult{Err: errors.New("parsed config sentinel")}
	downstream.parseResult = parseResult
	if got := baseBuilder.connection.ParseServiceConfig(`{"loadBalancingConfig":[]}`); got != parseResult {
		t.Fatalf("ParseServiceConfig() = %p, want %p", got, parseResult)
	}
	if downstream.parseCalls != 1 ||
		downstream.parseInput != `{"loadBalancingConfig":[]}` {
		t.Fatalf(
			"ParseServiceConfig() callback = calls %d input %q",
			downstream.parseCalls,
			downstream.parseInput,
		)
	}

	gotResolver.ResolveNow(resolver.ResolveNowOptions{})
	gotResolver.Close()
	if baseResolver.resolveNowCalls != 1 || baseResolver.closeCalls != 1 {
		t.Fatalf(
			"resolver lifecycle calls = resolve %d close %d, want 1 each",
			baseResolver.resolveNowCalls,
			baseResolver.closeCalls,
		)
	}
}

func TestResolverWrapperPreservesOptionalAuthorityOverride(t *testing.T) {
	t.Parallel()

	base := &authorityResolverBuilder{
		recordingResolverBuilder: recordingResolverBuilder{
			scheme:   "authority",
			resolver: &recordingResolver{},
		},
		authority: "service.internal",
	}
	wrapped := wrapResolverBuilder(base)
	authority, ok := wrapped.(resolver.AuthorityOverrider)
	if !ok {
		t.Fatal("wrapped resolver lost AuthorityOverrider")
	}
	if got := authority.OverrideAuthority(resolver.Target{}); got != base.authority {
		t.Fatalf("OverrideAuthority() = %q, want %q", got, base.authority)
	}

	ordinary := &recordingResolverBuilder{
		scheme:   "ordinary",
		resolver: &recordingResolver{},
	}
	if _, ok := wrapResolverBuilder(ordinary).(resolver.AuthorityOverrider); ok {
		t.Fatal("ordinary resolver unexpectedly gained AuthorityOverrider")
	}
}

func TestResolverWrapperPreservesBuildError(t *testing.T) {
	t.Parallel()

	buildErr := errors.New("resolver build failed")
	base := &recordingResolverBuilder{
		scheme:   "failing",
		buildErr: buildErr,
	}
	wrapped := wrapResolverBuilder(base)
	if _, err := wrapped.Build(
		resolver.Target{},
		&recordingResolverClientConn{},
		resolver.BuildOptions{},
	); !errors.Is(err, buildErr) {
		t.Fatalf("Build() error = %v, want sentinel", err)
	}
	if base.buildCalls != 1 {
		t.Fatalf("Build() calls = %d, want 1", base.buildCalls)
	}
}

type recordingResolver struct {
	resolveNowCalls int
	closeCalls      int
}

func (r *recordingResolver) ResolveNow(resolver.ResolveNowOptions) {
	r.resolveNowCalls++
}

func (r *recordingResolver) Close() {
	r.closeCalls++
}

type recordingResolverBuilder struct {
	scheme     string
	resolver   resolver.Resolver
	connection resolver.ClientConn
	buildErr   error
	buildCalls int
}

func (b *recordingResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder for delegation proof.
	_ resolver.Target,
	connection resolver.ClientConn,
	_ resolver.BuildOptions,
) (resolver.Resolver, error) {
	b.buildCalls++
	b.connection = connection
	return b.resolver, b.buildErr
}

func (b *recordingResolverBuilder) Scheme() string {
	return b.scheme
}

type authorityResolverBuilder struct {
	recordingResolverBuilder

	authority string
}

func (b authorityResolverBuilder) OverrideAuthority(resolver.Target) string {
	return b.authority
}

type recordingResolverClientConn struct {
	state           resolver.State
	addresses       []resolver.Address
	updateErr       error
	reportErr       error
	parseResult     *serviceconfig.ParseResult
	parseInput      string
	updateCalls     int
	reportCalls     int
	newAddressCalls int
	parseCalls      int
}

func (c *recordingResolverClientConn) UpdateState(state resolver.State) error {
	c.updateCalls++
	c.state = state
	return c.updateErr
}

func (c *recordingResolverClientConn) ReportError(err error) {
	c.reportCalls++
	c.reportErr = err
}

func (c *recordingResolverClientConn) NewAddress(addresses []resolver.Address) {
	c.newAddressCalls++
	c.addresses = addresses
}

func (c *recordingResolverClientConn) ParseServiceConfig(
	serviceConfig string,
) *serviceconfig.ParseResult {
	c.parseCalls++
	c.parseInput = serviceConfig
	return c.parseResult
}

func assertResolverMetadataRemoved(t *testing.T, addresses []resolver.Address) {
	t.Helper()

	for _, address := range addresses {
		if address.Metadata != nil {
			t.Fatalf("resolver address %q metadata = %v, want nil", address.Addr, address.Metadata)
		}
	}
}
