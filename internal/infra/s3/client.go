package s3

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Provider string

const (
	ProviderAmazonS3   Provider = "amazon_s3"
	ProviderCloudflare Provider = "cloudflare_r2"

	CredentialSourceAWSDefault = "aws_default"
	CredentialSourceStatic     = "static"

	multipartPartBytes      int64 = 8 << 20
	maximumUploadParts      int64 = 10_000
	maximumObjectBytes            = multipartPartBytes * maximumUploadParts
	maximumActiveOperations       = 4
	multipartFailureTimeout       = 5 * time.Second
	responseHeaderTimeout         = 30 * time.Second
	maximumPresignLifetime        = 7 * 24 * time.Hour
	maximumContentTypeBytes       = 1 << 10
)

var (
	bucketName          = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
	regionName          = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d+$`)
	r2Endpoint          = regexp.MustCompile(`^[0-9a-f]{32}(?:\.(?:eu|fedramp))?\.r2\.cloudflarestorage\.com$`)
	bucketOwner         = regexp.MustCompile(`^\d{12}$`)
	errRequestFailed    = errors.New("object storage request failed")
	errTransferComplete = errors.New("object storage transfer completion failed")
)

// Config is the provider tuple plus the one product size ceiling.
type Config struct {
	Provider            Provider
	Endpoint            string
	Region              string
	Bucket              string
	ExpectedBucketOwner string
	CredentialSource    string
	MaxObjectBytes      int64
}

type objectAPI interface {
	PutObject(ctx context.Context, input *awss3.PutObjectInput, options ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, input *awss3.GetObjectInput, options ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	HeadObject(ctx context.Context, input *awss3.HeadObjectInput, options ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, input *awss3.DeleteObjectInput, options ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
}

type uploadAPI interface {
	UploadObject(ctx context.Context, input *transfermanager.UploadObjectInput, options ...func(*transfermanager.Options)) (*transfermanager.UploadObjectOutput, error)
}

type presignAPI interface {
	PresignGetObject(ctx context.Context, input *awss3.GetObjectInput, options ...func(*awss3.PresignOptions)) (*signer.PresignedHTTPRequest, error)
}

type Client struct {
	config    Config
	sdk       objectAPI
	uploader  uploadAPI
	presigner presignAPI
	transport *http.Transport
	tokens    chan struct{}
}

// New builds the client without provider I/O. Credential retrieval remains owned
// by the explicitly selected SDK provider.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build S3 adapter: %w", errRequestFailed)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("build S3 adapter: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	httpClient, transport := newHTTPClient()
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithHTTPClient(httpClient),
		awsconfig.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(configureRetry)
		}),
	}
	if cfg.CredentialSource == CredentialSourceStatic {
		accessKey := strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))
		secretKey := strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))
		if accessKey == "" || secretKey == "" {
			return nil, errors.New("build S3 adapter: static AWS credentials are required")
		}
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, os.Getenv("AWS_SESSION_TOKEN")),
		))
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("build S3 adapter configuration: %w", errRequestFailed)
	}

	var baseEndpoint *string
	if cfg.Provider == ProviderCloudflare {
		baseEndpoint = aws.String(cfg.Endpoint)
	}
	sdk := awss3.NewFromConfig(awsConfig, func(options *awss3.Options) {
		options.BaseEndpoint = baseEndpoint
		options.UsePathStyle = false
		options.UseAccelerate = false
		options.UseARNRegion = false
		options.DisableMultiRegionAccessPoints = true
		options.DisableS3ExpressSessionAuth = aws.Bool(true)
		options.EndpointOptions.UseDualStackEndpoint = aws.DualStackEndpointStateDisabled
		options.EndpointOptions.UseFIPSEndpoint = aws.FIPSEndpointStateDisabled
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		options.DisableLogOutputChecksumValidationSkipped = true
		options.ClientLogMode = 0
	})
	uploader := transfermanager.New(transferClient{Client: sdk}, configureTransfer)

	return &Client{
		config: cfg, sdk: sdk, uploader: uploader, presigner: awss3.NewPresignClient(sdk),
		transport: transport, tokens: make(chan struct{}, maximumActiveOperations),
	}, nil
}

func configureRetry(options *retry.StandardOptions) {
	options.MaxAttempts = 3
	options.MaxBackoff = time.Second
}

func configureTransfer(options *transfermanager.Options) {
	options.PartSizeBytes = multipartPartBytes
	options.MultipartUploadThreshold = multipartPartBytes
	options.Concurrency = 1
	options.FailTimeout = multipartFailureTimeout
	options.MaxUploadParts = maximumUploadParts
	options.ChecksumAlgorithm = tmtypes.ChecksumAlgorithm("CRC64NVME")
	options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
}

// transferClient keeps the transfer manager's unavoidable completion log free
// of provider error text while preserving its public multipart implementation.
type transferClient struct{ *awss3.Client }

func (c transferClient) CompleteMultipartUpload(
	ctx context.Context,
	input *awss3.CompleteMultipartUploadInput,
	options ...func(*awss3.Options),
) (*awss3.CompleteMultipartUploadOutput, error) {
	output, err := c.Client.CompleteMultipartUpload(ctx, input, options...)
	if err != nil {
		return output, errTransferComplete
	}
	return output, nil
}

func (cfg Config) validate() error {
	if !bucketName.MatchString(cfg.Bucket) {
		return errors.New("build S3 adapter: bucket must be one dotless DNS name")
	}
	if cfg.MaxObjectBytes <= 0 || cfg.MaxObjectBytes > maximumObjectBytes {
		// ponytail: fixed 8 MiB parts cap objects at 80,000 MiB (78.125 GiB); add a measured
		// part-size policy if a real adopter needs larger objects.
		return fmt.Errorf("build S3 adapter: maximum object bytes must be in 1..%d", maximumObjectBytes)
	}
	if cfg.CredentialSource != CredentialSourceStatic && cfg.CredentialSource != CredentialSourceAWSDefault {
		return errors.New("build S3 adapter: credential source is invalid")
	}

	switch cfg.Provider {
	case ProviderAmazonS3:
		if cfg.Endpoint != "" || !regionName.MatchString(cfg.Region) || !bucketOwner.MatchString(cfg.ExpectedBucketOwner) {
			return errors.New("build S3 adapter: Amazon requires a region, expected owner, and no endpoint override")
		}
	case ProviderCloudflare:
		endpoint, err := url.Parse(cfg.Endpoint)
		if err != nil || !isHTTPSOrigin(endpoint) || !r2Endpoint.MatchString(endpoint.Host) || cfg.Region != "auto" || cfg.ExpectedBucketOwner != "" || cfg.CredentialSource != CredentialSourceStatic {
			return errors.New("build S3 adapter: R2 requires its HTTPS account endpoint, region auto, static credentials, and no expected owner")
		}
	default:
		return errors.New("build S3 adapter: provider is invalid")
	}
	return nil
}

func isHTTPSOrigin(endpoint *url.URL) bool {
	return endpoint != nil && endpoint.Scheme == "https" && endpoint.Host != "" && endpoint.User == nil &&
		endpoint.Path == "" && endpoint.RawQuery == "" && endpoint.Fragment == "" && endpoint.Port() == ""
}

func newHTTPClient() (*http.Client, *http.Transport) {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		panic("http.DefaultTransport is not *http.Transport")
	}
	transport := baseTransport.Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	transport.MaxResponseHeaderBytes = 64 << 10
	transport.MaxConnsPerHost = maximumActiveOperations
	transport.MaxIdleConnsPerHost = maximumActiveOperations
	return &http.Client{
		Transport:     transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}, transport
}

func (c *Client) expectedBucketOwner() *string {
	if c.config.Provider == ProviderAmazonS3 {
		return aws.String(c.config.ExpectedBucketOwner)
	}
	return nil
}

func (c *Client) Close() { c.transport.CloseIdleConnections() }
