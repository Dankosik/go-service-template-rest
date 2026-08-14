//go:build integration

package integration_test

import (
	"os"
	"testing"
)

// profile:object-storage:start
func TestS3ObjectStorageConformanceRequiresProviderCertification(t *testing.T) {
	if os.Getenv("REQUIRE_S3_CONFORMANCE") != "1" {
		t.Skip("provider conformance is not requested")
	}

	switch os.Getenv("S3_CONFORMANCE_PROVIDER") {
	case "amazon_s3", "cloudflare_r2":
		t.Fatal("provider conformance requires its separately authorized certification unit")
	default:
		t.Fatal("provider conformance requires one exact provider selector")
	}
}

// profile:object-storage:end
