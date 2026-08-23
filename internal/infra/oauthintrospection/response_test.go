package oauthintrospection

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
)

func TestResponseEnvelopeAdmission(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	valid := activeJSON("subject-1", "client-1")
	if _, err := admitResponse([]byte(valid), policy, testNow); err != nil {
		t.Fatalf("valid envelope error = %v", err)
	}
	for _, body := range []string{
		"",
		`[]`,
		`"active"`,
		`null`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":1,"active":true}`,
		`{"active":true,"unknown":1,"unknown":2}`,
		`{"active":true}{"active":true}`,
		`{"active":true} 0`,
		`{`,
		`{"iss":"` + testIssuer + `"}`,
		`{"active":"true"}`,
		`{"active":1}`,
		`{"active":null}`,
	} {
		if _, err := admitResponse([]byte(body), policy, testNow); err == nil {
			t.Fatalf("admitResponse(%q) succeeded", body)
		} else {
			requireKind(t, err, bearerauthn.KindUnavailable)
		}
	}
}

func TestInactiveResponseShortCircuits(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	canary := "inactive-canary-value"
	for _, body := range []string{
		`{"active":false}`,
		`{"active":false,"iss":1,"exp":"nope","aud":null}`,
		`{"active":false,"token":"` + canary + `","secret":"` + canary + `"}`,
		`{"active":false,"exp":1,"nbf":9999999999,"iss":"https://other.example","aud":"other"}`,
		`{"active":false,"sub":"s","client_id":"c"}`,
	} {
		result, err := admitResponse([]byte(body), policy, testNow)
		requireKind(t, err, bearerauthn.KindInvalid)
		if result.Principal.Subject != "" || result.Principal.ClientID != "" || !result.ExpiresAt.IsZero() {
			t.Fatalf("inactive result = %+v", result)
		}
		if strings.Contains(err.Error(), canary) {
			t.Fatalf("inactive error disclosed canary: %v", err)
		}
	}
}

func TestActiveResponseStructure(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	exp := strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	valid := `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + exp + `,"sub":"subject-1"}`
	result, err := admitResponse([]byte(valid), policy, testNow)
	if err != nil || result.ExpiresAt.Unix() != testNow.Add(time.Hour).Unix() || result.Principal.Subject != "subject-1" {
		t.Fatalf("valid control = %+v, %v", result, err)
	}

	for _, body := range []string{
		`{"active":true,"aud":"` + testAudience + `","exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":1,"aud":"` + testAudience + `","exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":"","aud":"` + testAudience + `","exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":[1], "exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":[null],"exp":` + exp + `,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":1.5,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":1e2,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":1e309,"sub":"s"}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + exp + `}`,
		`{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + exp + `,"sub":1}`,
	} {
		if _, err := admitResponse([]byte(body), policy, testNow); err == nil {
			t.Fatalf("admitResponse(%s) succeeded", body)
		} else {
			requireKind(t, err, bearerauthn.KindUnavailable)
		}
	}
}

func TestActiveResponseLocalTrust(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	expOK := strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	expInside := strconv.FormatInt(testNow.Add(-bearerauthn.ClockSkew).Unix(), 10)
	expOutside := strconv.FormatInt(testNow.Add(-bearerauthn.ClockSkew-time.Second).Unix(), 10)
	nbfInside := strconv.FormatInt(testNow.Add(bearerauthn.ClockSkew).Unix(), 10)
	nbfOutside := strconv.FormatInt(testNow.Add(bearerauthn.ClockSkew+time.Second).Unix(), 10)

	for _, testCase := range []struct {
		body string
		kind bearerauthn.Kind
	}{
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + expOK + `,"sub":"s"}`, kind: 0},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":["other","` + testAudience + `"],"exp":` + expOK + `,"sub":"s"}`, kind: 0},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + expInside + `,"sub":"s"}`, kind: 0},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + expOK + `,"nbf":` + nbfInside + `,"sub":"s"}`, kind: 0},
		{body: `{"active":true,"iss":"https://other.example","aud":"` + testAudience + `","exp":` + expOK + `,"sub":"s"}`, kind: bearerauthn.KindInvalid},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"https://other.example","exp":` + expOK + `,"sub":"s"}`, kind: bearerauthn.KindInvalid},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":["https://other.example"],"exp":` + expOK + `,"sub":"s"}`, kind: bearerauthn.KindInvalid},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + expOutside + `,"sub":"s"}`, kind: bearerauthn.KindInvalid},
		{body: `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + expOK + `,"nbf":` + nbfOutside + `,"sub":"s"}`, kind: bearerauthn.KindInvalid},
		{body: `{"active":true,"iss": 1,"aud":"` + testAudience + `","exp":` + expOK + `,"sub":"s"}`, kind: bearerauthn.KindUnavailable},
	} {
		result, err := admitResponse([]byte(testCase.body), policy, testNow)
		if testCase.kind == 0 {
			if err != nil || result.Principal.Subject != "s" {
				t.Fatalf("admitResponse() = %+v, %v", result, err)
			}
			continue
		}
		requireKind(t, err, testCase.kind)
		if result.Principal.Subject != "" {
			t.Fatalf("principal published on failure: %+v", result)
		}
	}
}

func TestPrincipalNormalizationAndMinimization(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	exp := strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	base := func(extra string) string {
		return `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience + `","exp":` + exp + extra + `}`
	}

	if got := activeJSON("subject-only", ""); !strings.Contains(got, "subject-only") {
		t.Fatalf("activeJSON omitted subject: %s", got)
	}
	subjectOnly, err := admitResponse([]byte(base(`,"sub":"subject-1"`)), policy, testNow)
	if err != nil || subjectOnly.Principal.Issuer != testIssuer || subjectOnly.Principal.Subject != "subject-1" || subjectOnly.Principal.ClientID != "" {
		t.Fatalf("subject-only = %+v, %v", subjectOnly, err)
	}

	clientOnly, err := admitResponse([]byte(base(`,"client_id":"client-1"`)), policy, testNow)
	if err != nil || clientOnly.Principal.ClientID != "client-1" || clientOnly.Principal.Subject != "" {
		t.Fatalf("client-only = %+v, %v", clientOnly, err)
	}

	both, err := admitResponse([]byte(base(`,"sub":"subject-1","client_id":"client-1"`)), policy, testNow)
	if err != nil || both.Principal.Subject != "subject-1" || both.Principal.ClientID != "client-1" {
		t.Fatalf("both = %+v, %v", both, err)
	}

	emptyOptional, err := admitResponse([]byte(base(`,"sub":"subject-1","client_id":""`)), policy, testNow)
	if err != nil || emptyOptional.Principal.ClientID != "" {
		t.Fatalf("empty optional = %+v, %v", emptyOptional, err)
	}

	if _, err := admitResponse([]byte(base(`,"sub":" subject-1"`)), policy, testNow); err == nil {
		t.Fatal("padded subject was admitted")
	} else {
		requireKind(t, err, bearerauthn.KindUnavailable)
	}

	sensitive, err := admitResponse([]byte(base(`,"sub":"subject-1","scope":"admin","username":"u","jti":"j","secret":"canary"`)), policy, testNow)
	if err != nil || sensitive.Principal.Subject != "subject-1" || sensitive.Principal.Scopes != nil {
		t.Fatalf("sensitive = %+v, %v", sensitive, err)
	}
}
