package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	envNetworkPublicIngressEnabled = "NETWORK_PUBLIC_INGRESS_ENABLED"
	envNetworkEgressAllowlist      = "NETWORK_EGRESS_ALLOWLIST"
	envNetworkEgressAllowedSchemes = "NETWORK_EGRESS_ALLOWED_SCHEMES"
)

type networkPolicy struct {
	now                  func() time.Time
	ingressPublicEnabled bool
	egressAllowlist      []networkHostMatcher
	egressAllowedSchemes map[string]struct{}
	ingressException     networkException
	egressException      networkException
}

type networkException struct {
	Active       bool
	ID           string
	Owner        string
	Reason       string
	Scope        string
	RollbackPlan string
	Expiry       time.Time
	scopeMatcher []networkHostMatcher
}

type networkHostMatcher struct {
	exact  string
	suffix string
}

type networkPolicyConfigError struct {
	policyClass string
	reasonClass string
	message     string
}

func (e *networkPolicyConfigError) Error() string {
	return e.message
}

func loadNetworkPolicyFromEnv() (networkPolicy, error) {
	ingressEnabled, err := parseOptionalBoolEnv(envNetworkPublicIngressEnabled, false, "ingress")
	if err != nil {
		return networkPolicy{}, err
	}

	egressAllowlist, err := parseHostMatchers(os.Getenv(envNetworkEgressAllowlist), "egress")
	if err != nil {
		return networkPolicy{}, err
	}
	egressSchemes, err := parseAllowedSchemes(os.Getenv(envNetworkEgressAllowedSchemes), "egress")
	if err != nil {
		return networkPolicy{}, err
	}

	ingressException, err := parseNetworkExceptionFromEnv("NETWORK_INGRESS_EXCEPTION", "ingress")
	if err != nil {
		return networkPolicy{}, err
	}
	egressException, err := parseNetworkExceptionFromEnv("NETWORK_EGRESS_EXCEPTION", "egress")
	if err != nil {
		return networkPolicy{}, err
	}

	return networkPolicy{
		now:                  time.Now,
		ingressPublicEnabled: ingressEnabled,
		egressAllowlist:      egressAllowlist,
		egressAllowedSchemes: egressSchemes,
		ingressException:     ingressException,
		egressException:      egressException,
	}, nil
}

func networkPolicyErrorLabels(err error) (string, string) {
	var cfgErr *networkPolicyConfigError
	if errors.As(err, &cfgErr) {
		return cfgErr.policyClass, cfgErr.reasonClass
	}
	return "ingress", "invalid_configuration"
}

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

func parseOptionalBoolEnv(name string, defaultValue bool, policyClass string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, &networkPolicyConfigError{
			policyClass: policyClass,
			reasonClass: "invalid_configuration",
			message:     fmt.Sprintf("%s must be a boolean value", name),
		}
	}
}

func parseAllowedSchemes(raw string, policyClass string) (map[string]struct{}, error) {
	result := map[string]struct{}{}
	for _, token := range strings.Split(raw, ",") {
		scheme := strings.ToLower(strings.TrimSpace(token))
		if scheme == "" {
			continue
		}
		if !isSchemeToken(scheme) {
			return nil, &networkPolicyConfigError{
				policyClass: policyClass,
				reasonClass: "invalid_configuration",
				message:     "NETWORK_EGRESS_ALLOWED_SCHEMES contains invalid scheme token",
			}
		}
		result[scheme] = struct{}{}
	}
	if len(result) == 0 {
		result["tcp"] = struct{}{}
	}
	return result, nil
}

func isSchemeToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		isLetter := b >= 'a' && b <= 'z'
		isDigit := b >= '0' && b <= '9'
		if i == 0 {
			if !isLetter {
				return false
			}
			continue
		}
		if !isLetter && !isDigit && b != '+' && b != '-' && b != '.' {
			return false
		}
	}
	return true
}

