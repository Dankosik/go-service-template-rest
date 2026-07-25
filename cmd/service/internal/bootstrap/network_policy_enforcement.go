package bootstrap

import (
	"fmt"
	"net"
	"strings"
)

// EnforceIngress requires an explicit operator acknowledgement before a
// non-local wildcard listener starts. It attests that the exposure decision was
// made; it does not restrict reachability, which only the deployment platform
// (firewall, security group, network policy, or service mesh) can enforce.
// Both "true" and "false" are accepted answers.
func (p networkPolicy) EnforceIngress() error {
	if p.ingressDeclarationRequired && !p.ingressAcknowledged {
		return fmt.Errorf(
			"%w: %s must be set to true or false to acknowledge the exposure of a non-local wildcard HTTP bind; "+
				"the deployment platform still owns reachability",
			errDependencyInit,
			envNetworkPublicIngressAcknowledged,
		)
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
