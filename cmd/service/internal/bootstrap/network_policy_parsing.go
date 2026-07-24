package bootstrap

import (
	"fmt"
	"os"
	"strings"
)

const envNetworkPublicIngressEnabled = "NETWORK_PUBLIC_INGRESS_ENABLED"

func loadNetworkPolicyFromEnv() (networkPolicy, error) {
	_, ingressExplicitValue, err := parseOptionalBoolEnvWithExplicitValue(envNetworkPublicIngressEnabled, false)
	if err != nil {
		return networkPolicy{}, err
	}

	return networkPolicy{
		ingressPublicExplicitValue: ingressExplicitValue,
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
