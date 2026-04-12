package bootstrap

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadNetworkPolicyFromEnvRequiresExceptionMetadata(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")

	_, err := loadNetworkPolicyFromEnv()
	if err == nil {
		t.Fatal("loadNetworkPolicyFromEnv() error = nil, want non-nil")
	}
	policyClass, reasonClass := networkPolicyErrorLabels(err)
	if policyClass != "ingress" {
		t.Fatalf("policyClass = %q, want %q", policyClass, "ingress")
	}
	if reasonClass != "missing_metadata" {
		t.Fatalf("reasonClass = %q, want %q", reasonClass, "missing_metadata")
	}
}

func TestLoadNetworkPolicyFromEnvPublicIngressExplicitValue(t *testing.T) {
	tests := []struct {
		name              string
		value             string
		set               bool
		wantEnabled       bool
		wantExplicitValue bool
		wantErr           string
	}{
		{
			name: "unset",
		},
		{
			name: "empty",
			set:  true,
		},
		{
			name:              "explicit false",
			value:             "false",
			set:               true,
			wantExplicitValue: true,
		},
		{
			name:              "explicit true",
			value:             "true",
			set:               true,
			wantEnabled:       true,
			wantExplicitValue: true,
		},
		{
			name:              "invalid",
			value:             "sometimes",
			set:               true,
			wantExplicitValue: true,
			wantErr:           "NETWORK_PUBLIC_INGRESS_ENABLED must be a boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv(envNetworkPublicIngressEnabled, tt.value)
			} else {
				unsetEnvForTest(t, envNetworkPublicIngressEnabled)
			}

			policy, err := loadNetworkPolicyFromEnv()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("loadNetworkPolicyFromEnv() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadNetworkPolicyFromEnv() error = %v, want %q detail", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
			}
			if policy.ingressPublicExplicitValue != tt.wantExplicitValue {
				t.Fatalf("ingressPublicExplicitValue = %v, want %v", policy.ingressPublicExplicitValue, tt.wantExplicitValue)
			}
			if policy.ingressPublicEnabled != tt.wantEnabled {
				t.Fatalf("ingressPublicEnabled = %v, want %v", policy.ingressPublicEnabled, tt.wantEnabled)
			}
		})
	}
}

func TestNetworkPolicyEnforceIngressRequiresDeclarationForNonLocalWildcardBind(t *testing.T) {
	wildcardAddrs := []string{
		":8080",
		"0.0.0.0:8080",
		"[::]:8080",
	}

	for _, addr := range wildcardAddrs {
		t.Run(addr, func(t *testing.T) {
			t.Setenv(envNetworkPublicIngressEnabled, "")

			policy, err := loadNetworkPolicyFromEnv()
			if err != nil {
				t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
			}
			policy = policy.withIngressExposure("prod", addr)

			err = policy.EnforceIngress()
			if err == nil {
				t.Fatal("EnforceIngress() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), envNetworkPublicIngressEnabled) {
				t.Fatalf("EnforceIngress() error = %v, want %s detail", err, envNetworkPublicIngressEnabled)
			}
		})
	}
}

func TestNetworkPolicyEnforceIngressAllowsExplicitPrivateIngressAssertion(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "false")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy = policy.withIngressExposure("prod", ":8080")

	if err := policy.EnforceIngress(); err != nil {
		t.Fatalf("EnforceIngress() error = %v, want nil", err)
	}
}

func TestNetworkPolicyEnforceIngressAllowsMissingDeclarationForLocalWildcardBind(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy = policy.withIngressExposure("local", ":8080")

	if err := policy.EnforceIngress(); err != nil {
		t.Fatalf("EnforceIngress() error = %v, want nil", err)
	}
}

func TestNetworkPolicyEnforceIngressFailClosedWithoutException(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "true")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	err = policy.EnforceIngress()
	if err == nil {
		t.Fatal("EnforceIngress() error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceIngressAllowsActiveException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-1")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-load-test")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(2*time.Hour).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	if err := policy.EnforceIngress(); err != nil {
		t.Fatalf("EnforceIngress() error = %v, want nil", err)
	}
}

