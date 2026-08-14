package s3

import (
	"bytes"
	"errors"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/objectstorage"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTelemetryContractIsBoundedAndSecret(t *testing.T) {
	reader, meter := telemetrytest.NewManualMeter(t, telemetryScope)
	telemetry, err := newTelemetryWithMeter(meter, noop.NewTracerProvider().Tracer(telemetryScope))
	if err != nil {
		t.Fatalf("newTelemetryWithMeter() error = %v", err)
	}
	client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodHead {
			t.Fatalf("unexpected request %s", request.Method)
		}
		return s3Response(http.StatusOK, http.Header{
			"Content-Length": []string{"1"},
			"Last-Modified":  []string{time.Now().UTC().Format(http.TimeFormat)},
		}, ""), nil
	})
	client.telemetry = telemetry

	if _, err := client.Metadata(t.Context(), "object-key-canary"); err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if _, err := client.Upload(t.Context(), "object-key-canary", nil, objectstorage.UploadOptions{ContentLength: 1, Intent: objectstorage.UploadReplace}); objectstorage.Kind(err) != objectstorage.KindInvalid {
		t.Fatalf("Upload() kind = %q, want invalid", objectstorage.Kind(err))
	}
	if _, err := client.PresignGET(t.Context(), "object-key-canary", time.Second); err != nil {
		t.Fatalf("PresignGET() error = %v", err)
	}
	client.tokens <- struct{}{}
	client.tokens <- struct{}{}
	if err := client.Delete(t.Context(), "object-key-canary"); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("Delete() kind = %q, want busy", objectstorage.Kind(err))
	}
	<-client.tokens
	<-client.tokens

	operations := map[string]string{}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "object_storage.operations" {
			return
		}
		sum, ok := measured.Data.(metricdata.Sum[int64])
		if !ok {
			t.Fatalf("operations aggregation = %T, want Sum[int64]", measured.Data)
		}
		for _, point := range sum.DataPoints {
			op := telemetrytest.Attribute(t, point.Attributes, "object_storage.operation")
			result := telemetrytest.Attribute(t, point.Attributes, "object_storage.result")
			path := telemetrytest.Attribute(t, point.Attributes, "object_storage.transfer_path")
			cleanup := telemetrytest.Attribute(t, point.Attributes, "object_storage.cleanup")
			if point.Attributes.Len() != 4 {
				t.Fatalf("operation attributes = %v, want only the four closed labels", point.Attributes)
			}
			operations[op] = result + "/" + path + "/" + cleanup
		}
	})
	want := map[string]string{
		"metadata": "success/none/none",
		"upload":   "invalid/none/none",
		"presign":  "success/none/none",
		"delete":   "busy/none/none",
	}
	if !maps.Equal(operations, want) {
		t.Fatalf("operation series = %#v, want %#v", operations, want)
	}
	telemetrytest.AssertNoAttributeContains(t, reader,
		"object-key-canary", "test-access-key", "test-secret-key", "examplebucket", "amazonaws.com", "X-Amz-",
	)
	if boundedResult("not-a-result") != "unknown" || boundedPath("not-a-path") != "unknown" || boundedCleanup("not-a-cleanup") != "unknown" {
		t.Fatal("unknown telemetry values did not collapse to the closed fallback")
	}

	for _, test := range []struct {
		name    string
		abortOK bool
		want    string
	}{
		{name: "complete", abortOK: true, want: "complete"},
		{name: "pending", want: "pending"},
	} {
		t.Run("multipart cleanup "+test.name, func(t *testing.T) {
			reader, meter := telemetrytest.NewManualMeter(t, telemetryScope)
			telemetry, err := newTelemetryWithMeter(meter, noop.NewTracerProvider().Tracer(telemetryScope))
			if err != nil {
				t.Fatal(err)
			}
			cfg := validConfig(ProviderAmazonS3)
			cfg.MaxObjectBytes = cfg.MultipartChunkBytes + 1
			cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
			client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
				query := request.URL.Query()
				switch {
				case request.Method == http.MethodPost && query.Has("uploads"):
					return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>"), nil
				case request.Method == http.MethodPut:
					return nil, errors.New("scripted part failure")
				case request.Method == http.MethodDelete:
					if test.abortOK {
						return s3Response(http.StatusNoContent, nil, ""), nil
					}
					return nil, errors.New("scripted abort failure")
				case request.Method == http.MethodGet:
					return s3Response(http.StatusOK, nil, "<ListPartsResult><IsTruncated>false</IsTruncated></ListPartsResult>"), nil
				default:
					t.Fatalf("unexpected request %s %s", request.Method, request.URL)
					return nil, errors.New("unexpected scripted request")
				}
			})
			client.telemetry = telemetry
			_, err = client.Upload(t.Context(), "object-key-canary", bytes.NewReader(make([]byte, cfg.MultipartChunkBytes+1)), objectstorage.UploadOptions{ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace})
			if objectstorage.Kind(err) != objectstorage.KindInternal {
				t.Fatalf("Upload() kind = %q, want internal", objectstorage.Kind(err))
			}
			gotCleanup := ""
			telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
				if measured.Name != "object_storage.operations" {
					return
				}
				for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
					gotCleanup = telemetrytest.Attribute(t, point.Attributes, "object_storage.cleanup")
				}
			})
			if gotCleanup != test.want {
				t.Fatalf("multipart cleanup telemetry = %q, want %q", gotCleanup, test.want)
			}
			telemetrytest.AssertNoAttributeContains(t, reader, "object-key-canary", "private-upload", "test-secret-key")
		})
	}
}
