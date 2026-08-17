package s3

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

const maximumPresignLifetime = 7 * 24 * time.Hour

// PresignGET issues one bounded SigV4 GET URL without provider I/O.
//
//nolint:contextcheck,wrapcheck // Preserve the span rooted in ctx and the closed object-storage errors.
func (c *Client) PresignGET(ctx context.Context, key string, lifetime time.Duration) (result objectstorage.PresignedGET, err error) {
	call := c.telemetry.begin(ctx, telemetryOperationPresign)
	defer func() { call.finish(err, 0) }()
	if err := objectstorage.ValidateKey(key); err != nil {
		return result, err
	}
	if lifetime < time.Second || lifetime > c.config.MaxPresignLifetime || lifetime > maximumPresignLifetime || lifetime%time.Second != 0 {
		return result, objectstorage.NewError(objectstorage.KindInvalid)
	}
	effective, release, err := c.admit(ctx, call)
	if err != nil {
		return result, c.admissionError(err)
	}
	defer release()
	if err := effective.Err(); err != nil {
		return result, c.admissionError(err)
	}

	request, err := awss3.NewPresignClient(c.sdk).PresignGetObject(effective, &awss3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket), ExpectedBucketOwner: c.expectedBucketOwner(), Key: aws.String(key),
	}, func(options *awss3.PresignOptions) { options.Expires = lifetime })
	if err != nil {
		return result, operationError(c.config.Provider, operationPresign, err, nil)
	}
	if err := effective.Err(); err != nil {
		return result, c.admissionError(err)
	}
	result, ok := validatedPresignedGET(request, c.transport.endpoint, key, lifetime, c.config)
	if !ok {
		return result, objectstorage.NewError(objectstorage.KindInternal)
	}
	return result, nil
}

func validatedPresignedGET(request *signerV4.PresignedHTTPRequest, endpoint url.URL, key string, lifetime time.Duration, cfg Config) (objectstorage.PresignedGET, bool) {
	signed, ok := validPresignedTarget(request, endpoint, key)
	if !ok {
		return objectstorage.PresignedGET{}, false
	}
	query := signed.Query()
	algorithm, algorithmOK := singleQueryValue(query, "X-Amz-Algorithm")
	credential, credentialOK := singleQueryValue(query, "X-Amz-Credential")
	signature, signatureOK := singleQueryValue(query, "X-Amz-Signature")
	signedHeaders, signedHeadersOK := singleQueryValue(query, "X-Amz-SignedHeaders")
	date, dateOK := singleQueryValue(query, "X-Amz-Date")
	expires, expiresOK := singleQueryValue(query, "X-Amz-Expires")
	if !algorithmOK || algorithm != "AWS4-HMAC-SHA256" || !credentialOK || !signatureOK || !signedHeadersOK || !dateOK || !expiresOK ||
		!validSignature(signature) || !validSignedHeaders(request.SignedHeader, signedHeaders, endpoint.Host) {
		return objectstorage.PresignedGET{}, false
	}
	issuedAt, err := time.Parse("20060102T150405Z", date)
	if err != nil || !validCredentialScope(credential, cfg, issuedAt) || !validSecurityToken(query, cfg.SessionToken) {
		return objectstorage.PresignedGET{}, false
	}
	if !validExpectedOwner(query, cfg) {
		return objectstorage.PresignedGET{}, false
	}
	seconds, ok := validPresignExpiry(expires, lifetime)
	if !ok {
		return objectstorage.PresignedGET{}, false
	}
	return objectstorage.PresignedGET{
		Method:             http.MethodGet,
		URL:                request.URL,
		Headers:            request.SignedHeader.Clone(),
		SignatureExpiresAt: issuedAt.UTC().Add(time.Duration(seconds) * time.Second),
	}, true
}

func validPresignedTarget(request *signerV4.PresignedHTTPRequest, endpoint url.URL, key string) (*url.URL, bool) {
	if request == nil || request.Method != http.MethodGet {
		return nil, false
	}
	signed, err := url.Parse(request.URL)
	if err != nil || signed.User != nil || signed.Fragment != "" || signed.Scheme != endpoint.Scheme || signed.Host != endpoint.Host {
		return nil, false
	}
	path, err := url.PathUnescape(signed.EscapedPath())
	return signed, err == nil && path == "/"+key
}

func validExpectedOwner(query url.Values, cfg Config) bool {
	owner, present := singleQueryValue(query, "x-amz-expected-bucket-owner")
	return cfg.Provider == ProviderAmazonS3 && present && owner == cfg.ExpectedBucketOwner ||
		cfg.Provider == ProviderCloudflare && !present
}

func validPresignExpiry(value string, lifetime time.Duration) (int, bool) {
	seconds, err := strconv.Atoi(value)
	return seconds, err == nil && seconds >= 1 && seconds <= int(maximumPresignLifetime/time.Second) && time.Duration(seconds)*time.Second == lifetime
}

func singleQueryValue(query url.Values, name string) (string, bool) {
	values, ok := query[name]
	returnValue := ""
	if ok && len(values) == 1 {
		returnValue = values[0]
	}
	return returnValue, ok && len(values) == 1
}

func validCredentialScope(credential string, cfg Config, issuedAt time.Time) bool {
	prefix := cfg.AccessKeyID + "/"
	if !strings.HasPrefix(credential, prefix) {
		return false
	}
	scope := strings.Split(strings.TrimPrefix(credential, prefix), "/")
	return len(scope) == 4 && scope[0] == issuedAt.UTC().Format("20060102") && scope[1] == cfg.Region && scope[2] == "s3" && scope[3] == "aws4_request"
}

func validSecurityToken(query url.Values, expected string) bool {
	actual, present := singleQueryValue(query, "X-Amz-Security-Token")
	if expected == "" {
		return !present
	}
	return present && actual == expected
}

func validSignature(signature string) bool {
	decoded, err := hex.DecodeString(signature)
	return err == nil && len(decoded) == 32
}

func validSignedHeaders(headers http.Header, signed, host string) bool {
	returned := make(map[string][]string, len(headers))
	for name, values := range headers {
		name = strings.ToLower(name)
		if name == "" || name == "authorization" || len(values) == 0 {
			return false
		}
		returned[name] = values
	}

	signedNames := strings.Split(signed, ";")
	seenHost := false
	previous := ""
	for _, name := range signedNames {
		if name == "" || name <= previous {
			return false
		}
		previous = name
		if name == "host" {
			values, ok := returned[name]
			if seenHost || !ok || len(values) != 1 || values[0] != host {
				return false
			}
			seenHost = true
			delete(returned, name)
			continue
		}
		if _, ok := returned[name]; !ok {
			return false
		}
		delete(returned, name)
	}
	return seenHost && len(returned) == 0
}