func TestNetworkPolicyIngressExceptionScopeIsMetadataOnly(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-metadata-scope")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", ".")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(2*time.Hour).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v, want nil for metadata-only ingress scope", err)
	}
	policy.now = func() time.Time { return now }

	if got := policy.ingressException.Scope; got != "." {
		t.Fatalf("ingress exception Scope = %q, want audit metadata", got)
	}
	if got := len(policy.ingressException.scopeMatcher); got != 0 {
		t.Fatalf("ingress exception scopeMatcher len = %d, want 0", got)
	}
	if err := policy.EnforceIngress(); err != nil {
		t.Fatalf("EnforceIngress() error = %v, want nil", err)
	}
}

func TestNetworkPolicyEnforceIngressRejectsExpiredException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-expired")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(-5*time.Minute).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	err = policy.EnforceIngress()
	if err == nil {
		t.Fatal("EnforceIngress() error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceEgressTargetDeniesPublicHost(t *testing.T) {
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	err = policy.EnforceEgressTarget("api.example.com:443", "tcp")
	if err == nil {
		t.Fatal("EnforceEgressTarget() error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceEgressTargetAllowsPrivateAndAllowlistedHosts(t *testing.T) {
	t.Setenv(envNetworkEgressAllowlist, "api.example.com,*.allowed.example")
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	privateTargetErr := policy.EnforceEgressTarget("10.0.0.12:5432", "tcp")
	if privateTargetErr != nil {
		t.Fatalf("EnforceEgressTarget(private) error = %v, want nil", privateTargetErr)
	}
	allowlistedErr := policy.EnforceEgressTarget("api.example.com:443", "tcp")
	if allowlistedErr != nil {
		t.Fatalf("EnforceEgressTarget(allowlisted exact) error = %v, want nil", allowlistedErr)
	}
	allowlistedSuffixErr := policy.EnforceEgressTarget("service.allowed.example:443", "tcp")
	if allowlistedSuffixErr != nil {
		t.Fatalf("EnforceEgressTarget(allowlisted suffix) error = %v, want nil", allowlistedSuffixErr)
	}
	allowlistedApexErr := policy.EnforceEgressTarget("allowed.example:443", "tcp")
	if allowlistedApexErr == nil {
		t.Fatal("EnforceEgressTarget(wildcard apex) error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceEgressTargetLeadingDotMatchesApexAndSubdomain(t *testing.T) {
	t.Setenv(envNetworkEgressAllowlist, ".allowed.example")
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	if err := policy.EnforceEgressTarget("allowed.example:443", "tcp"); err != nil {
		t.Fatalf("EnforceEgressTarget(leading-dot apex) error = %v, want nil", err)
	}
	if err := policy.EnforceEgressTarget("service.allowed.example:443", "tcp"); err != nil {
		t.Fatalf("EnforceEgressTarget(leading-dot subdomain) error = %v, want nil", err)
	}
}

func TestNetworkPolicyEnforceEgressTargetDeniesSchemeOutsideAllowlist(t *testing.T) {
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	err = policy.EnforceEgressTarget("10.0.0.12:5432", "udp")
	if err == nil {
		t.Fatal("EnforceEgressTarget() error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceEgressTargetInvalidTargetPreservesLocalReason(t *testing.T) {
	t.Setenv(envNetworkEgressAllowedSchemes, "tcp")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	tests := []struct {
		name       string
		target     string
		wantReason string
	}{
		{
			name:       "empty target",
			wantReason: "target is empty",
		},
		{
			name:       "empty host",
			target:     ":443",
			wantReason: "host is empty",
		},
		{
			name:       "malformed host port",
			target:     "api.example.com:443:extra",
			wantReason: "target must be host:port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.EnforceEgressTarget(tt.target, "tcp")
			if err == nil {
				t.Fatal("EnforceEgressTarget() error = nil, want non-nil")
			}
			if !errors.Is(err, errDependencyInit) {
				t.Fatalf("EnforceEgressTarget() error = %v, want wrapped %v", err, errDependencyInit)
			}
			if !errorChainContainsExactMessage(err, tt.wantReason) {
				t.Fatalf("EnforceEgressTarget() error chain = %v, want wrapped %q", err, tt.wantReason)
			}
		})
	}
}

func TestNetworkPolicyEgressExceptionScopeMatchesPublicTarget(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ID", "ex-egress-scope")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_REASON", "temporary-upstream-debug")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_SCOPE", "api.exception.example")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_EXPIRY", now.Add(2*time.Hour).Format(time.RFC3339))
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-egress-exception")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	if got := len(policy.egressException.scopeMatcher); got == 0 {
		t.Fatal("egress exception scopeMatcher len = 0, want host matcher")
	}
	if err := policy.EnforceEgressTarget("api.exception.example:443", "tcp"); err != nil {
		t.Fatalf("EnforceEgressTarget(exception scope) error = %v, want nil", err)
	}
	if err := policy.EnforceEgressTarget("api.other.example:443", "tcp"); err == nil {
		t.Fatal("EnforceEgressTarget(outside exception scope) error = nil, want non-nil")
	}
}

func TestNetworkPolicyEgressExceptionRejectsInvalidScopeMatcher(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ID", "ex-egress-invalid-scope")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_REASON", "temporary-upstream-debug")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_SCOPE", ".")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_EXPIRY", now.Add(2*time.Hour).Format(time.RFC3339))
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-egress-exception")

	_, err := loadNetworkPolicyFromEnv()
	if err == nil {
		t.Fatal("loadNetworkPolicyFromEnv() error = nil, want invalid egress exception scope")
	}
	policyClass, reasonClass := networkPolicyErrorLabels(err)
	if policyClass != "egress" {
		t.Fatalf("policyClass = %q, want %q", policyClass, "egress")
	}
	if reasonClass != "invalid_configuration" {
		t.Fatalf("reasonClass = %q, want %q", reasonClass, "invalid_configuration")
	}
}

func TestNetworkPolicyValidateEgressExceptionStateRejectsExpiredException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ID", "ex-egress-expired")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_REASON", "temporary-upstream-debug")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_SCOPE", "api.example.com")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_EXPIRY", now.Add(-5*time.Minute).Format(time.RFC3339))
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-egress-exception")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	err = policy.ValidateEgressExceptionState()
	if err == nil {
		t.Fatal("ValidateEgressExceptionState() error = nil, want non-nil")
	}
}

