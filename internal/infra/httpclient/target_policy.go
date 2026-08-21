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
	baseURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || baseURL == nil || !baseURL.IsAbs() || baseURL.Opaque != "" ||
		baseURL.Host == "" || baseURL.Hostname() == "" {
		return nil, errors.New("build outbound HTTP client: base URL must be absolute")
	}
	if !strings.EqualFold(baseURL.Scheme, "https") {
		return nil, errors.New("build outbound HTTP client: target requires HTTPS")
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.ForceQuery || baseURL.Fragment != "" {
		return nil, errors.New("build outbound HTTP client: base URL cannot contain user info, query, or fragment")
	}
	if policy.privateSuffix == "" {
		if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && !outboundtrust.PublicAddress(address) {
			return nil, ErrTargetDenied
		}
	} else {
		hostname := strings.ToLower(strings.TrimSuffix(baseURL.Hostname(), "."))
		if !strings.HasSuffix(hostname, policy.privateSuffix) {
			return nil, errors.New("build outbound HTTP client: private target requires the configured DNS suffix")
		}
	}
	baseURL.Scheme = "https"
	baseURL.Host = strings.ToLower(baseURL.Host)
	return baseURL, nil
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
	if (policy.privateSuffix == "" && outboundtrust.PublicAddress(resolved)) ||
		(policy.privateSuffix != "" && resolved.Unmap().IsPrivate()) {
		return nil
	}
	return ErrTargetDenied
}

type authorityTransport struct {
	base      http.RoundTripper
	scheme    string
	authority string
}

func (t authorityTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || request.URL.User != nil ||
		!strings.EqualFold(request.URL.Scheme, t.scheme) ||
		!strings.EqualFold(request.URL.Host, t.authority) {
		return nil, ErrTargetDenied
	}
	return t.base.RoundTrip(request) //nolint:wrapcheck // The transport error keeps its standard identity.
}
