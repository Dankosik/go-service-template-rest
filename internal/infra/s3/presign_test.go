package s3

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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
	issuedAt, err := time.Parse("20060102T150405Z", query.Get("X-Amz-Date"))
	if err != nil {
		t.Fatal(err)
	}
	seconds, err := strconv.Atoi(query.Get("X-Amz-Expires"))
	if err != nil || time.Duration(seconds)*time.Second != lifetime {
		t.Fatalf("X-Amz-Expires = %q, want %s", query.Get("X-Amz-Expires"), lifetime)
	}
	if !result.ExpiresAt.Equal(issuedAt.UTC().Add(time.Duration(seconds) * time.Second)) {
		t.Fatalf("ExpiresAt = %s, want signed expiry", result.ExpiresAt)
	}
}
