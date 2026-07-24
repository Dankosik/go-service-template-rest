package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestLoadNetworkPolicyFromEnvPublicIngressDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		set          bool
		wantExplicit bool
		wantErr      string
	}{
		{name: "unset"},
		{name: "empty", set: true},
		{name: "explicit false", value: "false", set: true, wantExplicit: true},
		{name: "explicit true", value: "true", set: true, wantExplicit: true},
		{
			name:         "invalid",
			value:        "sometimes",
			set:          true,
			wantExplicit: true,
			wantErr:      "NETWORK_PUBLIC_INGRESS_ENABLED must be a boolean value",
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
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadNetworkPolicyFromEnv() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
			}
			if policy.ingressPublicExplicitValue != tt.wantExplicit {
				t.Fatalf("ingressPublicExplicitValue = %v, want %v", policy.ingressPublicExplicitValue, tt.wantExplicit)
			}
		})
	}
}

func TestNetworkPolicyEnforceIngress(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		addr      string
		declared  string
		wantError bool
	}{
		{name: "local wildcard", env: "local", addr: ":8080"},
		{name: "production loopback", env: "prod", addr: "127.0.0.1:8080"},
		{name: "production public", env: "prod", addr: ":8080", declared: "true"},
		{name: "production private", env: "prod", addr: "0.0.0.0:8080", declared: "false"},
		{name: "production missing", env: "prod", addr: "[::]:8080", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envNetworkPublicIngressEnabled, tt.declared)
			policy, err := loadNetworkPolicyFromEnv()
			if err != nil {
				t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
			}
			err = policy.withIngressExposure(tt.env, tt.addr).EnforceIngress()
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), envNetworkPublicIngressEnabled) {
					t.Fatalf("EnforceIngress() error = %v, want missing declaration", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("EnforceIngress() error = %v", err)
			}
		})
	}
}

func unsetEnvForTest(t *testing.T, name string) {
	t.Helper()

	previous, hadPrevious := os.LookupEnv(name)
	if hadPrevious {
		t.Setenv(name, previous)
	}
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", name, err)
	}
}