func parseNetworkExceptionFromEnv(prefix, policyClass string) (networkException, error) {
	active, err := parseOptionalBoolEnv(prefix+"_ACTIVE", false, policyClass)
	if err != nil {
		return networkException{}, err
	}
	if !active {
		return networkException{}, nil
	}

	exception := networkException{
		Active:       true,
		ID:           strings.TrimSpace(os.Getenv(prefix + "_ID")),
		Owner:        strings.TrimSpace(os.Getenv(prefix + "_OWNER")),
		Reason:       strings.TrimSpace(os.Getenv(prefix + "_REASON")),
		Scope:        strings.TrimSpace(os.Getenv(prefix + "_SCOPE")),
		RollbackPlan: strings.TrimSpace(os.Getenv(prefix + "_ROLLBACK_PLAN")),
	}

	expiryRaw := strings.TrimSpace(os.Getenv(prefix + "_EXPIRY"))
	if expiryRaw != "" {
		expiry, parseErr := time.Parse(time.RFC3339, expiryRaw)
		if parseErr != nil {
			return networkException{}, &networkPolicyConfigError{
				policyClass: policyClass,
				reasonClass: "invalid_configuration",
				message:     fmt.Sprintf("%s_EXPIRY must be RFC3339", prefix),
			}
		}
		exception.Expiry = expiry
	}

	missing := make([]string, 0, 5)
	if exception.Owner == "" {
		missing = append(missing, "owner")
	}
	if exception.Reason == "" {
		missing = append(missing, "reason")
	}
	if exception.Scope == "" {
		missing = append(missing, "scope")
	}
	if exception.Expiry.IsZero() {
		missing = append(missing, "expiry")
	}
	if exception.RollbackPlan == "" {
		missing = append(missing, "rollback_plan")
	}
	if len(missing) > 0 {
		return networkException{}, &networkPolicyConfigError{
			policyClass: policyClass,
			reasonClass: "missing_metadata",
			message:     fmt.Sprintf("%s requires metadata: %s", prefix, strings.Join(missing, ", ")),
		}
	}

	scopeMatchers, parseErr := parseHostMatchers(exception.Scope, policyClass)
	if parseErr != nil {
		return networkException{}, parseErr
	}
	if len(scopeMatchers) == 0 {
		return networkException{}, &networkPolicyConfigError{
			policyClass: policyClass,
			reasonClass: "missing_metadata",
			message:     fmt.Sprintf("%s_SCOPE cannot be empty", prefix),
		}
	}
	exception.scopeMatcher = scopeMatchers
	return exception, nil
}

func parseHostMatchers(raw string, policyClass string) ([]networkHostMatcher, error) {
	matchers := make([]networkHostMatcher, 0)
	for _, token := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(strings.ToLower(token))
		if trimmed == "" {
			continue
		}
		normalized := normalizeHost(trimmed)
		if normalized == "" {
			return nil, &networkPolicyConfigError{
				policyClass: policyClass,
				reasonClass: "invalid_configuration",
				message:     "host matcher contains empty token",
			}
		}

		matcher := networkHostMatcher{}
		switch {
		case strings.HasPrefix(normalized, "*."):
			suffix := "." + strings.TrimPrefix(normalized, "*.")
			if suffix == "." {
				return nil, &networkPolicyConfigError{
					policyClass: policyClass,
					reasonClass: "invalid_configuration",
					message:     "wildcard host matcher cannot be empty",
				}
			}
			matcher.suffix = suffix
		case strings.HasPrefix(normalized, "."):
			if normalized == "." {
				return nil, &networkPolicyConfigError{
					policyClass: policyClass,
					reasonClass: "invalid_configuration",
					message:     "suffix host matcher cannot be empty",
				}
			}
			matcher.suffix = normalized
		default:
			matcher.exact = normalized
		}
		matchers = append(matchers, matcher)
	}

	return matchers, nil
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

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	pgxCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: parse postgres dsn", config.ErrDependencyInit)
	}
	host := strings.TrimSpace(pgxCfg.ConnConfig.Host)
	if host == "" || pgxCfg.ConnConfig.Port == 0 {
		return "", fmt.Errorf("%w: invalid postgres probe address", config.ErrDependencyInit)
	}
	return net.JoinHostPort(host, strconv.Itoa(int(pgxCfg.ConnConfig.Port))), nil
}

func redisStartupProbeAddress(cfg config.RedisConfig) (string, error) {
	address := strings.TrimSpace(cfg.Addr)
	if address == "" {
		return "", fmt.Errorf("%w: empty redis probe address", config.ErrDependencyInit)
	}
	return address, nil
}

func mongoStartupProbeAddress(cfg config.MongoConfig) (string, error) {
	address, err := config.MongoProbeAddress(cfg.URI)
	if err != nil {
		return "", fmt.Errorf("%w: resolve mongo probe address", config.ErrDependencyInit)
	}
	return address, nil
}
