package s3

import (
	"math"
	"strings"
	"testing"
)

func TestConfigRejectsInvalidTupleAndEnvelope(t *testing.T) {
	t.Parallel()
	for _, provider := range []Provider{ProviderAmazonS3, ProviderCloudflare} {
		t.Run(string(provider)+" valid", func(t *testing.T) {
			t.Parallel()
			if _, err := validConfig(provider).validate(); err != nil {
				t.Fatalf("validate() error = %v", err)
			}
		})
	}

	tests := []struct {
		name   string
		update func(*Config)
	}{
		{name: "provider", update: func(cfg *Config) { cfg.Provider = "other" }},
		{name: "missing credentials", update: func(cfg *Config) { cfg.AccessKeyID = "" }},
		{name: "missing secret", update: func(cfg *Config) { cfg.SecretAccessKey = "" }},
		{name: "Amazon missing session token", update: func(cfg *Config) { cfg.SessionToken = "" }},
		{name: "Amazon invalid expected owner", update: func(cfg *Config) { cfg.ExpectedBucketOwner = "owner" }},
		{name: "short bucket", update: func(cfg *Config) { cfg.Bucket = "ab" }},
		{name: "dotted bucket", update: func(cfg *Config) { cfg.Bucket = "bucket.with.dot" }},
		{name: "reserved prefix", update: func(cfg *Config) { cfg.Bucket = "xn--bucket" }},
		{name: "reserved access point suffix", update: func(cfg *Config) { cfg.Bucket = "bucket-s3alias" }},
		{name: "reserved object lambda suffix", update: func(cfg *Config) { cfg.Bucket = "bucket--ol-s3" }},
		{name: "reserved directory suffix", update: func(cfg *Config) { cfg.Bucket = "bucket--x-s3" }},
		{name: "reserved table suffix", update: func(cfg *Config) { cfg.Bucket = "bucket--table-s3" }},
		{name: "reserved account regional suffix", update: func(cfg *Config) { cfg.Bucket = "bucket-an" }},
		{name: "endpoint path", update: func(cfg *Config) { cfg.Endpoint += "/path" }},
		{name: "endpoint empty query", update: func(cfg *Config) { cfg.Endpoint += "?" }},
		{name: "Amazon region mismatch", update: func(cfg *Config) { cfg.Region = "us-west-2" }},
		{name: "Amazon GovCloud", update: func(cfg *Config) {
			cfg.Region = "us-gov-west-1"
			cfg.Endpoint = "https://s3.us-gov-west-1.amazonaws.com"
		}},
		{name: "Amazon China partition", update: func(cfg *Config) {
			cfg.Region = "cn-north-1"
			cfg.Endpoint = "https://s3.cn-north-1.amazonaws.com"
		}},
		{name: "R2 region mismatch", update: func(cfg *Config) {
			cfg.Provider = ProviderCloudflare
			cfg.Endpoint = "https://0123456789abcdef0123456789abcdef.r2.cloudflarestorage.com"
		}},
		{name: "R2 arbitrary account host", update: func(cfg *Config) {
			*cfg = validConfig(ProviderCloudflare)
			cfg.Endpoint = "https://account.r2.cloudflarestorage.com"
		}},
		{name: "R2 expected owner", update: func(cfg *Config) {
			*cfg = validConfig(ProviderCloudflare)
			cfg.ExpectedBucketOwner = "123456789012"
		}},
		{name: "R2 documented object limit", update: func(cfg *Config) {
			*cfg = validConfig(ProviderCloudflare)
			cfg.MaxObjectBytes = maximumR2ObjectBytes + 1
			cfg.MultipartChunkBytes = maximumMultipartChunk
			cfg.MaxWorkingMemoryBytes = math.MaxInt64
		}},
		{name: "object limit", update: func(cfg *Config) { cfg.MaxObjectBytes = 0 }},
		{name: "multipart chunk", update: func(cfg *Config) { cfg.MultipartChunkBytes = 1 }},
		{name: "part count", update: func(cfg *Config) { cfg.MaxObjectBytes = (maximumPartCount + 1) * minimumMultipartChunk }},
		{name: "active operations", update: func(cfg *Config) { cfg.MaxActiveOperations = 0 }},
		{name: "operation duration", update: func(cfg *Config) { cfg.MaxOperationDuration = 0 }},
		{name: "presign duration", update: func(cfg *Config) { cfg.MaxPresignLifetime = 0 }},
		{name: "response header limit", update: func(cfg *Config) { cfg.MaxResponseHeaderBytes = 0 }},
		{name: "control body limit", update: func(cfg *Config) { cfg.MaxControlResponseBytes = 0 }},
		{name: "memory envelope", update: func(cfg *Config) { cfg.MaxWorkingMemoryBytes-- }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(ProviderAmazonS3)
			test.update(&cfg)
			if _, err := cfg.validate(); err == nil {
				t.Fatal("validate() error = nil, want rejection")
			}
		})
	}
}

