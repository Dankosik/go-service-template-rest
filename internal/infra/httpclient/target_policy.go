package httpclient

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/example/go-service-template-rest/internal/outboundtrust"
)

// ErrTargetDenied reports a request or resolved address outside the client's fixed target.
var ErrTargetDenied = errors.New("outbound HTTP target denied")

// TargetClass selects the transport security policy for one fixed provider authority.
type TargetClass uint8

const (
	// ExternalHTTPS permits only HTTPS targets resolving to public addresses.
	ExternalHTTPS TargetClass = iota + 1
	// PrivateHTTP permits only plaintext HTTP targets under the platform's
	// private DNS zone, named by Config.PrivateHostSuffix. Transport security
	// is the platform's private network, not TLS.
	PrivateHTTP
	// profile:outbound-auth-oauth2-client-credentials:start
	// PrivateHTTPS permits only HTTPS targets under the platform's private DNS
	// zone and resolving to private addresses. TLS still uses the ordinary system
	// roots and hostname verification.
	PrivateHTTPS
	// profile:outbound-auth-oauth2-client-credentials:end
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
		if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && !outboundtrust.PublicAddress(address) {
			return ErrTargetDenied
		}
	case PrivateHTTP:
		return validatePrivateTarget(baseURL, "http", requiredPrivateSuffix)
	// profile:outbound-auth-oauth2-client-credentials:start
	case PrivateHTTPS:
		return validatePrivateTarget(baseURL, "https", requiredPrivateSuffix)
	// profile:outbound-auth-oauth2-client-credentials:end
	default:
		return errors.New("build outbound HTTP client: target class is invalid")
	}
	return nil
}

func validatePrivateTarget(baseURL *url.URL, requiredScheme, requiredPrivateSuffix string) error {
	if requiredPrivateSuffix == "." || requiredPrivateSuffix == "" {
		return errors.New(
			"build outbound HTTP client: private target requires Config.PrivateHostSuffix, " +
				"the deployment platform's private DNS zone",
		)
	}
	if !strings.EqualFold(baseURL.Scheme, requiredScheme) {
		return fmt.Errorf("build outbound HTTP client: private target requires %s", strings.ToUpper(requiredScheme))
	}
	hostname := strings.ToLower(strings.TrimSuffix(baseURL.Hostname(), "."))
	if !strings.HasSuffix(hostname, requiredPrivateSuffix) {
		return fmt.Errorf(
			"build outbound HTTP client: private target requires a %s hostname",
			strings.TrimPrefix(requiredPrivateSuffix, "."),
		)
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
		if outboundtrust.PublicAddress(resolved) {
			return nil
		}
	case PrivateHTTP:
		return enforcePrivateDialAddress(resolved)
	// profile:outbound-auth-oauth2-client-credentials:start
	case PrivateHTTPS:
		return enforcePrivateDialAddress(resolved)
		// profile:outbound-auth-oauth2-client-credentials:end
	}
	return ErrTargetDenied
}

func enforcePrivateDialAddress(resolved netip.Addr) error {
	if resolved.Unmap().IsPrivate() {
		return nil
	}
	return ErrTargetDenied
}
