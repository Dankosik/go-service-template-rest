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
		{raw: "https://user:secret@idp.example.com/introspect"},
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
