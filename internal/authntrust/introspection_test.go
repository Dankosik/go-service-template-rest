package authntrust_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

func TestValidIntrospectionEndpoint(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		raw  string
		want bool
	}{
		{raw: "https://idp.example.com/oauth/introspect", want: true},
		{raw: "https://idp.example.com/", want: true},
		{raw: "https://idp.example.com:8443/introspect", want: true},
		{raw: "HTTPS://idp.example.com/introspect", want: true},
		{raw: "https://idp.example.com/introspect?x=1"},
		{raw: "https://idp.example.com/introspect?"},
		{raw: "https://user:secret@idp.example.com/introspect"}, //nolint:gosec // Test fixture verifies user-info rejection; the value is not a credential.
		{raw: "https://idp.example.com/introspect#x"},
		{raw: "http://idp.example.com/introspect"},
		{raw: "  https://idp.example.com/introspect  "},
		{raw: ""},
	} {
		if got := authntrust.ValidIntrospectionEndpoint(testCase.raw); got != testCase.want {
			t.Errorf("ValidIntrospectionEndpoint(%q) = %v, want %v", testCase.raw, got, testCase.want)
		}
	}
}

func TestValidIntrospectionTargetClass(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		raw  string
		want bool
	}{
		{raw: authntrust.TargetClassExternalHTTPS, want: true},
		{raw: authntrust.TargetClassPrivateHTTPS, want: true},
		{raw: "public-https"},
		{raw: " external-https "},
		{raw: ""},
	} {
		if got := authntrust.ValidIntrospectionTargetClass(testCase.raw); got != testCase.want {
			t.Errorf("ValidIntrospectionTargetClass(%q) = %v, want %v", testCase.raw, got, testCase.want)
		}
	}
}

func TestIntrospectionTargetPolicyIssue(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name          string
		targetClass   string
		privateSuffix string
		want          authntrust.IntrospectionTargetIssue
	}{
		{name: "external", targetClass: authntrust.TargetClassExternalHTTPS, want: authntrust.IntrospectionTargetValid},
		{name: "private", targetClass: authntrust.TargetClassPrivateHTTPS, privateSuffix: "service.internal", want: authntrust.IntrospectionTargetValid},
		{name: "unknown class", targetClass: "external-https ", want: authntrust.IntrospectionTargetClassInvalid},
		{name: "private suffix missing", targetClass: authntrust.TargetClassPrivateHTTPS, want: authntrust.IntrospectionPrivateSuffixRequired},
		{name: "private whitespace suffix missing", targetClass: authntrust.TargetClassPrivateHTTPS, privateSuffix: " \t", want: authntrust.IntrospectionPrivateSuffixRequired},
		{name: "external suffix forbidden", targetClass: authntrust.TargetClassExternalHTTPS, privateSuffix: "service.internal", want: authntrust.IntrospectionPrivateSuffixForbidden},
		{name: "external whitespace suffix forbidden", targetClass: authntrust.TargetClassExternalHTTPS, privateSuffix: " ", want: authntrust.IntrospectionPrivateSuffixForbidden},
		{name: "private suffix whitespace retained", targetClass: authntrust.TargetClassPrivateHTTPS, privateSuffix: " service.internal ", want: authntrust.IntrospectionTargetValid},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := authntrust.IntrospectionTargetPolicyIssue(testCase.targetClass, testCase.privateSuffix); got != testCase.want {
				t.Fatalf("IntrospectionTargetPolicyIssue(%q, %q) = %v, want %v", testCase.targetClass, testCase.privateSuffix, got, testCase.want)
			}
		})
	}
}
