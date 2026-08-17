package s3

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

type scriptedDoer struct {
	body   string
	closed bool
}

func (d *scriptedDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: closeRecorder{ReadCloser: io.NopCloser(strings.NewReader(d.body)), closed: &d.closed}}, nil
}

type closeRecorder struct {
	io.ReadCloser

	closed *bool
}

func (r closeRecorder) Close() error {
	*r.closed = true
	return r.ReadCloser.Close() //nolint:wrapcheck // The test wrapper preserves the source close error.
}

func TestTransportRefusesAlternateAuthority(t *testing.T) {
	t.Parallel()
	endpoint, err := url.Parse("https://bucket.s3.us-east-1.amazonaws.com")
	if err != nil {
		t.Fatal(err)
	}
	transport := transport{base: &scriptedDoer{}, endpoint: *endpoint, controlLimit: 4, objectLimit: 8}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://other.example/object", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.Do(request)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, httpclient.ErrTargetDenied) {
		t.Fatalf("Do() error = %v, want ErrTargetDenied", err)
	}
}

func TestTransportBoundsControlAndObjectBodies(t *testing.T) {
	t.Parallel()
	endpoint, err := url.Parse("https://bucket.s3.us-east-1.amazonaws.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		method string
		limit  int64
		body   string
	}{
		{name: "control", method: http.MethodPost, limit: 4, body: "12345"},
		{name: "object", method: http.MethodGet, limit: 8, body: "123456789"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			doer := &scriptedDoer{body: test.body}
			transport := transport{base: doer, endpoint: *endpoint, controlLimit: 4, objectLimit: 8}
			request, err := http.NewRequestWithContext(t.Context(), test.method, endpoint.String()+"/object", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			response, err := transport.Do(request)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			_, err = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if _, ok := errors.AsType[*bodyTooLargeError](err); !ok {
				t.Fatalf("ReadAll() error = %v, want body limit error", err)
			}
			if !doer.closed {
				t.Fatal("oversized body was not closed")
			}
		})
	}
}

func TestSDKGetObjectUsesObjectBodyLimit(t *testing.T) {
	t.Parallel()
	endpoint, err := url.Parse("https://bucket.s3.us-east-1.amazonaws.com")
	if err != nil {
		t.Fatal(err)
	}
	doer := &scriptedDoer{body: "12345678"}
	transport := &transport{base: doer, endpoint: *endpoint, controlLimit: 4, objectLimit: 8}
	client := awss3.New(awss3.Options{
		Credentials:        credentials.NewStaticCredentialsProvider("key", "secret", ""),
		Region:             "us-east-1",
		Retryer:            aws.NopRetryer{},
		HTTPClient:         transport,
		EndpointResolverV2: fixedResolver{endpoint: *endpoint, region: "us-east-1", bucket: "bucket"},
	})
	response, err := client.GetObject(t.Context(), &awss3.GetObjectInput{Bucket: aws.String("bucket"), Key: aws.String("object")})
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("GetObject body error = %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("GetObject body close error = %v", err)
	}
	if got := string(body); got != "12345678" {
		t.Fatalf("GetObject body = %q, want object body through object limit", got)
	}
}
