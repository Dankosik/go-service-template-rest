package oidcjwt

// Proof for the two documents that describe this package: docs/authentication.md,
// which publishes a contract to operators, and doc.go, which is the extension
// manual the next maintainer reads first. Both make claims about code, and both
// are held to it here.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

func TestDocumentedTokenExample(t *testing.T) {
	guide := readAuthenticationGuide(t)
	for _, required := range []string{
		"AUTHN=oidc-jwt",
		"APP__AUTHN__ISSUER=https://",
		"APP__AUTHN__AUDIENCE=",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS=",
		"RS256 access token",
		"non-empty `sub`, `client_id`, and `jti`",
		"Integer, fractional, and exponent notation",
		"`reqctx.Principal{Issuer, Subject,\nClientID}`",
		strconv.Itoa(int(MaxTokenLifetime/time.Minute)) + "-minute maximum issued lifetime",
		"`Health/Check` remains public",
		"stream handlers must stop when their\nauthenticated context is done",
		"one-or-more ASCII spaces after the Bearer scheme",
		"bounded ±10% jitter",
		"## Bearer replay and revocation",
		"Sender-constrained access tokens",
		"provider-owned query is allowed on\n`jwks_uri`",
		"native gRPC is enabled with this profile, its server transport must be TLS",
		"There is no bypass, fake principal mode, or accept-all switch",
	} {
		if !strings.Contains(guide, required) {
			t.Fatalf("authentication guide is missing %q", required)
		}
	}

	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	signed := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	principal, err := verifier.verify(t.Context(), signed, transportHTTP)
	if err != nil ||
		principal.Issuer != testIssuer ||
		principal.Subject != "opaque-subject" ||
		principal.ClientID != "client-1" {
		t.Fatalf("documented signed-token flow = (%+v, %v), want verified issuer, subject, and client ID", principal, err)
	}
}

// TestDocumentedMetricReasonsMatchTheGuide holds every authn.reason label this
// package can emit to the closed set docs/authentication.md publishes.
func TestDocumentedMetricReasonsMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	labels := []string{verificationReason(context.Canceled)}
	for _, detail := range kindDetails[1:] {
		labels = append(labels, detail.reason)
	}

	for _, label := range labels {
		if !strings.Contains(guide, "`"+label+"`") {
			t.Errorf(
				"authentication guide does not publish the %q reason label; "+
					"add it to the authn.reason set operators are told to expect",
				label,
			)
		}
	}
}

// TestDocumentedTriggersMatchTheGuide holds every authn.refresh.trigger value
// this package can emit to the closed set docs/authentication.md publishes.
//
// It is [TestDocumentedMetricReasonsMatchTheGuide] for the other vocabulary that
// reaches operators. Making refreshTrigger a named type closed the compiler half
// — a fourth trigger now fails rateLimited under the exhaustive linter — but
// nothing made it reach the guide, so a trigger added and not published left
// operators reading a list that no longer described the service.
//
// The declarations are read from the source rather than listed here, so there is
// no second enumeration of refreshTrigger to keep in step.
func TestDocumentedTriggersMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	triggers := declaredConstants(t, ".", "refreshTrigger")

	// A walk that found nothing would pass the loop below while covering nothing.
	// The rate-limited trigger is the cheapest witness that it read the block.
	if !slices.Contains(triggers, string(triggerKeyMiss)) {
		t.Fatalf("the refreshTrigger walk collected %v and never reached the rate-limited trigger", triggers)
	}

	for _, trigger := range triggers {
		if !strings.Contains(guide, "`"+trigger+"`") {
			t.Errorf(
				"authentication guide does not publish the %q refresh trigger; "+
					"add it to the authn.refresh.trigger set operators are told to expect",
				trigger,
			)
		}
	}
}

func TestDocumentedProviderFailureReasonsMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	reasons := append(declaredConstants(t, ".", "providerError"), "unknown")
	if !slices.Contains(reasons, string(errProviderTransport)) {
		t.Fatalf("the providerError walk collected %v and never reached transport", reasons)
	}
	for _, reason := range reasons {
		if !strings.Contains(guide, "`"+reason+"`") {
			t.Errorf("authentication guide does not publish the %q provider failure reason", reason)
		}
	}
}

func readAuthenticationGuide(t *testing.T) string {
	t.Helper()
	document, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "authentication.md"))
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	return string(document)
}

// declaredConstants returns the string value of every constant in dir declared
// with the named type. Reading them out of the source is what keeps a test that
// checks a published vocabulary from holding a second copy of it.
func declaredConstants(t *testing.T, dir, typeName string) []string {
	t.Helper()
	values := make([]string, 0)
	packagetest.EachGoFile(t, dir, parser.SkipObjectResolution, func(fileName string, parsed *ast.File, _ *token.FileSet) {
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, isValue := node.(*ast.ValueSpec)
			if !isValue {
				return true
			}
			// A grouped const block repeats the type on each spec that states one,
			// so an untyped or differently typed neighbour is skipped rather than
			// inherited.
			if named, isIdent := spec.Type.(*ast.Ident); !isIdent || named.Name != typeName {
				return true
			}
			for _, value := range spec.Values {
				literal, isLiteral := value.(*ast.BasicLit)
				if !isLiteral || literal.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("unquote %s in %s: %v", literal.Value, fileName, err)
				}
				values = append(values, unquoted)
			}
			return true
		})
	})
	slices.Sort(values)
	return slices.Compact(values)
}
