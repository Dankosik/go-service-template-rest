package httpclient

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// ErrTargetDenied reports a request or resolved address outside the client's fixed target.
var ErrTargetDenied = errors.New("outbound HTTP target denied")

var (
	// netip.Addr.IsGlobalUnicast intentionally follows the IP protocol's broad
	// definition, not IANA's globally-reachable registry. These are the
	// special-purpose ranges it would otherwise admit.
	nonPublicIPv4Prefixes = [...]netip.Prefix{
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

// TargetClass selects the transport security policy for one fixed provider authority.
type TargetClass uint8

const (
	// ExternalHTTPS permits only HTTPS targets resolving to public addresses.
	ExternalHTTPS TargetClass = iota + 1
	// PrivateHTTP permits only plaintext HTTP targets under the platform's
	// private DNS zone, named by Config.PrivateHostSuffix. Transport security
	// is the platform's private network, not TLS.
	PrivateHTTP
)

// There is deliberately no default private DNS zone: one would succeed silently
// on the platform it was written for and fail confusingly everywhere else. The
// caller names its own — ".railway.internal", ".svc.cluster.local", ".internal".

// privateHostSuffix normalizes the configured private DNS zone. An empty zone
// stays empty so validateTarget rejects it.
func privateHostSuffix(configured string) string {
	suffix := strings.ToLower(strings.TrimSpace(configured))
	if suffix == "" {
		return ""
	}
	if !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}
	return strings.TrimSuffix(suffix, ".")
}

func validateTarget(baseURL *url.URL, targetClass TargetClass, requiredPrivateSuffix string) error {
	switch targetClass {
	case ExternalHTTPS:
		if !strings.EqualFold(baseURL.Scheme, "https") {
			return errors.New("build outbound HTTP client: external target requires HTTPS")
		}
		if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && isForbiddenExternalAddress(address) {
			return ErrTargetDenied
		}
	case PrivateHTTP:
		if requiredPrivateSuffix == "." || requiredPrivateSuffix == "" {
			return errors.New(
				"build outbound HTTP client: private target requires Config.PrivateHostSuffix, " +
					"the deployment platform's private DNS zone",
			)
		}
		if !strings.EqualFold(baseURL.Scheme, "http") {
			return errors.New("build outbound HTTP client: private target requires HTTP")
		}
		hostname := strings.ToLower(strings.TrimSuffix(baseURL.Hostname(), "."))
		if !strings.HasSuffix(hostname, requiredPrivateSuffix) {
			return fmt.Errorf(
				"build outbound HTTP client: private target requires a %s hostname",
				strings.TrimPrefix(requiredPrivateSuffix, "."),
			)
		}
	default:
		return errors.New("build outbound HTTP client: target class is invalid")
	}
	return nil
}

// authorityTransport judges the per-request URL: it refuses any request whose
// scheme or authority drifted from the one the client was built for, before a
// dial can happen.
type authorityTransport struct {
	base      http.RoundTripper
	scheme    string
	authority string
}

func (t authorityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil ||
		req.URL.User != nil ||
		!strings.EqualFold(req.URL.Scheme, t.scheme) ||
		!strings.EqualFold(req.URL.Host, t.authority) {
		return nil, ErrTargetDenied
	}
	response, err := t.base.RoundTrip(req)
	if err != nil {
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	return response, nil
}

// The fixed-target guarantee has three checks, none of which alone is enough:
// validateTarget judges the configured hostname at construction, but a hostname
// can resolve to a different address at dial time; enforceDialAddress judges the
// post-DNS address on every dial, but a pooled connection is reused across
// requests without redialing; and authorityTransport.RoundTrip judges the
// per-request URL before it reaches a pooled or new connection, but it cannot
// see what an accepted hostname resolves to.

// enforceDialAddress is the post-DNS check: validateTarget can only judge the
// configured hostname, and this judges what that hostname actually resolved to,
// on every dial.
func enforceDialAddress(targetClass TargetClass, address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrTargetDenied
	}
	resolved, err := netip.ParseAddr(host)
	if err != nil {
		return ErrTargetDenied
	}
	switch targetClass {
	case ExternalHTTPS:
		if !isForbiddenExternalAddress(resolved) {
			return nil
		}
	case PrivateHTTP:
		if resolved.Unmap().IsPrivate() {
			return nil
		}
	}
	return ErrTargetDenied
}

func isForbiddenExternalAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsMulticast() ||
		address.IsUnspecified() {
		return true
	}

	if address.Is4() {
		for _, prefix := range globallyReachableIPv4SpecialPrefixes {
			if prefix.Contains(address) {
				return false
			}
		}
		for _, prefix := range nonPublicIPv4Prefixes {
			if prefix.Contains(address) {
				return true
			}
		}
		return false
	}

	if publicNAT64Prefix.Contains(address) {
		bits := address.As16()
		return isForbiddenExternalAddress(netip.AddrFrom4([4]byte{
			bits[12], bits[13], bits[14], bits[15],
		}))
	}
	if !allocatedGlobalIPv6Prefix.Contains(address) {
		return true
	}
	for _, prefix := range globallyReachableIPv6SpecialPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	for _, prefix := range nonPublicIPv6Prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
