package oauthintrospection

import (
	"errors"
	"strings"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

// Policy is the immutable introspection trust configuration.
type Policy struct {
	issuer        string
	audience      string
	endpoint      string
	targetClass   string
	privateSuffix string
	clientID      string
	clientSecret  string
}

// PolicyInput carries the configured trust values one deployment holds.
//
// The exhaustruct_v5 enforce pattern for this type in .golangci.yml is load-bearing:
// a field added here fails lint at every production call site that does not set
// it.
type PolicyInput struct {
	Issuer        string
	Audience      string
	Endpoint      string
	TargetClass   string
	PrivateSuffix string
	ClientID      string
	ClientSecret  string
}

// NewPolicy validates and copies the complete trust policy.
//
// Credential components are copied exactly. The configuration loader applies
// the same rules earlier: a deployment value refused here should already have
// stopped the process at configuration load.
func NewPolicy(input PolicyInput) (Policy, error) {
	if !authntrust.ValidIssuerURL(input.Issuer) {
		return Policy{}, errors.New("authn issuer must be an absolute HTTPS URL without user info, query, or fragment")
	}
	if strings.TrimSpace(input.Audience) == "" || strings.TrimSpace(input.Audience) != input.Audience {
		return Policy{}, errors.New("authn audience cannot be empty")
	}
	if !authntrust.ValidIntrospectionEndpoint(input.Endpoint) {
		return Policy{}, errors.New("authn introspection endpoint must be an absolute HTTPS URL without user info, query, or fragment")
	}
	if !authntrust.ValidIntrospectionTargetClass(input.TargetClass) {
		return Policy{}, errors.New("authn introspection target class must be one of external-https or private-https")
	}
	if input.TargetClass == authntrust.TargetClassPrivateHTTPS {
		if strings.TrimSpace(input.PrivateSuffix) == "" {
			return Policy{}, errors.New("authn introspection private host suffix is required for private-https")
		}
	} else if input.PrivateSuffix != "" {
		return Policy{}, errors.New("authn introspection private host suffix is forbidden for external-https")
	}
	if input.ClientID == "" {
		return Policy{}, errors.New("authn introspection client id cannot be empty")
	}
	if input.ClientSecret == "" {
		return Policy{}, errors.New("authn introspection client secret cannot be empty")
	}

	return Policy{
		issuer:        input.Issuer,
		audience:      input.Audience,
		endpoint:      input.Endpoint,
		targetClass:   input.TargetClass,
		privateSuffix: input.PrivateSuffix,
		clientID:      input.ClientID,
		clientSecret:  input.ClientSecret,
	}, nil
}
