// Package outboundtrust owns transport-independent outbound target rules: the
// fixed HTTPS target shape and public-address admission.
package outboundtrust

import (
	"net/netip"
	"slices"
)

// ianaSpecialPurposeRegistryRevision pins both special-purpose registries used
// below. Update the corpus with the revision, never the date alone.
const ianaSpecialPurposeRegistryRevision = "2025-10-09"

var (
	nonPublicIPv4Prefixes = [...]netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),       // "This network"
		netip.MustParsePrefix("100.64.0.0/10"),   // Shared Address Space
		netip.MustParsePrefix("192.0.0.0/24"),    // IETF Protocol Assignments
		netip.MustParsePrefix("192.0.2.0/24"),    // Documentation (TEST-NET-1)
		netip.MustParsePrefix("192.88.99.0/24"),  // Deprecated (6to4 Relay Anycast)
		netip.MustParsePrefix("198.18.0.0/15"),   // Benchmarking
		netip.MustParsePrefix("198.51.100.0/24"), // Documentation (TEST-NET-2)
		netip.MustParsePrefix("203.0.113.0/24"),  // Documentation (TEST-NET-3)
		netip.MustParsePrefix("240.0.0.0/4"),     // Reserved
	}
	globallyReachableIPv4SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("192.0.0.9/32"),  // Port Control Protocol Anycast
		netip.MustParsePrefix("192.0.0.10/32"), // Traversal Using Relays around NAT Anycast
	}
	allocatedGlobalIPv6Prefix = netip.MustParsePrefix("2000::/3")
	publicNAT64Prefix         = netip.MustParsePrefix("64:ff9b::/96") // IPv4-IPv6 Translat.
	nonPublicIPv6Prefixes     = [...]netip.Prefix{
		netip.MustParsePrefix("2001::/23"),     // IETF Protocol Assignments
		netip.MustParsePrefix("2001:db8::/32"), // Documentation
		netip.MustParsePrefix("2002::/16"),     // 6to4
		netip.MustParsePrefix("3fff::/20"),     // Documentation
	}
	globallyReachableIPv6SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("2001:1::1/128"),   // Port Control Protocol Anycast
		netip.MustParsePrefix("2001:1::2/128"),   // Traversal Using Relays around NAT Anycast
		netip.MustParsePrefix("2001:1::3/128"),   // DNS-SD Service Registration Protocol Anycast
		netip.MustParsePrefix("2001:3::/32"),     // AMT
		netip.MustParsePrefix("2001:4:112::/48"), // AS112-v6
		netip.MustParsePrefix("2001:20::/28"),    // ORCHIDv2
		netip.MustParsePrefix("2001:30::/28"),    // Drone Remote ID Protocol Entity Tags (DETs) Prefix
	}
)

// PublicAddress reports whether address is admitted for public egress under the
// pinned IANA IPv4 and IPv6 Special-Purpose Address Space registry policy.
// Ambiguous registry entries and NAT64 addresses embedding non-public IPv4 fail
// closed.
func PublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() {
		return false
	}
	if address.Is4() {
		if containsAddr(globallyReachableIPv4SpecialPrefixes[:], address) {
			return true
		}
		return !containsAddr(nonPublicIPv4Prefixes[:], address)
	}
	if publicNAT64Prefix.Contains(address) {
		bits := address.As16()
		return PublicAddress(netip.AddrFrom4([4]byte{bits[12], bits[13], bits[14], bits[15]}))
	}
	if !allocatedGlobalIPv6Prefix.Contains(address) {
		return false
	}
	if containsAddr(globallyReachableIPv6SpecialPrefixes[:], address) {
		return true
	}
	return !containsAddr(nonPublicIPv6Prefixes[:], address)
}

func containsAddr(prefixes []netip.Prefix, address netip.Addr) bool {
	return slices.ContainsFunc(prefixes, func(prefix netip.Prefix) bool { return prefix.Contains(address) })
}
