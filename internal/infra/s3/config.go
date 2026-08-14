package s3

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Provider string

const (
	ProviderAmazonS3   Provider = "amazon_s3"
	ProviderCloudflare Provider = "cloudflare_r2"

	minimumMultipartChunk       = 5 << 20
	maximumMultipartChunk       = 5 << 30
	maximumObjectBytes          = 5 << 40
	maximumPartCount            = 10_000
	maximumKeyBytes       int64 = 1024

	fixedStreamBufferBytes  int64 = 2 << 20
	sharedStreamBufferBytes int64 = 64 << 10
	heapSmallObjectLimit    int64 = 32 << 10
	heapPageBytes           int64 = 8 << 10
	variableOwnerOverhead   int64 = 64 << 10
)

var (
	bucketName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])?$`)
	regionName = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d+$`)
)

// Config is one explicit immutable provider tuple and its finite local bounds.
type Config struct {
	Provider        Provider
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string

	MaxObjectBytes          int64
	MultipartChunkBytes     int64
	MaxActiveOperations     int
	MaxOperationDuration    time.Duration
	MaxPresignLifetime      time.Duration
	MaxResponseHeaderBytes  int64
	MaxControlResponseBytes int64
	MaxWorkingMemoryBytes   int64
}

func (cfg Config) validate() (*url.URL, error) {
	endpoint, err := validateTuple(cfg)
	if err != nil {
		return nil, err
	}
	if err := validateBounds(cfg); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func validateTuple(cfg Config) (*url.URL, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, errors.New("build S3 adapter: static credentials are required")
	}
	if !bucketName.MatchString(cfg.Bucket) || strings.Contains(cfg.Bucket, ".") {
		return nil, errors.New("build S3 adapter: bucket must be a dotless DNS name")
	}
	endpoint, err := url.Parse(strings.TrimSpace(cfg.Endpoint))
	if err != nil || !isHTTPSOrigin(endpoint) {
		return nil, errors.New("build S3 adapter: endpoint must be an HTTPS origin without a port, path, query, fragment, or user info")
	}
	if net.ParseIP(endpoint.Hostname()) != nil {
		return nil, errors.New("build S3 adapter: endpoint must be a DNS name")
	}
	endpoint.Host = strings.ToLower(endpoint.Host)

	switch cfg.Provider {
	case ProviderAmazonS3:
		if !regionName.MatchString(cfg.Region) || cfg.Region == "auto" || strings.HasPrefix(cfg.Region, "us-gov-") || endpoint.Host != "s3."+cfg.Region+".amazonaws.com" {
			return nil, errors.New("build S3 adapter: Amazon endpoint and region must be one commercial regional tuple")
		}
	case ProviderCloudflare:
		if cfg.Region != "auto" || !strings.HasSuffix(endpoint.Host, ".r2.cloudflarestorage.com") || strings.TrimSuffix(endpoint.Host, ".r2.cloudflarestorage.com") == "" {
			return nil, errors.New("build S3 adapter: Cloudflare R2 endpoint and region must be one account tuple")
		}
	default:
		return nil, errors.New("build S3 adapter: provider is invalid")
	}
	return endpoint, nil
}

func isHTTPSOrigin(endpoint *url.URL) bool {
	return endpoint.Scheme == "https" && endpoint.Host != "" && endpoint.Hostname() != "" &&
		endpoint.User == nil && endpoint.Path == "" && endpoint.RawQuery == "" && !endpoint.ForceQuery && endpoint.Fragment == "" && endpoint.Port() == ""
}

func validateBounds(cfg Config) error {
	if cfg.MaxObjectBytes <= 0 || cfg.MaxObjectBytes > maximumObjectBytes {
		return errors.New("build S3 adapter: maximum object bytes are invalid")
	}
	if cfg.MultipartChunkBytes < minimumMultipartChunk || cfg.MultipartChunkBytes > maximumMultipartChunk || cfg.MultipartChunkBytes > cfg.MaxObjectBytes {
		return errors.New("build S3 adapter: multipart chunk bytes are invalid")
	}
	if cfg.MaxActiveOperations <= 0 || cfg.MaxOperationDuration <= 0 || cfg.MaxPresignLifetime <= 0 ||
		cfg.MaxResponseHeaderBytes <= 0 || cfg.MaxControlResponseBytes <= 0 || cfg.MaxWorkingMemoryBytes <= 0 {
		return errors.New("build S3 adapter: every bound must be positive")
	}
	if parts := partCount(cfg.MaxObjectBytes, cfg.MultipartChunkBytes); parts == 0 || parts > maximumPartCount {
		return errors.New("build S3 adapter: multipart part count is invalid")
	}
	required, ok := cfg.requiredMemory()
	if !ok || cfg.MaxWorkingMemoryBytes < required {
		return fmt.Errorf("build S3 adapter: working memory must cover %d bytes", required)
	}
	return nil
}

