package oidcjwt

// Proof that the two owners of the authn trust rules agree on which deployments
// they admit. The rules themselves belong to internal/authntrust and are tested
// there; what each owner does with them alone is in policy_test.go and
// internal/config's own validation tests.

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
)

// authnConfigCase is one deployment's trust values and the answer both owners of
// the trust rules have to give for it.
type authnConfigCase struct {
	name       string
	issuer     string
	audience   string
	cidrs      string
	acceptable bool
}

// TestPolicyRulesMatchConfigValidation holds [NewPolicy] and internal/config's
// validateAuthnConfig to a single answer over one corpus.
//
// Both now call internal/authntrust rather than each carrying a predicate, so
// this no longer guards two copies against drifting apart. It guards the half
// that sharing cannot fix: that each owner still asks. A validateAuthnConfig
// that stopped running the shared CIDR parse, or a [NewPolicy] that stopped
// checking the issuer, would let a deployment hold a value the other refuses —
// at startup, in production — and no compiler or linter says otherwise.
//
// loadAuthnConfig runs the real loader rather than validateAuthnConfig directly,
// so a rule that is applied but never reached from a configuration load fails
// here as well.
//
// The check lives here rather than in internal/config because the import only
// works in this direction: feature_packages_no_adapters exempts internal/infra,
// while config_no_runtime_owners covers internal/config's tests too.
//
// It can only compare the values the corpus below varies, so the corpus itself
// has to be held to [PolicyInput]. requireRejectedCasePerPolicyInputField does
// that, and owns why a configured value added without a case here would
// otherwise be reported as agreed.
func TestPolicyRulesMatchConfigValidation(t *testing.T) {
	const (
		goodIssuer   = "https://issuer.example.com"
		goodAudience = "service-api"
		goodCIDRs    = "127.0.0.0/8,::1/128"
	)
	cases := []authnConfigCase{
		{name: "canonical", issuer: goodIssuer, audience: goodAudience, cidrs: goodCIDRs, acceptable: true},
		{name: "issuer with path", issuer: "https://issuer.example.com/realms/main", audience: goodAudience, cidrs: goodCIDRs, acceptable: true},
		{name: "issuer with port", issuer: "https://issuer.example.com:8443", audience: goodAudience, cidrs: goodCIDRs, acceptable: true},
		{name: "single CIDR", issuer: goodIssuer, audience: goodAudience, cidrs: "10.0.0.0/8", acceptable: true},

		// One entry per term in ValidIssuerURL, so an owner that stops applying
		// the shared predicate fails here rather than at a deployment's startup.
		{name: "http issuer", issuer: "http://issuer.example.com", audience: goodAudience, cidrs: goodCIDRs},
		{name: "relative issuer", issuer: "/realms/main", audience: goodAudience, cidrs: goodCIDRs},
		{name: "opaque issuer", issuer: "https:issuer.example.com", audience: goodAudience, cidrs: goodCIDRs},
		{name: "issuer without host", issuer: "https://", audience: goodAudience, cidrs: goodCIDRs},
		{name: "issuer with user info", issuer: "https://user:secret@issuer.example.com", audience: goodAudience, cidrs: goodCIDRs},
		{name: "issuer with query", issuer: "https://issuer.example.com?next=a", audience: goodAudience, cidrs: goodCIDRs},
		{name: "issuer with forced query", issuer: "https://issuer.example.com?", audience: goodAudience, cidrs: goodCIDRs},
		{name: "issuer with fragment", issuer: "https://issuer.example.com#frag", audience: goodAudience, cidrs: goodCIDRs},
		{name: "unparseable issuer", issuer: "https://issuer.example.com/%zz", audience: goodAudience, cidrs: goodCIDRs},
		{name: "empty issuer", issuer: "", audience: goodAudience, cidrs: goodCIDRs},

		{name: "empty audience", issuer: goodIssuer, audience: "", cidrs: goodCIDRs},
		{name: "blank audience", issuer: goodIssuer, audience: "   ", cidrs: goodCIDRs},

		{name: "no CIDRs", issuer: goodIssuer, audience: goodAudience, cidrs: ""},
		{name: "CIDR without prefix length", issuer: goodIssuer, audience: goodAudience, cidrs: "127.0.0.1"},
		{name: "duplicate CIDR", issuer: goodIssuer, audience: goodAudience, cidrs: "127.0.0.0/8,127.0.0.1/8"},
	}

	requireRejectedCasePerPolicyInputField(t, cases)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, policyErr := NewPolicy(policyInputOf(testCase))
			configErr := loadAuthnConfig(t, testCase)

			if (policyErr == nil) != testCase.acceptable {
				t.Fatalf("NewPolicy() error = %v, want acceptable = %v", policyErr, testCase.acceptable)
			}
			if (configErr == nil) != (policyErr == nil) {
				t.Fatalf(
					"trust rules disagree: NewPolicy() error = %v, config load error = %v",
					policyErr, configErr,
				)
			}
		})
	}
}

func policyInputOf(testCase authnConfigCase) PolicyInput {
	return PolicyInput{
		Issuer:            testCase.issuer,
		Audience:          testCase.audience,
		TrustedProxyCIDRs: testCase.cidrs,
	}
}

// requireRejectedCasePerPolicyInputField fails unless the corpus holds, for every
// [PolicyInput] field, a refused case that varies that field away from the
// canonical acceptable row.
//
// This is what makes a configured trust value reach internal/config rather than
// only the composition root. The exhaustruct entry for [PolicyInput] already
// fails the build until a new field is set at every production call site, but a
// field wired that far and no further left the comparison below running over a
// value the corpus never varied — reporting an agreement it had not checked.
// Requiring a *refused* case rather than merely a varying one is the point: only
// a value one owner must reject can show that validateAuthnConfig enforces the
// rule [NewPolicy] enforces.
//
// Cases vary one field at a time, so a field's rejected case is found by
// comparing against the canonical row rather than by naming it.
func requireRejectedCasePerPolicyInputField(t *testing.T, cases []authnConfigCase) {
	t.Helper()
	if len(cases) == 0 || !cases[0].acceptable {
		t.Fatal("the corpus must open with an acceptable case; every other case is read against it")
	}
	canonical := reflect.ValueOf(policyInputOf(cases[0]))
	for index := range canonical.NumField() {
		rejected := slices.ContainsFunc(cases, func(testCase authnConfigCase) bool {
			candidate := reflect.ValueOf(policyInputOf(testCase))
			differences := 0
			for fieldIndex := range canonical.NumField() {
				if !candidate.Field(fieldIndex).Equal(canonical.Field(fieldIndex)) {
					differences++
				}
			}
			return !testCase.acceptable &&
				differences == 1 &&
				!candidate.Field(index).Equal(canonical.Field(index))
		})
		if !rejected {
			t.Errorf(
				"the corpus has no refused case that varies PolicyInput.%s, so this test reports an "+
					"agreement it never checked for that value; add a case both NewPolicy and "+
					"internal/config's validateAuthnConfig must reject",
				canonical.Type().Field(index).Name,
			)
		}
	}
}

// loadAuthnConfig runs one corpus entry through the real configuration loader
// and returns whether it survived validation.
func loadAuthnConfig(t *testing.T, testCase authnConfigCase) error {
	t.Helper()
	configtest.IsolateEnv(t)
	t.Setenv("APP__AUTHN__ISSUER", testCase.issuer)
	t.Setenv("APP__AUTHN__AUDIENCE", testCase.audience)
	t.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", testCase.cidrs)

	if _, _, err := config.LoadDetailed(config.LoadOptions{}); err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	return nil
}
