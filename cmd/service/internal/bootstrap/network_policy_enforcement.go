package bootstrap

import (
	"fmt"
	"net"
	"strings"
)

// EnforceIngress validates that non-local wildcard listeners have an explicit
// public/private declaration.
func (p networkPolicy) EnforceIngress() error {
	if p.ingressDeclarationRequired && !p.ingressPublicExplicitValue {
		return fmt.Errorf("%w: %s must be explicitly set for non-local wildcard HTTP bind", errDependencyInit, envNetworkPublicIngressEnabled)
	}
	return nil
}

func (p networkPolicy) withIngressExposure(env, addr string) networkPolicy {
	p.ingressDeclarationRequired = requiresPublicIngressDeclaration(env, addr)
	return p
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

func normalizeHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host
}
