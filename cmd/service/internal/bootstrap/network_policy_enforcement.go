package bootstrap

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func (p networkPolicy) EnforceIngress(ctx context.Context, recorder *deployTelemetryRecorder) error {
	if !p.ingressPublicEnabled {
		recorder.RecordNetworkExceptionStateChange(ctx, "ingress", "closed", "deny", p.ingressException.ID)
		return nil
	}

	if !p.ingressException.Active {
		recorder.RecordNetworkExceptionStateChange(ctx, "ingress", "denied", "deny", p.ingressException.ID)
		recorder.RecordNetworkIngressPolicyViolation(ctx, "missing_exception", "deny")
		return fmt.Errorf("%w: public ingress denied without approved exception", config.ErrDependencyInit)
	}

	if p.isExceptionExpired(p.ingressException) {
		recorder.RecordNetworkExceptionStateChange(ctx, "ingress", "expired", "deny", p.ingressException.ID)
		recorder.RecordNetworkIngressPolicyViolation(ctx, "expired_exception", "deny")
		return fmt.Errorf("%w: ingress exception is expired", config.ErrDependencyInit)
	}

	recorder.RecordNetworkExceptionStateChange(ctx, "ingress", "active", "allow", p.ingressException.ID)
	return nil
}

func (p networkPolicy) EmitEgressExceptionState(ctx context.Context, recorder *deployTelemetryRecorder) error {
	if !p.egressException.Active {
		recorder.RecordNetworkExceptionStateChange(ctx, "egress", "closed", "deny", p.egressException.ID)
		return nil
	}
	if p.isExceptionExpired(p.egressException) {
		recorder.RecordNetworkExceptionStateChange(ctx, "egress", "expired", "deny", p.egressException.ID)
		recorder.RecordNetworkEgressPolicyViolation(ctx, "expired_exception", "deny")
		return fmt.Errorf("%w: egress exception is expired", config.ErrDependencyInit)
	}

	recorder.RecordNetworkExceptionStateChange(ctx, "egress", "active", "allow", p.egressException.ID)
	return nil
}

func (p networkPolicy) EnforceEgressTarget(ctx context.Context, recorder *deployTelemetryRecorder, target, scheme string) error {
	normalizedScheme := strings.ToLower(strings.TrimSpace(scheme))
	if !p.isSchemeAllowed(normalizedScheme) {
		recorder.RecordNetworkEgressPolicyViolation(ctx, "scheme_denied", "deny")
		return fmt.Errorf("%w: egress scheme denied by policy", config.ErrDependencyInit)
	}

	host, err := extractHost(target)
	if err != nil {
		recorder.RecordNetworkEgressPolicyViolation(ctx, "invalid_configuration", "deny")
		return fmt.Errorf("%w: invalid egress target", config.ErrDependencyInit)
	}

	if classifyHostExposure(host) != "public" {
		return nil
	}
	if matchesHost(host, p.egressAllowlist) {
		return nil
	}

	if p.egressException.Active && !p.isExceptionExpired(p.egressException) && matchesHost(host, p.egressException.scopeMatcher) {
		recorder.RecordNetworkExceptionStateChange(ctx, "egress", "active", "allow", p.egressException.ID)
		return nil
	}

	if p.egressException.Active && p.isExceptionExpired(p.egressException) {
		recorder.RecordNetworkExceptionStateChange(ctx, "egress", "expired", "deny", p.egressException.ID)
		recorder.RecordNetworkEgressPolicyViolation(ctx, "expired_exception", "deny")
		return fmt.Errorf("%w: egress exception is expired", config.ErrDependencyInit)
	}

	recorder.RecordNetworkEgressPolicyViolation(ctx, "public_target_denied", "deny")
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
			if normalized == base || strings.HasSuffix(normalized, matcher.suffix) {
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
	if !strings.Contains(normalized, ".") {
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
