// profile:authn-oidc-jwt:start
package authntrust_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

func TestValidTokenProfile(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		value string
		want  bool
	}{
		{value: authntrust.TokenProfileResourceServer, want: true},
		{value: authntrust.TokenProfileRFC9068, want: true},
		{value: "", want: false},
		{value: "strict", want: false},
	} {
		if got := authntrust.ValidTokenProfile(testCase.value); got != testCase.want {
			t.Fatalf("ValidTokenProfile(%q) = %v, want %v", testCase.value, got, testCase.want)
		}
	}
}

// profile:authn-oidc-jwt:end