func TestProviderObjectSizeCeilings(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		provider Provider
		maximum  int64
	}{
		{provider: ProviderAmazonS3, maximum: maximumObjectBytes},
		{provider: ProviderCloudflare, maximum: maximumR2ObjectBytes},
	} {
		t.Run(string(test.provider), func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(test.provider)
			cfg.MaxObjectBytes = test.maximum
			cfg.MultipartChunkBytes = maximumMultipartChunk
			cfg.MaxWorkingMemoryBytes = math.MaxInt64
			if _, err := cfg.validate(); err != nil {
				t.Fatalf("documented maximum %d was rejected: %v", test.maximum, err)
			}
			cfg.MaxObjectBytes++
			if _, err := cfg.validate(); err == nil {
				t.Fatalf("object size above documented maximum %d was accepted", test.maximum)
			}
		})
	}
}

func TestPortableBucketNameAcceptsOnlyTheSharedNamespace(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"abc", "portable-bucket-123", strings.Repeat("a", 63)} {
		for _, provider := range []Provider{ProviderAmazonS3, ProviderCloudflare} {
			cfg := validConfig(provider)
			cfg.Bucket = name
			cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
			if _, err := cfg.validate(); err != nil {
				t.Fatalf("%s bucket %q was rejected: %v", provider, name, err)
			}
		}
	}
}