func partCount(objectBytes, chunkBytes int64) int64 {
	if objectBytes <= 0 || chunkBytes <= 0 {
		return 0
	}
	return 1 + (objectBytes-1)/chunkBytes
}

func (cfg Config) requiredMemory() (int64, bool) {
	return cfg.requiredMemoryForIntSize(strconv.IntSize)
}

func (cfg Config) requiredMemoryForIntSize(intSize int) (int64, bool) {
	if intSize != 64 || cfg.MaxActiveOperations <= 0 {
		return 0, false
	}
	parts := partCount(cfg.MaxObjectBytes, cfg.MultipartChunkBytes)
	if parts == 0 {
		return 0, false
	}
	configured, ok := cfg.configuredStringBytes()
	if !ok {
		return 0, false
	}
	trustShared, trustStartup, trustVerify, ok := trustMemory()
	if !ok {
		return 0, false
	}
	requestWithContentType, ok := addMemory(maximumKeyBytes, maximumContentTypeBytes)
	if !ok {
		return 0, false
	}
	multipart, ok := cfg.multipartMemory(configured, parts, requestWithContentType, trustVerify)
	if !ok {
		return 0, false
	}
	single, ok := cfg.operationMemory(configured, requestWithContentType, true, trustVerify)
	if !ok {
		return 0, false
	}
	download, ok := cfg.operationMemory(configured, maximumKeyBytes, false, trustVerify)
	if !ok {
		return 0, false
	}
	control, ok := cfg.operationMemory(configured, maximumKeyBytes, true, trustVerify)
	if !ok {
		return 0, false
	}
	configuredOwner, ok := variableOwnerBytes(configured)
	if !ok {
		return 0, false
	}
	construction, ok := addMemory(sharedStreamBufferBytes, configuredOwner, trustStartup)
	if !ok {
		return 0, false
	}
	shared, ok := addMemory(sharedStreamBufferBytes, configuredOwner, trustShared)
	if !ok {
		return 0, false
	}
	perOperation := max(multipart, single, download, control)
	active := int64(cfg.MaxActiveOperations)
	activeOperations, ok := multiplyMemory(active, perOperation)
	if !ok {
		return 0, false
	}
	steady, ok := addMemory(shared, activeOperations)
	if !ok {
		return 0, false
	}
	return max(construction, steady), true
}

func (cfg Config) configuredStringBytes() (int64, bool) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Host == "" {
		return 0, false
	}
	return addMemory(
		int64(len(cfg.Provider)),
		int64(len(cfg.Endpoint)),
		int64(len(cfg.Bucket))+1+int64(len(endpoint.Host)),
		int64(len(cfg.Region)),
		int64(len(cfg.Bucket)),
		int64(len(cfg.AccessKeyID)),
		int64(len(cfg.SecretAccessKey)),
		int64(len(cfg.SessionToken)),
	)
}

func trustMemory() (shared, startup, verify int64, ok bool) {
	rootParseBytes, ok := multiplyMemory(128, maxImageRootCertificates)
	if !ok {
		return 0, 0, 0, false
	}
	rootPoolBytes, ok := addMemory(maxImageRootBundleBytes, rootParseBytes)
	if !ok {
		return 0, 0, 0, false
	}
	shared, ok = variableOwnerBytes(rootPoolBytes)
	if !ok {
		return 0, 0, 0, false
	}
	startupSource, ok := variableOwnerBytes(maxImageRootBundleBytes + 1)
	if !ok {
		return 0, 0, 0, false
	}
	startup, ok = addMemory(shared, startupSource)
	if !ok {
		return 0, 0, 0, false
	}
	verifyInput, ok := multiplyMemory(80, maxImageRootCertificates)
	if !ok {
		return 0, 0, 0, false
	}
	verifyHeap, ok := heapBytes(verifyInput)
	if !ok {
		return 0, 0, 0, false
	}
	verify, ok = multiplyMemory(2*101, verifyHeap)
	return shared, startup, verify, ok
}

