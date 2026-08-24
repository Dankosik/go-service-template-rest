package authntrust_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

func TestIntrospectionEndpointAndTargetClass(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		raw        string
		wantValid  bool
		wantClass  bool
		classValue string
	}{
		{name: "canonical endpoint", raw: "https://idp.example.com/oauth/introspect", wantValid: true},
		{name: "root path", raw: "https://idp.example.com/", wantValid: true},
		{name: "port", raw: "https://idp.example.com:8443/introspect", wantValid: true},
		{name: "uppercase scheme", raw: "HTTPS://idp.example.com/introspect", wantValid: true},
		{name: "query", raw: "https://idp.example.com/introspect?x=1"},
		{name: "forced query", raw: "https://idp.example.com/introspect?"},
		{name: "user info", raw: "https://user:secret@idp.example.com/introspect"},
		{name: "fragment", raw: "https://idp.example.com/introspect#x"},
		{name: "plaintext", raw: "http://idp.example.com/introspect"},
		{name: "surrounding space", raw: "  https://idp.example.com/introspect  "},
		{name: "empty", raw: ""},
		{name: "external class", classValue: authntrust.TargetClassExternalHTTPS, wantClass: true},
		{name: "private class", classValue: authntrust.TargetClassPrivateHTTPS, wantClass: true},
		{name: "unknown class", classValue: "public-https"},
		{name: "padded class", classValue: " external-https "},
		{name: "empty class", classValue: ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.classValue != "" || testCase.name == "empty class" || testCase.name == "unknown class" || testCase.name == "padded class" {
				if got := authntrust.ValidIntrospectionTargetClass(testCase.classValue); got != testCase.wantClass {
					t.Errorf("ValidIntrospectionTargetClass(%q) = %v, want %v", testCase.classValue, got, testCase.wantClass)
				}
				return
			}
			if got := authntrust.ValidIntrospectionEndpoint(testCase.raw); got != testCase.wantValid {
				t.Errorf("ValidIntrospectionEndpoint(%q) = %v, want %v", testCase.raw, got, testCase.wantValid)
			}
		})
	}
}
