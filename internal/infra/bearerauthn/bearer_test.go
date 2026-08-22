package bearerauthn

import (
	"strings"
	"testing"
)

func TestBearerTokenGrammar(t *testing.T) {
	t.Parallel()

	exact := strings.Repeat("x", MaxTokenBytes)
	for _, testCase := range []struct {
		name   string
		values []string
		want   string
		kind   Kind
	}{
		{name: "missing", kind: KindMissing},
		{name: "duplicate", values: []string{"Bearer one", "Bearer two"}, kind: KindMalformed},
		{name: "wrong scheme", values: []string{"Basic token"}, kind: KindMalformed},
		{name: "leading space", values: []string{" Bearer token"}, kind: KindMalformed},
		{name: "empty token", values: []string{"Bearer "}, kind: KindMalformed},
		{name: "internal space", values: []string{"Bearer tok en"}, kind: KindMalformed},
		{name: "comma", values: []string{"Bearer token,extra"}, kind: KindMalformed},
		{name: "oversize", values: []string{"Bearer " + strings.Repeat("x", MaxTokenBytes+1)}, kind: KindOversize},
		{name: "accepted", values: []string{"Bearer token"}, want: "token"},
		{name: "scheme case", values: []string{"bearer token"}, want: "token"},
		{name: "size boundary", values: []string{"Bearer " + exact}, want: exact},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := bearerToken(testCase.values)
			if testCase.kind != 0 {
				requireKind(t, err, testCase.kind)
				if got != "" {
					t.Fatalf("token = %q, want empty", got)
				}
				return
			}
			if err != nil || got != testCase.want {
				t.Fatalf("bearerToken() = %q, %v; want %q, nil", got, err, testCase.want)
			}
		})
	}
}
