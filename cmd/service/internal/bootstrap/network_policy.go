package bootstrap

type networkPolicy struct {
	// ingressAcknowledged records that an operator answered the public-ingress
	// question, not which answer they gave.
	ingressAcknowledged        bool
	ingressDeclarationRequired bool
}

type networkPolicyLoadResult struct {
	policy networkPolicy
	err    error
}

type networkPolicyConfigError struct {
	message string
}

func (e *networkPolicyConfigError) Error() string {
	return e.message
}
