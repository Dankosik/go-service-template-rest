package bootstrap

import (
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

func TestLoadNetworkPolicyFromEnvDistinguishesMissingPublicIngressDeclaration(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	if policy.ingressPublicDeclared {
		t.Fatalf("ingressPublicDeclared = true, want false for missing %s", envNetworkPublicIngressEnabled)
	}
	if policy.ingressPublicEnabled {
		t.Fatalf("ingressPublicEnabled = true, want false for missing %s", envNetworkPublicIngressEnabled)
	}

	t.Setenv(envNetworkPublicIngressEnabled, "false")
	policy, err = loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() explicit false error = %v", err)
	}
	if !policy.ingressPublicDeclared {
		t.Fatalf("ingressPublicDeclared = false, want true for explicit false")
	}
	if policy.ingressPublicEnabled {
		t.Fatalf("ingressPublicEnabled = true, want false for explicit false")
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

func TestNetworkPolicyEmitEgressExceptionStateRejectsExpiredException(t *testing.T) {
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

	err = policy.EmitEgressExceptionState()
	if err == nil {
		t.Fatal("EmitEgressExceptionState() error = nil, want non-nil")
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
