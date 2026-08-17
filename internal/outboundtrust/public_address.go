// Package outboundtrust owns transport-independent public-address admission.
package outboundtrust

import "net/netip"

// IANASpecialPurposeRegistryRevision pins both special-purpose registries used
// below. Update the corpus with the revision, never the date alone.
const IANASpecialPurposeRegistryRevision = "2025-10-09"

var (
	nonPublicIPv4Prefixes = [...]netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
	}
	globallyReachableIPv4SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("192.0.0.9/32"),
		netip.MustParsePrefix("192.0.0.10/32"),
	}
	allocatedGlobalIPv6Prefix = netip.MustParsePrefix("2000::/3")
	publicNAT64Prefix         = netip.MustParsePrefix("64:ff9b::/96")
	nonPublicIPv6Prefixes     = [...]netip.Prefix{
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"),
		netip.MustParsePrefix("3fff::/20"),
	}
	globallyReachableIPv6SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("2001:1::1/128"),
		netip.MustParsePrefix("2001:1::2/128"),
		netip.MustParsePrefix("2001:1::3/128"),
		netip.MustParsePrefix("2001:3::/32"),
		netip.MustParsePrefix("2001:4:112::/48"),
		netip.MustParsePrefix("2001:20::/28"),
		netip.MustParsePrefix("2001:30::/28"),
	}
)

// PublicAddress reports whether address is globally reachable under the pinned
// IANA IPv4 and IPv6 Special-Purpose Address Space registries.
func PublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() ||
		address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	if address.Is4() {
		for _, prefix := range globallyReachableIPv4SpecialPrefixes {
			if prefix.Contains(address) {
				return true
			}
		}
		for _, prefix := range nonPublicIPv4Prefixes {
			if prefix.Contains(address) {
				return false
			}
		}
		return true
	}
	if publicNAT64Prefix.Contains(address) {
		bits := address.As16()
		return PublicAddress(netip.AddrFrom4([4]byte{bits[12], bits[13], bits[14], bits[15]}))
	}
	if !allocatedGlobalIPv6Prefix.Contains(address) {
		return false
	}
	for _, prefix := range globallyReachableIPv6SpecialPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	for _, prefix := range nonPublicIPv6Prefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
