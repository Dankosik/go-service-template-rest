package s3

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestPresignGETIsBoundedAndSecret(t *testing.T) {
	var requests int
	cfg := validConfig(ProviderAmazonS3)
	cfg.MaxPresignLifetime = maximumPresignLifetime
	client := scriptedClientWithConfig(t, cfg, func(*http.Request) (*http.Response, error) {
		requests++
		return nil, http.ErrAbortHandler
	})
	for _, lifetime := range []time.Duration{0, time.Second, client.config.MaxPresignLifetime, client.config.MaxPresignLifetime + time.Second} {
		result, err := client.PresignGET(context.Background(), "path/object", lifetime)
		if lifetime < time.Second || lifetime > client.config.MaxPresignLifetime {
			if got := objectstorage.Kind(err); got != objectstorage.KindInvalid {
				t.Fatalf("Kind(PresignGET(%s)) = %q, want invalid", lifetime, got)
			}
			if strings.Contains(err.Error(), "test-secret-key") || strings.Contains(err.Error(), "X-Amz-Signature") {
				t.Fatalf("invalid presign error leaked bearer material: %q", err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("PresignGET(%s) error = %v", lifetime, err)
		}
		assertPresignedGET(t, client, result, lifetime)
	}
	if requests != 0 {
		t.Fatalf("presign made %d HTTP requests, want 0", requests)
	}
	r2 := scriptedClientWithConfig(t, validConfig(ProviderCloudflare), func(*http.Request) (*http.Response, error) {
		t.Fatal("R2 presign made an HTTP request")
		return nil, http.ErrAbortHandler
	})
	result, err := r2.PresignGET(context.Background(), "path/object", time.Second)
	if err != nil {
		t.Fatalf("R2 PresignGET() error = %v", err)
	}
	assertPresignedGET(t, r2, result, time.Second)
}

func TestPresignedGETValidatorRejectsScopeAndLifetimeDrift(t *testing.T) {
	cfg := validConfig(ProviderAmazonS3)
	client := scriptedClientWithConfig(t, cfg, func(*http.Request) (*http.Response, error) {
		t.Fatal("presign validator made an HTTP request")
		return nil, http.ErrAbortHandler
	})
	const lifetime = time.Minute
	result, err := client.PresignGET(t.Context(), "path/object", lifetime)
	if err != nil {
		t.Fatal(err)
	}
	baseline := &signerV4.PresignedHTTPRequest{URL: result.URL, Method: result.Method, SignedHeader: result.Headers}
	if _, ok := validatedPresignedGET(baseline, client.transport.endpoint, "path/object", lifetime, cfg); !ok {
		t.Fatal("baseline presigned request was rejected")
	}

	for _, test := range []struct {
		name   string
		mutate func(*url.URL, url.Values, http.Header)
	}{
		{name: "duplicate algorithm", mutate: func(_ *url.URL, query url.Values, _ http.Header) { query.Add("X-Amz-Algorithm", "AWS4-HMAC-SHA256") }},
		{name: "access key", mutate: func(_ *url.URL, query url.Values, _ http.Header) {
			parts := strings.Split(query.Get("X-Amz-Credential"), "/")
			parts[0] = "another-access-key"
			query.Set("X-Amz-Credential", strings.Join(parts, "/"))
		}},
		{name: "scope date", mutate: func(_ *url.URL, query url.Values, _ http.Header) {
			parts := strings.Split(query.Get("X-Amz-Credential"), "/")
			parts[1] = "19700101"
			query.Set("X-Amz-Credential", strings.Join(parts, "/"))
		}},
		{name: "scope region", mutate: func(_ *url.URL, query url.Values, _ http.Header) {
			parts := strings.Split(query.Get("X-Amz-Credential"), "/")
			parts[2] = "us-west-2"
			query.Set("X-Amz-Credential", strings.Join(parts, "/"))
		}},
		{name: "scope service", mutate: func(_ *url.URL, query url.Values, _ http.Header) {
			parts := strings.Split(query.Get("X-Amz-Credential"), "/")
			parts[3] = "ec2"
			query.Set("X-Amz-Credential", strings.Join(parts, "/"))
		}},
		{name: "lifetime", mutate: func(_ *url.URL, query url.Values, _ http.Header) { query.Set("X-Amz-Expires", "59") }},
		{name: "session token", mutate: func(_ *url.URL, query url.Values, _ http.Header) { query.Del("X-Amz-Security-Token") }},
		{name: "expected owner duplicate", mutate: func(_ *url.URL, query url.Values, _ http.Header) {
			query.Add("x-amz-expected-bucket-owner", cfg.ExpectedBucketOwner)
		}},
		{name: "signature", mutate: func(_ *url.URL, query url.Values, _ http.Header) { query.Set("X-Amz-Signature", "not-hex") }},
		{name: "fragment", mutate: func(target *url.URL, _ url.Values, _ http.Header) { target.Fragment = "fragment" }},
		{name: "signed headers order", mutate: func(_ *url.URL, query url.Values, _ http.Header) { query.Set("X-Amz-SignedHeaders", "host;host") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			target, parseErr := url.Parse(result.URL)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			query := target.Query()
			headers := result.Headers.Clone()
			test.mutate(target, query, headers)
			target.RawQuery = query.Encode()
			request := &signerV4.PresignedHTTPRequest{URL: target.String(), Method: result.Method, SignedHeader: headers}
			if _, ok := validatedPresignedGET(request, client.transport.endpoint, "path/object", lifetime, cfg); ok {
				t.Fatal("mutated presigned request was accepted")
			}
		})
	}
}

func assertPresignedGET(t *testing.T, client *Client, result objectstorage.PresignedGET, lifetime time.Duration) {
	t.Helper()
	if result.Method != http.MethodGet {
		t.Fatalf("method = %q, want GET", result.Method)
	}
	signed, err := url.Parse(result.URL)
	if err != nil {
		t.Fatal(err)
	}
	if signed.Host != client.transport.endpoint.Host || signed.Path != "/path/object" {
		t.Fatalf("presigned target = %s%s", signed.Host, signed.Path)
	}
	query := signed.Query()
	for _, name := range []string{"X-Amz-Algorithm", "X-Amz-Credential", "X-Amz-Date", "X-Amz-Expires", "X-Amz-SignedHeaders", "X-Amz-Signature"} {
		if query.Get(name) == "" {
			t.Fatalf("presigned query missing %s", name)
		}
	}
	if query.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" || !validSignedHeaders(result.Headers, query.Get("X-Amz-SignedHeaders"), client.transport.endpoint.Host) {
		t.Fatalf("presigned headers = %#v, signed = %q", result.Headers, query.Get("X-Amz-SignedHeaders"))
	}
	owner := query.Get("x-amz-expected-bucket-owner")
	if client.config.Provider == ProviderAmazonS3 && owner != client.config.ExpectedBucketOwner {
		t.Fatalf("presigned expected owner = %q, want %q", owner, client.config.ExpectedBucketOwner)
	}
	if client.config.Provider == ProviderCloudflare && owner != "" {
		t.Fatalf("R2 presign included expected owner %q", owner)
	}
	issuedAt, err := time.Parse("20060102T150405Z", query.Get("X-Amz-Date"))
	if err != nil {
		t.Fatal(err)
	}
	seconds, err := strconv.Atoi(query.Get("X-Amz-Expires"))
	if err != nil || time.Duration(seconds)*time.Second != lifetime {
		t.Fatalf("X-Amz-Expires = %q, want %s", query.Get("X-Amz-Expires"), lifetime)
	}
	if !result.SignatureExpiresAt.Equal(issuedAt.UTC().Add(time.Duration(seconds) * time.Second)) {
		t.Fatalf("SignatureExpiresAt = %s, want signed expiry", result.SignatureExpiresAt)
	}
	if !validCredentialScope(query.Get("X-Amz-Credential"), client.config, issuedAt) || !validSecurityToken(query, client.config.SessionToken) || !validSignature(query.Get("X-Amz-Signature")) {
		t.Fatal("presigned SigV4 scope, session token, or signature shape is invalid")
	}
}
