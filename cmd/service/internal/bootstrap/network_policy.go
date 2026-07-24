package bootstrap

type networkPolicy struct {
	ingressPublicExplicitValue bool
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