func TestWorkingMemoryAccounting(t *testing.T) {
	t.Parallel()
	cfg := validConfig(ProviderAmazonS3)

	configured, ok := cfg.configuredStringBytes()
	if !ok || configured != 165 {
		t.Fatalf("configured string bytes = %d, %t; want 165, true", configured, ok)
	}
	sharedTrust, startupTrust, verifyTrust, ok := trustMemory()
	if !ok || sharedTrust != 32_047_104 || startupTrust != 61_997_056 || verifyTrust != 9_309_776 {
		t.Fatalf("trust bytes = %d, %d, %d, %t", sharedTrust, startupTrust, verifyTrust, ok)
	}
	if heap, ok := heapBytes(0); !ok || heap != 0 {
		t.Fatalf("heap(0) = %d, %t", heap, ok)
	}
	if heap, ok := heapBytes(32 << 10); !ok || heap != 65_544 {
		t.Fatalf("heap(32KiB) = %d, %t", heap, ok)
	}
	if heap, ok := heapBytes((32 << 10) + 1); !ok || heap != 40_960 {
		t.Fatalf("heap(32KiB+1) = %d, %t", heap, ok)
	}
	if heap, ok := heapBytes(40_960); !ok || heap != 40_960 {
		t.Fatalf("heap(40KiB) = %d, %t", heap, ok)
	}
	if heap, ok := heapBytes(40_961); !ok || heap != 49_152 {
		t.Fatalf("heap(40KiB+1) = %d, %t", heap, ok)
	}

	parts := partCount(cfg.MaxObjectBytes, cfg.MultipartChunkBytes)
	retained, retainedOK := retainedPartsMemory(parts, cfg.MaxResponseHeaderBytes)
	complete, completeOK := completeMultipartXMLMemory(parts, cfg.MaxResponseHeaderBytes)
	multipart, multipartOK := cfg.multipartMemory(configured, parts, maximumKeyBytes+maximumContentTypeBytes, verifyTrust)
	single, singleOK := cfg.operationMemory(configured, maximumKeyBytes+maximumContentTypeBytes, true, verifyTrust)
	download, downloadOK := cfg.operationMemory(configured, maximumKeyBytes, false, verifyTrust)
	control, controlOK := cfg.operationMemory(configured, maximumKeyBytes, true, verifyTrust)
	if !retainedOK || retained != 4_648 || !completeOK || complete != 29_302 ||
		!multipartOK || multipart != 12_188_550 || !singleOK || single != 12_152_528 ||
		!downloadOK || download != 11_823_312 || !controlOK || control != 12_020_432 {
		t.Fatalf("operation bytes = retained:%d complete:%d multipart:%d single:%d download:%d control:%d", retained, complete, multipart, single, download, control)
	}

	required, ok := cfg.requiredMemory()
	if !ok || required != 62_149_760 {
		t.Fatalf("required memory = %d, %t; want 62149760, true", required, ok)
	}
	cfg.MaxWorkingMemoryBytes = required
	if _, err := cfg.validate(); err != nil {
		t.Fatalf("equality validate() error = %v", err)
	}
	client, err := newClient(cfg, testImageRootSource(t))
	if err != nil {
		t.Fatalf("equality newClient() error = %v", err)
	}
	client.Close()
	cfg.MaxWorkingMemoryBytes = required - 1
	if _, err := cfg.validate(); err == nil {
		t.Fatal("one-byte-short validate() error = nil")
	}
	opened := 0
	if _, err := newClient(cfg, func() (imageRootFile, error) {
		opened++
		return testImageRootSource(t)()
	}); err == nil || opened != 0 {
		t.Fatalf("one-byte-short newClient() error = %v, root opens = %d", err, opened)
	}
	if _, ok := cfg.requiredMemoryForIntSize(32); ok {
		t.Fatal("32-bit shallow receipt was accepted")
	}

	if got := partCount(minimumMultipartChunk, minimumMultipartChunk); got != 1 {
		t.Fatalf("minimum part count = %d, want 1", got)
	}
	if got := partCount(maximumPartCount*minimumMultipartChunk, minimumMultipartChunk); got != maximumPartCount {
		t.Fatalf("maximum part count = %d, want %d", got, maximumPartCount)
	}
	tooMany := validConfig(ProviderAmazonS3)
	tooMany.MaxObjectBytes = (maximumPartCount + 1) * minimumMultipartChunk
	tooMany.MaxWorkingMemoryBytes = math.MaxInt64
	if _, err := tooMany.validate(); err == nil {
		t.Fatal("10,001-part configuration was accepted")
	}

	for _, test := range []struct {
		name     string
		header   int64
		control  int64
		required int64
	}{
		{name: "H less than E", header: 64 << 10, control: 128 << 10, required: 84_308_828},
		{name: "H equals E", header: 128 << 10, control: 128 << 10, required: 95_581_020},
		{name: "H greater than E", header: 256 << 10, control: 128 << 10, required: 118_125_404},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(ProviderAmazonS3)
			cfg.MaxResponseHeaderBytes = test.header
			cfg.MaxControlResponseBytes = test.control
			if got, ok := cfg.requiredMemory(); !ok || got != test.required {
				t.Fatalf("required memory = %d, %t; want %d, true", got, ok, test.required)
			}
		})
	}

	overflows := []struct {
		name string
		fn   func() bool
	}{
		{name: "sum", fn: func() bool { _, ok := addMemory(math.MaxInt64, 1); return ok }},
		{name: "product", fn: func() bool { _, ok := multiplyMemory(math.MaxInt64, 2); return ok }},
		{name: "heap alignment", fn: func() bool { _, ok := heapBytes(math.MaxInt64); return ok }},
		{name: "variable owner", fn: func() bool { _, ok := variableOwnerBytes(math.MaxInt64 / 2); return ok }},
		{name: "request state", fn: func() bool { _, ok := requestStateMemory(math.MaxInt64, 1); return ok }},
		{name: "retained parts", fn: func() bool { _, ok := retainedPartsMemory(math.MaxInt64, 1); return ok }},
		{name: "complete XML header escape", fn: func() bool { _, ok := completeMultipartXMLMemory(1, math.MaxInt64); return ok }},
		{name: "complete XML parts", fn: func() bool { _, ok := completeMultipartXMLMemory(math.MaxInt64, 1); return ok }},
		{name: "complete XML double", fn: func() bool { _, ok := completeMultipartXMLMemory(math.MaxInt64/150, 0); return ok }},
	}
	for _, test := range overflows {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.fn() {
				t.Fatal("overflow was accepted")
			}
		})
	}

	for _, update := range []func(*Config){
		func(cfg *Config) { cfg.MaxResponseHeaderBytes = math.MaxInt64 },
		func(cfg *Config) { cfg.MaxControlResponseBytes = math.MaxInt64 },
		func(cfg *Config) { cfg.MaxActiveOperations = math.MaxInt },
	} {
		cfg := validConfig(ProviderAmazonS3)
		update(&cfg)
		if _, ok := cfg.requiredMemory(); ok {
			t.Fatal("overflowing configuration produced a memory envelope")
		}
	}
}

