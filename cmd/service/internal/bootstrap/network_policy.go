package bootstrap

import (
	"time"
)

type networkPolicy struct {
	now                        func() time.Time
	ingressPublicEnabled       bool
	ingressPublicDeclared      bool
	ingressDeclarationRequired bool
	egressAllowlist            []networkHostMatcher
	egressAllowedSchemes       map[string]struct{}
	ingressException           networkException
	egressException            networkException
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
	exact       string
	suffix      string
	includeApex bool
}

type networkPolicyConfigError struct {
	policyClass string
	reasonClass string
	message     string
}

func (e *networkPolicyConfigError) Error() string {
	return e.message
}
