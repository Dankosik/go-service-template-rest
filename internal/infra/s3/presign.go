package s3

import (
	"context"
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
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key),
	}, func(options *awss3.PresignOptions) { options.Expires = lifetime })
	if err != nil {
		return result, operationError(operationPresign, err, false)
	}
	if err := effective.Err(); err != nil {
		return result, c.admissionError(err)
	}
	result, ok := validatedPresignedGET(request, c.transport.endpoint, key)
	if !ok {
		return result, objectstorage.NewError(objectstorage.KindInternal)
	}
	return result, nil
}

func validatedPresignedGET(request *signerV4.PresignedHTTPRequest, endpoint url.URL, key string) (objectstorage.PresignedGET, bool) {
	if request == nil || request.Method != http.MethodGet {
		return objectstorage.PresignedGET{}, false
	}
	signed, err := url.Parse(request.URL)
	if err != nil || signed.User != nil || signed.Scheme != endpoint.Scheme || signed.Host != endpoint.Host {
		return objectstorage.PresignedGET{}, false
	}
	path, err := url.PathUnescape(signed.EscapedPath())
	if err != nil || path != "/"+key {
		return objectstorage.PresignedGET{}, false
	}
	query := signed.Query()
	if query.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" || query.Get("X-Amz-Credential") == "" || query.Get("X-Amz-Signature") == "" || !validSignedHeaders(request.SignedHeader, query.Get("X-Amz-SignedHeaders"), endpoint.Host) {
		return objectstorage.PresignedGET{}, false
	}
	issuedAt, err := time.Parse("20060102T150405Z", query.Get("X-Amz-Date"))
	if err != nil {
		return objectstorage.PresignedGET{}, false
	}
	seconds, err := strconv.Atoi(query.Get("X-Amz-Expires"))
	if err != nil || seconds < 1 || seconds > int(maximumPresignLifetime/time.Second) {
		return objectstorage.PresignedGET{}, false
	}
	return objectstorage.PresignedGET{
		Method:    http.MethodGet,
		URL:       request.URL,
		Headers:   request.SignedHeader.Clone(),
		ExpiresAt: issuedAt.UTC().Add(time.Duration(seconds) * time.Second),
	}, true
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
	for _, name := range signedNames {
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