func TestWorkingMemoryAccountingRejectsInvalidIntermediateInputs(t *testing.T) {
	t.Parallel()
	cfg := validConfig(ProviderAmazonS3)

	for _, test := range []struct {
		name  string
		check func() bool
	}{
		{name: "invalid configured endpoint", check: func() bool {
			invalid := cfg
			invalid.Endpoint = "://"
			_, ok := invalid.configuredStringBytes()
			return ok
		}},
		{name: "negative header retained parts", check: func() bool {
			_, ok := retainedPartsMemory(1, -1)
			return ok
		}},
		{name: "retained part multiplication", check: func() bool {
			_, ok := retainedPartsMemory(math.MaxInt64, 0)
			return ok
		}},
		{name: "header heap alignment", check: func() bool {
			_, ok := retainedPartsMemory(1, math.MaxInt64)
			return ok
		}},
		{name: "multipart upload ID", check: func() bool {
			invalid := cfg
			invalid.MaxControlResponseBytes = -1
			_, ok := invalid.multipartMemory(1, 1, 1, 1)
			return ok
		}},
		{name: "multipart response bound", check: func() bool {
			invalid := cfg
			invalid.MaxResponseHeaderBytes = -1
			_, ok := invalid.multipartMemory(1, 1, 1, 1)
			return ok
		}},
		{name: "operation response bound", check: func() bool {
			invalid := cfg
			invalid.MaxResponseHeaderBytes = -1
			_, ok := invalid.operationMemory(1, 1, false, 1)
			return ok
		}},
		{name: "operation control bound", check: func() bool {
			invalid := cfg
			invalid.MaxControlResponseBytes = -1
			_, ok := invalid.operationMemory(1, 1, true, 1)
			return ok
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.check() {
				t.Fatal("memory accounting accepted an invalid intermediate bound")
			}
		})
	}
	for _, input := range []struct {
		object, chunk int64
	}{
		{object: 0, chunk: minimumMultipartChunk},
		{object: minimumMultipartChunk, chunk: 0},
	} {
		if got := partCount(input.object, input.chunk); got != 0 {
			t.Fatalf("partCount(%d, %d) = %d, want 0", input.object, input.chunk, got)
		}
	}
	withoutActiveOperations := cfg
	withoutActiveOperations.MaxActiveOperations = 0
	if _, ok := withoutActiveOperations.requiredMemoryForIntSize(64); ok {
		t.Fatal("memory accounting accepted zero active operations")
	}
	withoutParts := cfg
	withoutParts.MaxObjectBytes = 0
	if _, ok := withoutParts.requiredMemoryForIntSize(64); ok {
		t.Fatal("memory accounting accepted zero object parts")
	}
}
