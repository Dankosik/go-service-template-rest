package oauthintrospection

import (
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

func TestIntrospectionPolicyAdmission(t *testing.T) {
	t.Parallel()

	valid := PolicyInput{
		Issuer:       testIssuer,
		Audience:     testAudience,
		Endpoint:     "https://idp.example.com/oauth/introspect",
		TargetClass:  authntrust.TargetClassExternalHTTPS,
		ClientID:     testClientID,
		ClientSecret: testSecret,
	}
	if _, err := NewPolicy(valid); err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}

	for _, testCase := range []struct {
		name   string
		mutate func(*PolicyInput)
		want   string
	}{
		{name: "issuer query", mutate: func(in *PolicyInput) { in.Issuer = "https://issuer.example.com?x=1" }, want: "issuer"},
		{name: "empty audience", mutate: func(in *PolicyInput) { in.Audience = "" }, want: "audience"},
		{name: "endpoint user info", mutate: func(in *PolicyInput) { in.Endpoint = "https://user:x@idp.example.com/introspect" }, want: "endpoint"},
		{name: "endpoint query", mutate: func(in *PolicyInput) { in.Endpoint = "https://idp.example.com/introspect?x=1" }, want: "endpoint"},
		{name: "unknown class", mutate: func(in *PolicyInput) { in.TargetClass = "public-https" }, want: "target class"},
		{name: "private suffix missing", mutate: func(in *PolicyInput) {
			in.TargetClass = authntrust.TargetClassPrivateHTTPS
			in.Endpoint = "https://idp.service.internal/introspect"
		}, want: "private host suffix"},
		{name: "private suffix forbidden", mutate: func(in *PolicyInput) { in.PrivateSuffix = ".internal" }, want: "private host suffix"},
		{name: "empty client id", mutate: func(in *PolicyInput) { in.ClientID = "" }, want: "client id"},
		{name: "empty client secret", mutate: func(in *PolicyInput) { in.ClientSecret = "" }, want: "client secret"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			input := valid
			testCase.mutate(&input)
			_, err := NewPolicy(input)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("NewPolicy() error = %v, want %q", err, testCase.want)
			}
			if strings.Contains(err.Error(), testSecret) || strings.Contains(err.Error(), testClientID) {
				t.Fatalf("NewPolicy() disclosed a credential: %v", err)
			}
		})
	}

	private := valid
	private.TargetClass = authntrust.TargetClassPrivateHTTPS
	private.Endpoint = "https://idp.service.internal/introspect"
	private.PrivateSuffix = "service.internal"
	if _, err := NewPolicy(private); err != nil {
		t.Fatalf("NewPolicy(private) error = %v", err)
	}

	reserved := valid
	reserved.ClientID = "id with space+:%"
	reserved.ClientSecret = "secret with space+:%"
	policy, err := NewPolicy(reserved)
	if err != nil {
		t.Fatalf("NewPolicy(reserved) error = %v", err)
	}
	if policy.clientID != reserved.ClientID || policy.clientSecret != reserved.ClientSecret {
		t.Fatal("NewPolicy() trimmed a credential")
	}
}
