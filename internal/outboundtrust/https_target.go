package outboundtrust

import (
	"net/url"
	"strings"
)

// TargetIssue names the first rule a fixed HTTPS target breaks, so each caller
// can report it in its own error format.
type TargetIssue uint8

const (
	// TargetOK means the target is an absolute HTTPS URL with one exact
	// destination.
	TargetOK TargetIssue = iota
	// TargetNotAbsolute means the value does not parse as an absolute URL with
	// a host.
	TargetNotAbsolute
	// TargetNotHTTPS means the scheme is not https.
	TargetNotHTTPS
	// TargetHasUserInfoOrFragment means the URL carries user info or a
	// fragment.
	TargetHasUserInfoOrFragment
	// TargetHasQuery means the URL carries a query or a bare "?". It is checked
	// last, so a caller that admits provider-owned query parameters can accept
	// it.
	TargetHasQuery
)

// HTTPSTarget checks raw as a fixed HTTPS target and returns it with a
// lowercase scheme and host. The URL is meaningful only with TargetOK. raw is
// not trimmed: whether surrounding whitespace is repaired or rejected belongs
// to the caller.
func HTTPSTarget(raw string) (url.URL, TargetIssue) {
	parsed, err := url.Parse(raw)
	// The parsed nil check is not redundant. url.Parse's contract pairs a nil
	// result with a non-nil error, but nilaway reads the two as independent, and
	// dropping the guard fails the deep lint gate rather than a test.
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Opaque != "" ||
		parsed.Host == "" || parsed.Hostname() == "" {
		return url.URL{}, TargetNotAbsolute
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return url.URL{}, TargetNotHTTPS
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return url.URL{}, TargetHasUserInfoOrFragment
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return url.URL{}, TargetHasQuery
	}
	parsed.Scheme = "https"
	parsed.Host = strings.ToLower(parsed.Host)
	return *parsed, TargetOK
}
