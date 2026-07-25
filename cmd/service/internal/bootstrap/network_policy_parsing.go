package bootstrap

import (
	"fmt"
	"os"
	"strings"
)

const (
	// envNetworkPublicIngressAcknowledged records that an operator consciously
	// decided whether a non-local wildcard listener is public. Bootstrap checks
	// that the decision was made, not what it was: reachability is enforced by
	// the deployment platform, which is the only layer that sees every
	// connection attempt.
	envNetworkPublicIngressAcknowledged = "NETWORK_PUBLIC_INGRESS_ACKNOWLEDGED"

	// envNetworkPublicIngressLegacy is the previous name. It is detected only
	// to produce an actionable rename error; it never satisfies the gate.
	envNetworkPublicIngressLegacy = "NETWORK_PUBLIC_INGRESS_ENABLED"
)

func loadNetworkPolicyFromEnv() (networkPolicy, error) {
	// The declared value is intentionally discarded: both "public" and
	// "private" are valid answers, and only the presence of an answer is a
	// precondition this process can enforce.
	_, acknowledged, err := parseOptionalBoolEnvWithExplicitValue(envNetworkPublicIngressAcknowledged, false)
	if err != nil {
		return networkPolicy{}, err
	}
	if !acknowledged && strings.TrimSpace(os.Getenv(envNetworkPublicIngressLegacy)) != "" {
		return networkPolicy{}, &networkPolicyConfigError{
			message: fmt.Sprintf(
				"%s was renamed to %s; the old name no longer acknowledges public ingress",
				envNetworkPublicIngressLegacy,
				envNetworkPublicIngressAcknowledged,
			),
		}
	}

	return networkPolicy{
		ingressAcknowledged: acknowledged,
	}, nil
}

func loadNetworkPolicy() networkPolicyLoadResult {
	policy, err := loadNetworkPolicyFromEnv()
	return networkPolicyLoadResult{
		policy: policy,
		err:    err,
	}
}

func parseOptionalBoolEnvWithExplicitValue(name string, defaultValue bool) (bool, bool, error) {
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
			message: fmt.Sprintf("%s must be a boolean value", name),
		}
	}
}
