package grpcclient

import (
	"net/url"
	"slices"

	"google.golang.org/grpc/resolver"
)

// sanitizingResolverBuilders wraps every resolver grpc-go could select for
// target, so the selected one cannot hand addresses to the balancer with
// Address.Metadata still attached.
//
// That deprecated field is the third strip seam named in propagation.go, and it
// is not theoretical: TestResolverSelectionHarnessExposesUnwrappedAddressMetadata
// drives an unwrapped resolver and observes its stale request ID arrive at the
// server as x-request-id. A resolver is not part of this client's trust
// boundary, so its metadata never reaches the wire.
//
// The scheme set mirrors grpc-go's own selection rules rather than guessing one:
// an explicit registered scheme in the target wins, and a bare target, an
// unregistered scheme, or a relocated global default all fall back to
// resolver.GetDefaultScheme(). Wrapping the union means whichever builder grpc-go
// picks is already wrapped — an unused wrapper costs nothing, while missing the
// selected one silently drops the guard.
func sanitizingResolverBuilders(target string) []resolver.Builder {
	schemes := []string{"dns", resolver.GetDefaultScheme()}
	if parsed, err := url.Parse(target); err == nil {
		schemes = append(schemes, parsed.Scheme)
	}

	builders := make([]resolver.Builder, 0, len(schemes))
	for _, scheme := range schemes {
		base := resolver.Get(scheme)
		if base == nil {
			continue
		}
		if slices.ContainsFunc(builders, func(builder resolver.Builder) bool {
			return builder.Scheme() == base.Scheme()
		}) {
			continue
		}
		builders = append(builders, wrapResolverBuilder(base))
	}
	return builders
}

func wrapResolverBuilder( //nolint:ireturn // Optional AuthorityOverrider support requires two concrete wrappers.
	base resolver.Builder,
) resolver.Builder {
	wrapped := sanitizingResolverBuilder{base: base}
	authority, ok := base.(resolver.AuthorityOverrider)
	if !ok {
		return wrapped
	}
	return sanitizingAuthorityResolverBuilder{
		sanitizingResolverBuilder: wrapped,
		authority:                 authority,
	}
}

type sanitizingResolverBuilder struct {
	base resolver.Builder
}

func (b sanitizingResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder.
	target resolver.Target,
	connection resolver.ClientConn,
	options resolver.BuildOptions,
) (resolver.Resolver, error) {
	//nolint:wrapcheck // Preserve the selected resolver's exact build failure.
	return b.base.Build(target, sanitizingResolverClientConn{ClientConn: connection}, options)
}

func (b sanitizingResolverBuilder) Scheme() string {
	return b.base.Scheme()
}

type sanitizingAuthorityResolverBuilder struct {
	sanitizingResolverBuilder

	authority resolver.AuthorityOverrider
}

func (b sanitizingAuthorityResolverBuilder) OverrideAuthority(target resolver.Target) string {
	return b.authority.OverrideAuthority(target)
}

type sanitizingResolverClientConn struct {
	resolver.ClientConn
}

func (c sanitizingResolverClientConn) UpdateState(state resolver.State) error {
	// The resolver contract requires returning the downstream ClientConn error
	// unchanged; callers may use its identity for recovery.
	//nolint:wrapcheck // Preserve resolver.ClientConn recovery semantics and error identity.
	return c.ClientConn.UpdateState(sanitizeResolverState(state))
}

func (c sanitizingResolverClientConn) NewAddress(addresses []resolver.Address) {
	c.ClientConn.NewAddress( //nolint:staticcheck // Compatibility path required by resolver.ClientConn.
		sanitizeResolverAddresses(addresses),
	)
}

func sanitizeResolverState(state resolver.State) resolver.State {
	state.Addresses = sanitizeResolverAddresses(state.Addresses)
	state.Endpoints = slices.Clone(state.Endpoints)
	for index := range state.Endpoints {
		state.Endpoints[index].Addresses = sanitizeResolverAddresses(state.Endpoints[index].Addresses)
	}
	return state
}

func sanitizeResolverAddresses(addresses []resolver.Address) []resolver.Address {
	sanitized := slices.Clone(addresses)
	for index := range sanitized {
		sanitized[index].Metadata = nil //nolint:staticcheck // Deprecated Metadata remains a live grpc-go disclosure path.
	}
	return sanitized
}
