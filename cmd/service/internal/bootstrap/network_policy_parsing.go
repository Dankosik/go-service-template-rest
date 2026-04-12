package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	envNetworkPublicIngressEnabled = "NETWORK_PUBLIC_INGRESS_ENABLED"
	envNetworkEgressAllowlist      = "NETWORK_EGRESS_ALLOWLIST"
	envNetworkEgressAllowedSchemes = "NETWORK_EGRESS_ALLOWED_SCHEMES"
)

func loadNetworkPolicyFromEnv() (networkPolicy, error) {
	ingressEnabled, ingressExplicitValue, err := parseOptionalBoolEnvWithExplicitValue(envNetworkPublicIngressEnabled, false, "ingress")
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
		now:                        time.Now,
		ingressPublicEnabled:       ingressEnabled,
		ingressPublicExplicitValue: ingressExplicitValue,
		egressAllowlist:            egressAllowlist,
		egressAllowedSchemes:       egressSchemes,
		ingressException:           ingressException,
		egressException:            egressException,
	}, nil
}

func networkPolicyErrorLabels(err error) (string, string) {
	var cfgErr *networkPolicyConfigError
	if errors.As(err, &cfgErr) {
		return cfgErr.policyClass, cfgErr.reasonClass
	}
	return "ingress", "invalid_configuration"
}

func parseOptionalBoolEnv(name string, defaultValue bool, policyClass string) (bool, error) {
	value, _, err := parseOptionalBoolEnvWithExplicitValue(name, defaultValue, policyClass)
	return value, err
}

func parseOptionalBoolEnvWithExplicitValue(name string, defaultValue bool, policyClass string) (bool, bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, false, nil
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true, true, nil
	case "0", "false", "no", "off":
		return false, true, nil
	default:
		return false, true, &networkPolicyConfigError{
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
			matcher.includeApex = true
		default:
			matcher.exact = normalized
		}
		matchers = append(matchers, matcher)
	}

	return matchers, nil
}
