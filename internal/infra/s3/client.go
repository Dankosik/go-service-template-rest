package s3

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

// Client is the local, credential-free construction boundary for one S3 tuple.
type Client struct {
	config    Config
	sdk       *awss3.Client
	client    *httpclient.Client
	transport *transport
	tokens    chan struct{}
	telemetry *telemetry
	roots     *x509.CertPool
	closed    sync.Once
}

func withReadRetry(options *awss3.Options) {
	options.Retryer = retry.NewStandard(func(standard *retry.StandardOptions) {
		standard.MaxAttempts = 3
		standard.MaxBackoff = 100 * time.Millisecond
		standard.Retryables = []retry.IsErrorRetryable{retry.IsErrorRetryableFunc(readRetryable)}
	})
}

func (c *Client) expectedBucketOwner() *string {
	if c.config.Provider != ProviderAmazonS3 {
		return nil
	}
	return aws.String(c.config.ExpectedBucketOwner)
}

func (c *Client) admit(ctx context.Context, call *operationCall) (context.Context, func(), error) {
	if ctx == nil {
		return nil, nil, errors.New("S3 operation context is required")
	}
	select {
	case c.tokens <- struct{}{}:
		call.admitted()
	default:
		call.rejected()
		return nil, nil, errors.New("S3 adapter admission is full")
	}

	deadline := time.Now().Add(c.config.MaxOperationDuration)
	if callerDeadline, ok := ctx.Deadline(); ok && callerDeadline.Before(deadline) {
		deadline = callerDeadline
	}
	effective, cancel := context.WithDeadline(ctx, deadline)
	return effective, func() {
		cancel()
		<-c.tokens
		call.released()
	}, nil
}

// New validates and constructs one immutable adapter without DNS or provider I/O.
func New(cfg Config) (*Client, error) {
	return newClient(cfg, productionImageRootSource)
}

func newClient(cfg Config, rootSource imageRootSource) (*Client, error) {
	endpoint, err := cfg.validate()
	if err != nil {
		return nil, err
	}
	roots, err := loadImageRootBundle(rootSource)
	if err != nil {
		return nil, err
	}
	finalEndpoint := *endpoint
	finalEndpoint.Host = cfg.Bucket + "." + endpoint.Host

	client, err := httpclient.New(httpclient.Config{
		DependencyName:         "object-storage",
		BaseURL:                finalEndpoint.String(),
		TargetClass:            httpclient.ExternalHTTPS,
		OneAttempt:             true,
		RootCAs:                roots.pool,
		DisableInstrumentation: true,
		RequestTimeout:         cfg.MaxOperationDuration,
		ResponseHeaderTimeout:  cfg.MaxOperationDuration,
		MaxResponseHeaderBytes: cfg.MaxResponseHeaderBytes,
		MaxResponseBodyBytes:   cfg.MaxObjectBytes + 1,
		MaxConnsPerHost:        cfg.MaxActiveOperations,
		MaxIdleConnsPerHost:    0,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("build S3 adapter transport: %w", err)
	}

	resolver := fixedResolver{endpoint: finalEndpoint, region: cfg.Region, bucket: cfg.Bucket}
	adapterTransport := &transport{base: client, endpoint: finalEndpoint, controlLimit: cfg.MaxControlResponseBytes, objectLimit: cfg.MaxObjectBytes + 1}
	options := awss3.Options{
		Credentials:                    credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		Region:                         cfg.Region,
		Retryer:                        aws.NopRetryer{},
		RequestChecksumCalculation:     aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation:     aws.ResponseChecksumValidationWhenRequired,
		DisableClockSkewCorrection:     true,
		DisableMultiRegionAccessPoints: true,
		DisableS3ExpressSessionAuth:    aws.Bool(true),
		UseARNRegion:                   false,
		UseAccelerate:                  false,
		UseDualstack:                   false,
		UsePathStyle:                   false,
		HTTPClient:                     adapterTransport,
		EndpointResolverV2:             resolver,
	}

	telemetry, err := newTelemetry()
	if err != nil {
		return nil, fmt.Errorf("build S3 adapter telemetry: %w", err)
	}

	return &Client{
		config:    cfg,
		sdk:       awss3.New(options),
		client:    client,
		transport: adapterTransport,
		tokens:    make(chan struct{}, cfg.MaxActiveOperations),
		telemetry: telemetry,
		roots:     roots.pool,
	}, nil
}

// Close releases idle transport resources. Active operations retain their own
// caller contexts and are never given a detached shutdown path.
func (c *Client) Close() {
	c.closed.Do(c.client.CloseIdleConnections)
}

type fixedResolver struct {
	endpoint url.URL
	region   string
	bucket   string
}

func (r fixedResolver) ResolveEndpoint(_ context.Context, params awss3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	if params.Region == nil || *params.Region != r.region || params.Bucket == nil || *params.Bucket != r.bucket ||
		(params.ForcePathStyle != nil && *params.ForcePathStyle) ||
		(params.Accelerate != nil && *params.Accelerate) ||
		(params.UseDualStack != nil && *params.UseDualStack) ||
		(params.UseFIPS != nil && *params.UseFIPS) ||
		(params.UseArnRegion != nil && *params.UseArnRegion) {
		return smithyendpoints.Endpoint{}, errors.New("resolve S3 endpoint: alternate authority is denied")
	}
	return smithyendpoints.Endpoint{URI: r.endpoint}, nil
}