func TestNetworkPolicyValidateIngressRuntimeRejectsExpiredException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-expired-runtime")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(-5*time.Minute).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	err = policy.ValidateIngressRuntime()
	if err == nil {
		t.Fatal("ValidateIngressRuntime() error = nil, want non-nil")
	}
}

func TestNetworkPolicyEnforceEgressTargetDeniesSingleLabelHostByDefault(t *testing.T) {
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	err = policy.EnforceEgressTarget("redis:6379", "tcp")
	if err == nil {
		t.Fatal("EnforceEgressTarget(single label) error = nil, want non-nil")
	}
}

func unsetEnvForTest(t *testing.T, name string) {
	t.Helper()

	previous, hadPrevious := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", name, err)
	}
	t.Cleanup(func() {
		var err error
		if hadPrevious {
			err = os.Setenv(name, previous)
		} else {
			err = os.Unsetenv(name)
		}
		if err != nil {
			t.Errorf("restore env %q error = %v", name, err)
		}
	})
}

func errorChainContainsExactMessage(err error, want string) bool {
	if err == nil {
		return false
	}
	if err.Error() == want {
		return true
	}
	type multiUnwrapper interface {
		Unwrap() []error
	}
	if wrapped, ok := err.(multiUnwrapper); ok {
		for _, child := range wrapped.Unwrap() {
			if errorChainContainsExactMessage(child, want) {
				return true
			}
		}
		return false
	}
	type singleUnwrapper interface {
		Unwrap() error
	}
	if wrapped, ok := err.(singleUnwrapper); ok {
		return errorChainContainsExactMessage(wrapped.Unwrap(), want)
	}
	return false
}