func (cfg Config) multipartMemory(configured, parts, requestBytes, trustVerify int64) (int64, bool) {
	retained, ok := retainedPartsMemory(parts, cfg.MaxResponseHeaderBytes)
	if !ok {
		return 0, false
	}
	complete, ok := completeMultipartXMLMemory(parts, cfg.MaxResponseHeaderBytes)
	if !ok {
		return 0, false
	}
	uploadID, ok := heapBytes(cfg.MaxControlResponseBytes)
	if !ok {
		return 0, false
	}
	uploadSession, ok := addMemory(16, uploadID)
	if !ok {
		return 0, false
	}
	request, ok := requestStateMemory(configured, requestBytes)
	if !ok {
		return 0, false
	}
	response, ok := variableOwnerBytes(cfg.MaxResponseHeaderBytes)
	if !ok {
		return 0, false
	}
	control, ok := variableOwnerBytes(cfg.MaxControlResponseBytes)
	if !ok {
		return 0, false
	}
	return addMemory(
		fixedStreamBufferBytes, trustVerify, request, response, control,
		maximumKeyBytes, maximumContentTypeBytes, uploadSession, retained, complete,
	)
}

func (cfg Config) operationMemory(configured, requestBytes int64, includeControl bool, trustVerify int64) (int64, bool) {
	request, ok := requestStateMemory(configured, requestBytes)
	if !ok {
		return 0, false
	}
	response, ok := variableOwnerBytes(cfg.MaxResponseHeaderBytes)
	if !ok {
		return 0, false
	}
	values := []int64{fixedStreamBufferBytes, trustVerify, request, response, maximumKeyBytes}
	if requestBytes > maximumKeyBytes {
		values = append(values, maximumContentTypeBytes)
	}
	if includeControl {
		control, controlOK := variableOwnerBytes(cfg.MaxControlResponseBytes)
		if !controlOK {
			return 0, false
		}
		values = append(values, control)
	}
	return addMemory(values...)
}

func requestStateMemory(configured, requestBytes int64) (int64, bool) {
	total, ok := addMemory(configured, requestBytes)
	if !ok {
		return 0, false
	}
	return variableOwnerBytes(total)
}

func retainedPartsMemory(parts, headerBytes int64) (int64, bool) {
	shallowInput, ok := multiplyMemory(96, parts)
	if !ok {
		return 0, false
	}
	shallow, ok := heapBytes(shallowInput)
	if !ok {
		return 0, false
	}
	checksum, ok := heapBytes(12)
	if !ok {
		return 0, false
	}
	header, ok := heapBytes(headerBytes)
	if !ok {
		return 0, false
	}
	perPart, ok := addMemory(8, 32, checksum, header)
	if !ok {
		return 0, false
	}
	partPointees, ok := multiplyMemory(parts, perPart)
	if !ok {
		return 0, false
	}
	return addMemory(shallow, partPointees)
}

func completeMultipartXMLMemory(parts, headerBytes int64) (int64, bool) {
	escapedHeader, ok := multiplyMemory(5, headerBytes)
	if !ok {
		return 0, false
	}
	perPart, ok := addMemory(107, escapedHeader)
	if !ok {
		return 0, false
	}
	partsXML, ok := multiplyMemory(parts, perPart)
	if !ok {
		return 0, false
	}
	xmlLength, ok := addMemory(101, partsXML)
	if !ok {
		return 0, false
	}
	doubled, ok := multiplyMemory(2, xmlLength)
	if !ok {
		return 0, false
	}
	return addMemory(doubled, 8<<10)
}

func variableOwnerBytes(input int64) (int64, bool) {
	heap, ok := heapBytes(input)
	if !ok {
		return 0, false
	}
	owners, ok := multiplyMemory(64, heap)
	if !ok {
		return 0, false
	}
	return addMemory(owners, variableOwnerOverhead)
}

func heapBytes(input int64) (int64, bool) {
	if input < 0 {
		return 0, false
	}
	if input == 0 {
		return 0, true
	}
	if input <= heapSmallObjectLimit {
		doubled, ok := multiplyMemory(2, input)
		if !ok {
			return 0, false
		}
		return addMemory(doubled, 8)
	}
	aligned, ok := addMemory(input, heapPageBytes-1)
	if !ok {
		return 0, false
	}
	return (aligned / heapPageBytes) * heapPageBytes, true
}

func multiplyMemory(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || left != 0 && right > math.MaxInt64/left {
		return 0, false
	}
	return left * right, true
}

func addMemory(values ...int64) (int64, bool) {
	var total int64
	for _, value := range values {
		if value < 0 || value > math.MaxInt64-total {
			return 0, false
		}
		total += value
	}
	return total, true
}
