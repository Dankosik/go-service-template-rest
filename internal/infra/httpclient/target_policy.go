package httpclient

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/example/go-service-template-rest/internal/outboundtrust"
)

// ErrTargetDenied reports a request or resolved address outside the fixed target.
var ErrTargetDenied = errors.New("outbound HTTP target denied")

type targetPolicy struct {
	privateSuffix string
}

// private reports whether the policy admits only hosts under a configured
// private DNS suffix instead of public addresses.
func (p targetPolicy) private() bool {
	return p.privateSuffix != ""
}

// allowsAddress reports whether a dial may reach resolved: a public address
// for a public policy, a private one for a private policy.
func (p targetPolicy) allowsAddress(resolved netip.Addr) bool {
	if p.private() {
		return resolved.Unmap().IsPrivate()
	}
	return outboundtrust.PublicAddress(resolved)
}

func privateHostSuffix(configured string) string {
	suffix := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(configured)), ".")
	if suffix == "" {
		return ""
	}
	if !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}
	return suffix
}

func validateTarget(raw string, policy targetPolicy) (*url.URL, error) {
	baseURL, issue := outboundtrust.HTTPSTarget(strings.TrimSpace(raw))
	switch issue {
	case outboundtrust.TargetOK:
	case outboundtrust.TargetNotAbsolute:
		return nil, errors.New("build outbound HTTP client: base URL must be absolute")
	case outboundtrust.TargetNotHTTPS:
		return nil, errors.New("build outbound HTTP client: target requires HTTPS")
	case outboundtrust.TargetHasUserInfoOrFragment, outboundtrust.TargetHasQuery:
		return nil, errors.New("build outbound HTTP client: base URL cannot contain user info, query, or fragment")
	}
	if policy.private() {
		hostname := strings.TrimSuffix(baseURL.Hostname(), ".")
		if !strings.HasSuffix(hostname, policy.privateSuffix) {
			return nil, errors.New("build outbound HTTP client: private target requires the configured DNS suffix")
		}
	} else if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && !policy.allowsAddress(address) {
		return nil, ErrTargetDenied
	}
	return &baseURL, nil
}

func enforceDialAddress(policy targetPolicy, address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrTargetDenied
	}
	resolved, err := netip.ParseAddr(host)
	if err != nil {
		return ErrTargetDenied
	}
	if !policy.allowsAddress(resolved) {
		return ErrTargetDenied
	}
	return nil
}

type authorityTransport struct {
	base      http.RoundTripper
	scheme    string
	authority string
}

func (t authorityTransport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t authorityTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || request.URL.User != nil ||
		!strings.EqualFold(request.URL.Scheme, t.scheme) ||
		!strings.EqualFold(request.URL.Host, t.authority) ||
		request.Host != "" && !strings.EqualFold(request.Host, t.authority) {
		return nil, ErrTargetDenied
	}
	return t.base.RoundTrip(request) //nolint:wrapcheck // The transport error keeps its standard identity.
}
