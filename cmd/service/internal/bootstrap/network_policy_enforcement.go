package bootstrap

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func (p networkPolicy) EnforceIngress() error {
	return p.validatePublicIngress()
}

func (p networkPolicy) ValidateIngressRuntime() error {
	return p.validatePublicIngress()
}

func (p networkPolicy) withIngressExposure(env, addr string) networkPolicy {
	p.ingressDeclarationRequired = requiresPublicIngressDeclaration(env, addr)
	return p
}

func (p networkPolicy) validatePublicIngress() error {
	if p.ingressDeclarationRequired && !p.ingressPublicDeclared {
		return fmt.Errorf("%w: %s must be explicitly set for non-local wildcard HTTP bind", config.ErrDependencyInit, envNetworkPublicIngressEnabled)
	}
	if !p.ingressPublicEnabled {
		return nil
	}
	if !p.ingressException.Active {
		return fmt.Errorf("%w: public ingress denied without approved exception", config.ErrDependencyInit)
	}
	if p.isExceptionExpired(p.ingressException) {
		return fmt.Errorf("%w: ingress exception is expired", config.ErrDependencyInit)
	}
	return nil
}

func requiresPublicIngressDeclaration(env, addr string) bool {
	if strings.EqualFold(strings.TrimSpace(env), "local") {
		return false
	}
	return isWildcardHTTPBind(addr)
}

func isWildcardHTTPBind(addr string) bool {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return false
	}

	host, _, err := net.SplitHostPort(trimmed)
	if err != nil {
		host = trimmed
	}
	host = normalizeHost(host)
	if host == "" || host == "*" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

func (p networkPolicy) EmitEgressExceptionState() error {
	if !p.egressException.Active {
		return nil
	}
	if p.isExceptionExpired(p.egressException) {
		return fmt.Errorf("%w: egress exception is expired", config.ErrDependencyInit)
	}

	return nil
}

func (p networkPolicy) EnforceEgressTarget(target, scheme string) error {
	normalizedScheme := strings.ToLower(strings.TrimSpace(scheme))
	if !p.isSchemeAllowed(normalizedScheme) {
		return fmt.Errorf("%w: egress scheme denied by policy", config.ErrDependencyInit)
	}

	host, err := extractHost(target)
	if err != nil {
		return fmt.Errorf("%w: invalid egress target", config.ErrDependencyInit)
	}

	if classifyHostExposure(host) != "public" {
		return nil
	}
	if matchesHost(host, p.egressAllowlist) {
		return nil
	}

	if p.egressException.Active && !p.isExceptionExpired(p.egressException) && matchesHost(host, p.egressException.scopeMatcher) {
		return nil
	}

	if p.egressException.Active && p.isExceptionExpired(p.egressException) {
		return fmt.Errorf("%w: egress exception is expired", config.ErrDependencyInit)
	}

	return fmt.Errorf("%w: egress target denied by policy", config.ErrDependencyInit)
}

func (p networkPolicy) isSchemeAllowed(scheme string) bool {
	if scheme == "" {
		return false
	}
	_, ok := p.egressAllowedSchemes[scheme]
	return ok
}

func (p networkPolicy) isExceptionExpired(exception networkException) bool {
	if p.now == nil {
		return time.Now().After(exception.Expiry)
	}
	return p.now().After(exception.Expiry)
}

func matchesHost(host string, matchers []networkHostMatcher) bool {
	normalized := normalizeHost(host)
	if normalized == "" {
		return false
	}
	for _, matcher := range matchers {
		if matcher.exact != "" && normalized == matcher.exact {
			return true
		}
		if matcher.suffix != "" {
			base := strings.TrimPrefix(matcher.suffix, ".")
			if (matcher.includeApex && normalized == base) || strings.HasSuffix(normalized, matcher.suffix) {
				return true
			}
		}
	}
	return false
}

func normalizeHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host
}

func classifyHostExposure(host string) string {
	normalized := normalizeHost(host)
	if normalized == "" {
		return "public"
	}

	if ip := net.ParseIP(normalized); ip != nil {
		if isPrivateIP(ip) {
			return "private"
		}
		return "public"
	}

	if normalized == "localhost" {
		return "private"
	}
	if strings.HasSuffix(normalized, ".internal") ||
		strings.HasSuffix(normalized, ".local") ||
		strings.HasSuffix(normalized, ".svc") ||
		strings.HasSuffix(normalized, ".cluster.local") {
		return "private"
	}

	return "public"
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func extractHost(target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", fmt.Errorf("target is empty")
	}

	host, _, err := net.SplitHostPort(trimmed)
	if err == nil {
		host = normalizeHost(host)
		if host == "" {
			return "", fmt.Errorf("host is empty")
		}
		return host, nil
	}

	if strings.Count(trimmed, ":") == 0 {
		host = normalizeHost(trimmed)
		if host == "" {
			return "", fmt.Errorf("host is empty")
		}
		return host, nil
	}

	return "", fmt.Errorf("target must be host:port")
}
