package s3

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"github.com/example/go-service-template-rest/internal/objectstorage"
	"github.com/example/go-service-template-rest/internal/waittest"
	"go.uber.org/goleak"
)

func TestNewUsesOnlyStaticConfigurationAndPerformsNoIO(t *testing.T) {
	t.Parallel()
	testNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO(t)
}

func TestExpectedBucketOwnerProjectsEveryOperation(t *testing.T) {
	t.Parallel()
	for _, provider := range []Provider{ProviderAmazonS3, ProviderCloudflare} {
		t.Run(string(provider), func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(provider)
			multipartData := bytes.Repeat([]byte("m"), int(cfg.MultipartChunkBytes)+1)
			downloadData := []byte("download")
			seen := map[string]bool{}
			creates := 0
			client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
				query := request.URL.Query()
				switch {
				case request.Method == http.MethodPut && query.Get("partNumber") == "":
					seen["PutObject"] = true
					payload, _ := decodeAWSChunked(t, request.Body)
					return s3Response(http.StatusOK, http.Header{
						"X-Amz-Checksum-Crc64nvme": {testCRC64NVME(payload)},
						"X-Amz-Checksum-Type":      {"FULL_OBJECT"},
					}, ""), nil
				case request.Method == http.MethodPost && query.Has("uploads"):
					seen["CreateMultipartUpload"] = true
					creates++
					return s3Response(http.StatusOK, nil, fmt.Sprintf("<InitiateMultipartUploadResult><UploadId>upload-%d</UploadId></InitiateMultipartUploadResult>", creates)), nil
				case request.Method == http.MethodPut && query.Get("partNumber") != "":
					seen["UploadPart"] = true
					payload, _ := decodeAWSChunked(t, request.Body)
					header := http.Header{"Etag": {"part"}}
					if query.Get("uploadId") == "upload-1" {
						header.Set("X-Amz-Checksum-Crc64nvme", testCRC64NVME(payload))
					}
					return s3Response(http.StatusOK, header, ""), nil
				case request.Method == http.MethodPost && query.Get("uploadId") == "upload-1":
					seen["CompleteMultipartUpload"] = true
					_, _ = io.Copy(io.Discard, request.Body)
					return s3Response(http.StatusOK, nil, "<CompleteMultipartUploadResult><ChecksumCRC64NVME>"+testCRC64NVME(multipartData)+"</ChecksumCRC64NVME><ChecksumType>FULL_OBJECT</ChecksumType></CompleteMultipartUploadResult>"), nil
				case request.Method == http.MethodDelete && query.Get("uploadId") != "":
					seen["AbortMultipartUpload"] = true
					return s3Response(http.StatusNoContent, nil, ""), nil
				case request.Method == http.MethodGet && query.Get("uploadId") != "":
					seen["ListParts"] = true
					return s3Response(http.StatusOK, nil, "<ListPartsResult><IsTruncated>false</IsTruncated></ListPartsResult>"), nil
				case request.Method == http.MethodGet:
					seen["GetObject"] = true
					return s3Response(http.StatusOK, http.Header{
						"Content-Length":           {strconv.Itoa(len(downloadData))},
						"X-Amz-Checksum-Crc64nvme": {testCRC64NVME(downloadData)},
						"X-Amz-Checksum-Type":      {"FULL_OBJECT"},
					}, string(downloadData)), nil
				case request.Method == http.MethodHead:
					seen["HeadObject"] = true
					return s3Response(http.StatusOK, http.Header{
						"Content-Length": {"1"},
						"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
					}, ""), nil
				case request.Method == http.MethodDelete:
					seen["DeleteObject"] = true
					return s3Response(http.StatusNoContent, nil, ""), nil
				default:
					t.Fatalf("unexpected operation request %s %s", request.Method, request.URL)
					return nil, http.ErrAbortHandler
				}
			})

			if _, err := client.Upload(t.Context(), "single", uploadSource(strings.NewReader("x")), objectstorage.UploadOptions{ContentLength: 1, Intent: objectstorage.UploadReplace}); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Upload(t.Context(), "multipart", uploadSource(bytes.NewReader(multipartData)), objectstorage.UploadOptions{ContentLength: int64(len(multipartData)), Intent: objectstorage.UploadReplace}); err != nil {
				t.Fatal(err)
			}
			failed, err := client.Upload(t.Context(), "cleanup", uploadSource(bytes.NewReader(multipartData)), objectstorage.UploadOptions{ContentLength: int64(len(multipartData)), Intent: objectstorage.UploadReplace})
			if objectstorage.Kind(err) != objectstorage.KindIntegrityFailed || failed.Cleanup == objectstorage.CleanupNone {
				t.Fatalf("cleanup upload = %#v, %v", failed, err)
			}
			download, err := client.Download(t.Context(), "download")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadAll(download.Body); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Metadata(t.Context(), "metadata"); err != nil {
				t.Fatal(err)
			}
			if err := client.Delete(t.Context(), "delete"); err != nil {
				t.Fatal(err)
			}
			if provider == ProviderCloudflare {
				empty, complete := client.multipartPartsEmpty(t.Context(), "projection", "projection")
				if !empty || !complete {
					t.Fatal("R2 ListParts projection did not complete")
				}
			}
			presigned, err := client.PresignGET(t.Context(), "presign", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			signed, err := url.Parse(presigned.URL)
			if err != nil {
				t.Fatal(err)
			}
			if got := signed.Query().Get("x-amz-expected-bucket-owner"); got != cfg.ExpectedBucketOwner {
				t.Fatalf("presigned expected owner = %q, want %q", got, cfg.ExpectedBucketOwner)
			}

			want := []string{"PutObject", "CreateMultipartUpload", "UploadPart", "CompleteMultipartUpload", "AbortMultipartUpload", "ListParts", "GetObject", "HeadObject", "DeleteObject"}
			for _, operation := range want {
				if !seen[operation] {
					t.Errorf("%s was not exercised", operation)
				}
			}
		})
	}
}

func TestNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO(t *testing.T) {
	t.Parallel()
	testNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO(t)
}

func testNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO(t *testing.T) {
	t.Helper()

	for _, key := range []string{"AWS_REGION", "AWS_DEFAULT_REGION", "AWS_PROFILE", "AWS_ENDPOINT_URL", "AWS_RETRY_MODE", "AWS_MAX_ATTEMPTS", "AWS_SDK_LOAD_CONFIG", "HTTP_PROXY", "HTTPS_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR"} {
		t.Setenv(key, "hostile-value")
	}

	for _, provider := range []Provider{ProviderAmazonS3, ProviderCloudflare} {
		t.Run(string(provider), func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(provider)
			client, err := newClient(cfg, testImageRootSource(t))
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			base, ok := client.transport.base.(*httpclient.Client)
			if !ok {
				t.Fatalf("transport base = %T, want *httpclient.Client", client.transport.base)
			}
			t.Cleanup(base.CloseIdleConnections)

			options := client.sdk.Options()
			if options.Region != cfg.Region || options.Retryer.MaxAttempts() != (aws.NopRetryer{}).MaxAttempts() {
				t.Fatal("SDK options did not retain the static region and one-attempt retryer")
			}
			if options.RequestChecksumCalculation != aws.RequestChecksumCalculationWhenRequired || options.ResponseChecksumValidation != aws.ResponseChecksumValidationWhenRequired || !options.DisableClockSkewCorrection {
				t.Fatal("SDK options did not retain the required local checksum and clock policy")
			}
			credentials, err := options.Credentials.Retrieve(context.Background())
			if err != nil || credentials.AccessKeyID != cfg.AccessKeyID || credentials.SecretAccessKey != cfg.SecretAccessKey {
				t.Fatalf("static credentials = %#v, %v", credentials, err)
			}
			if client.transport.endpoint.Host != cfg.Bucket+"."+mustHost(t, cfg.Endpoint) {
				t.Fatalf("final authority = %q", client.transport.endpoint.Host)
			}
			if client.roots == nil {
				t.Fatal("image root pool = nil")
			}
		})
	}
}

func TestObjectStorageStartupLoadsImageRootsLocally(t *testing.T) {
	t.Parallel()
	opened := 0
	source := testImageRootSource(t)
	client, err := newClient(validConfig(ProviderAmazonS3), func() (imageRootFile, error) {
		opened++
		return source()
	})
	if err != nil {
		t.Fatalf("newClient() error = %v", err)
	}
	t.Cleanup(client.Close)
	if opened != 1 || client.roots == nil {
		t.Fatalf("image roots opened %d times, pool = %p", opened, client.roots)
	}
}

func mustHost(t *testing.T, rawURL string) string {
	t.Helper()
	endpoint, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return endpoint.Host
}

func TestAdmissionIsProcessWideAndNonBlocking(t *testing.T) {
	t.Parallel()
	cfg := validConfig(ProviderAmazonS3)
	cfg.MaxActiveOperations = 2
	cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
	var heads int
	var requests int
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		requests++
		switch request.Method {
		case http.MethodGet:
			return admissionDownloadResponse("x"), nil
		case http.MethodHead:
			heads++
			if heads == 1 {
				return s3Response(http.StatusOK, http.Header{"Content-Length": []string{"1"}, "Last-Modified": []string{time.Now().UTC().Format(http.TimeFormat)}}, ""), nil
			}
			return s3Response(http.StatusInternalServerError, nil, ""), nil
		case http.MethodDelete:
			return s3Response(http.StatusNoContent, nil, ""), nil
		default:
			t.Fatalf("unexpected request %s", request.Method)
			return nil, errors.New("unexpected request")
		}
	})

	first, err := client.Download(t.Context(), "first")
	if err != nil {
		t.Fatalf("first Download() error = %v", err)
	}
	second, err := client.Download(t.Context(), "second")
	if err != nil {
		t.Fatalf("second Download() error = %v", err)
	}

	source := &admissionReader{}
	if _, err := client.Upload(t.Context(), "upload", source, objectstorage.UploadOptions{ContentLength: 1, Intent: objectstorage.UploadReplace}); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("Upload() kind = %q, want busy", objectstorage.Kind(err))
	}
	if _, err := client.Download(t.Context(), "third"); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("Download() kind = %q, want busy", objectstorage.Kind(err))
	}
	if _, err := client.Metadata(t.Context(), "metadata"); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("Metadata() kind = %q, want busy", objectstorage.Kind(err))
	}
	if err := client.Delete(t.Context(), "delete"); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("Delete() kind = %q, want busy", objectstorage.Kind(err))
	}
	if _, err := client.PresignGET(t.Context(), "presign", time.Second); objectstorage.Kind(err) != objectstorage.KindBusy {
		t.Fatalf("PresignGET() kind = %q, want busy", objectstorage.Kind(err))
	}
	if source.reads != 0 || source.closes != 1 || requests != 2 {
		t.Fatalf("saturated operations read source=%d, closed source=%d, and made requests=%d; want 0, 1, and 2", source.reads, source.closes, requests)
	}

	if _, err := io.ReadAll(first.Body); err != nil {
		t.Fatalf("read first download: %v", err)
	}
	if _, err := client.Metadata(t.Context(), "metadata"); err != nil {
		t.Fatalf("Metadata() after EOF error = %v", err)
	}
	if _, err := client.Metadata(t.Context(), "metadata"); objectstorage.Kind(err) != objectstorage.KindTemporary {
		t.Fatalf("Metadata() error release kind = %q, want temporary", objectstorage.Kind(err))
	}
	if err := second.Body.Close(); err != nil {
		t.Fatalf("close second download: %v", err)
	}
	if err := second.Body.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := client.Delete(t.Context(), "delete"); err != nil {
		t.Fatalf("Delete() after Close error = %v", err)
	}
	if got := len(client.tokens); got != 0 {
		t.Fatalf("admission tokens still held = %d", got)
	}
}

func TestEffectiveDeadlineAndLifecycleOwnEveryPhase(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		cfg := validConfig(ProviderAmazonS3)
		cfg.MaxOperationDuration = 2 * time.Second
		deadline := time.Now().Add(time.Second)
		client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
			got, ok := request.Context().Deadline()
			if !ok || !got.Equal(deadline) {
				t.Fatalf("request deadline = %s, present=%t; want caller deadline %s", got, ok, deadline)
			}
			switch request.Method {
			case http.MethodHead:
				return s3Response(http.StatusOK, http.Header{"Content-Length": []string{"1"}, "Last-Modified": []string{time.Now().UTC().Format(http.TimeFormat)}}, ""), nil
			case http.MethodGet:
				return admissionDownloadResponse("x"), nil
			default:
				t.Fatalf("unexpected request %s", request.Method)
				return nil, errors.New("unexpected request")
			}
		})
		ctx, cancel := context.WithDeadline(t.Context(), deadline)
		defer cancel()
		if _, err := client.Metadata(ctx, "object"); err != nil {
			t.Fatalf("Metadata() error = %v", err)
		}
		download, err := client.Download(ctx, "object")
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		if err := download.Body.Close(); err != nil {
			t.Fatalf("Download().Body.Close() error = %v", err)
		}
		if err := download.Body.Close(); err != nil {
			t.Fatalf("second Download().Body.Close() error = %v", err)
		}

		expired, stop := context.WithCancel(t.Context())
		stop()
		source := &admissionReader{}
		if _, err := client.Upload(expired, "upload", source, objectstorage.UploadOptions{ContentLength: 1, Intent: objectstorage.UploadReplace}); objectstorage.Kind(err) != objectstorage.KindCancelled {
			t.Fatalf("cancelled Upload() kind = %q, want cancelled", objectstorage.Kind(err))
		}
		if source.reads != 0 || source.closes != 1 {
			t.Fatalf("cancelled upload read source %d times and closed it %d times", source.reads, source.closes)
		}
		if got := len(client.tokens); got != 0 {
			t.Fatalf("admission tokens still held = %d", got)
		}
	})
}

func admissionDownloadResponse(body string) *http.Response {
	checksum := testCRC64NVME([]byte(body))
	return &http.Response{StatusCode: http.StatusOK, Status: "200", Header: http.Header{
		"Content-Length":           []string{strconv.Itoa(len(body))},
		"X-Amz-Checksum-Crc64nvme": []string{checksum},
		"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
	}, ContentLength: int64(len(body)), Body: io.NopCloser(strings.NewReader(body))}
}

type admissionReader struct {
	reads  int
	closes int
}

func (r *admissionReader) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}

func (r *admissionReader) Close() error {
	r.closes++
	return nil
}

const (
	envelopeChildEnv       = "S3_ENVELOPE_CHILD"
	envelopeBundleEnv      = "S3_ENVELOPE_BUNDLE"
	envelopeFixtureEnv     = "S3_ENVELOPE_FIXTURE_ADDR"
	envelopeControlFDEnv   = "S3_ENVELOPE_CONTROL_FD"
	envelopeReportFDEnv    = "S3_ENVELOPE_REPORT_FD"
	envelopeRecordBytes    = 128
	envelopeRecordPhase    = 1
	envelopeRecordComplete = 2
)

type envelopeRecord struct {
	kind              uint64
	phase             uint64
	baselineRSS       uint64
	peakRSS           uint64
	peakDelta         uint64
	ceiling           uint64
	construction      uint64
	idle              uint64
	vmRSS             uint64
	heap              uint64
	stack             uint64
	baselineGoroutine uint64
	finalGoroutine    uint64
	activeTokens      uint64
	activeConnections uint64
}

func TestLinuxProcessEnvelope(t *testing.T) {
	t.Parallel()
	if os.Getenv(envelopeChildEnv) == "1" {
		runLinuxProcessEnvelopeChild(t)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("T9 Linux envelope runs through make test-s3-envelope")
	}
	if runtime.GOMAXPROCS(0) != 1 {
		t.Skip("T9 Linux envelope runs through make test-s3-envelope with GOMAXPROCS=1")
	}

	bundle, root, signer := envelopeRootBundle(t)
	bundlePath := t.TempDir() + "/roots.pem"
	if err := os.WriteFile(bundlePath, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	var maxDelta uint64
	for run := 1; run <= 5; run++ {
		fixture := newEnvelopeFixture(t, envelopeLeafCertificate(t, root, signer))
		record := runEnvelopeChild(t, bundlePath, fixture)
		if record.peakDelta > record.ceiling {
			t.Fatalf("run %d peak delta = %d, ceiling = %d", run, record.peakDelta, record.ceiling)
		}
		if record.activeTokens != 0 || record.activeConnections != 0 || record.finalGoroutine > record.baselineGoroutine {
			t.Fatalf("run %d child cleanup tokens=%d connections=%d goroutines=%d baseline=%d", run, record.activeTokens, record.activeConnections, record.finalGoroutine, record.baselineGoroutine)
		}
		fixture.stop()
		fixture.waitForFinished(t, int(envelopeRequestsAtPhase[len(envelopeRequestsAtPhase)-1]))
		fixture.waitForConnections(t)
		if got := fixture.active(); got != (envelopeFixtureCounters{}) {
			t.Fatalf("run %d fixture cleanup = %+v", run, got)
		}
		maxDelta = max(maxDelta, record.peakDelta)
		t.Logf("run %d smaps baseline=%d peak=%d delta=%d ceiling=%d construction=%d idle=%d diagnostics vmrss=%d heap=%d stack=%d goroutines=%d", run, record.baselineRSS, record.peakRSS, record.peakDelta, record.ceiling, record.construction, record.idle, record.vmRSS, record.heap, record.stack, record.finalGoroutine)
	}
	t.Logf("five-run maximum smaps delta=%d", maxDelta)
}

const (
	envelopeNetworkPhases = 5
	envelopeHeaderBytes   = 1024
	envelopeControlBytes  = 1024
)

var (
	envelopeRequestsAtPhase  = [...]int64{2, 4, 6, 8, 10}
	envelopeResponsesAtPhase = [...]int64{2, 2, 2, 2, 4}
)

func runEnvelopeChild(t *testing.T, bundlePath string, fixture *envelopeFixture) envelopeRecord {
	t.Helper()
	controlRead, controlWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reportRead, reportWrite, err := os.Pipe()
	if err != nil {
		_ = controlRead.Close()
		_ = controlWrite.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = controlWrite.Close()
		_ = reportRead.Close()
	})

	var output bytes.Buffer
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run", "^TestLinuxProcessEnvelope$", "-test.count=1")
	command.Env = append(os.Environ(),
		envelopeChildEnv+"=1",
		envelopeBundleEnv+"="+bundlePath,
		envelopeFixtureEnv+"="+fixture.address,
		envelopeControlFDEnv+"=3",
		envelopeReportFDEnv+"=4",
	)
	command.ExtraFiles = []*os.File{controlRead, reportWrite}
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	_ = controlRead.Close()
	_ = reportWrite.Close()

	for phase := uint64(1); phase <= envelopeNetworkPhases; phase++ {
		record, err := readEnvelopeRecord(reportRead)
		if err != nil {
			_ = command.Wait()
			t.Fatalf("read child phase %d: %v\n%s", phase, err, output.String())
		}
		if record.kind != envelopeRecordPhase || record.phase != phase {
			_ = command.Wait()
			t.Fatalf("child phase record = %+v\n%s", record, output.String())
		}
		fixture.waitForRequests(t, envelopeRequestsAtPhase[phase-1])
		fixture.waitForResponses(t, envelopeResponsesAtPhase[phase-1])
		if _, err := controlWrite.Write([]byte{1}); err != nil {
			_ = command.Wait()
			t.Fatalf("release child phase %d: %v", phase, err)
		}
	}
	record, err := readEnvelopeRecord(reportRead)
	if err != nil {
		_ = command.Wait()
		t.Fatalf("read child report: %v\n%s", err, output.String())
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("child failed: %v\n%s", err, output.String())
	}
	if record.kind != envelopeRecordComplete {
		t.Fatalf("child completion record = %+v", record)
	}
	return record
}

func runLinuxProcessEnvelopeChild(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("Linux child lost the required execution envelope")
	}
	control := envelopeFile(t, envelopeControlFDEnv)
	report := envelopeFile(t, envelopeReportFDEnv)
	fixtureAddress := os.Getenv(envelopeFixtureEnv)
	if fixtureAddress == "" {
		t.Fatal("missing envelope fixture address")
	}

	runtime.GC()
	baseline := readSmapsRollup(t)
	peak := baseline
	sample := func() uint64 {
		current := readSmapsRollup(t)
		if current.rss < baseline.rss {
			t.Fatalf("smaps RSS decreased below baseline: baseline=%d current=%d", baseline.rss, current.rss)
		}
		if current.rss > peak.rss {
			peak = current
		}
		return current.rss - baseline.rss
	}

	cfg := envelopeConfig(t)
	bundle, err := os.ReadFile(os.Getenv(envelopeBundleEnv))
	if err != nil || len(bundle) != maxImageRootBundleBytes {
		t.Fatalf("read envelope bundle: bytes=%d err=%v", len(bundle), err)
	}
	client, err := newClient(cfg, memoryImageRootSource(bundle, 0, nil))
	if err != nil {
		t.Fatal(err)
	}
	construction := sample()
	runtime.KeepAlive(bundle)
	runtime.GC()
	idle := sample()
	base := newEnvelopeHTTPClient(t, client.roots, client.transport.endpoint.Host, fixtureAddress, cfg.MaxResponseHeaderBytes)
	client.transport.base = base
	leakBaseline := goleak.IgnoreCurrent()
	baselineGoroutines := uint64(runtime.NumGoroutine())

	key := strings.Repeat("k", int(maximumKeyBytes)-1)
	chunkReads := new(atomic.Int64)
	for phase, test := range []envelopePhase{
		{headerBytes: envelopeControlHeaderBytes(), responseBytes: envelopeControlBytes, operation: func(ctx context.Context) error { return client.Delete(ctx, key+"d") }},
		{sourceReads: chunkReads, operation: func(ctx context.Context) error {
			source := envelopeBlockingReader{done: make(chan struct{}), reads: chunkReads}
			_, err := client.Upload(ctx, key+"u", &source, objectstorage.UploadOptions{ContentLength: minimumMultipartChunk, ContentType: strings.Repeat("a", maximumContentTypeBytes), Intent: objectstorage.UploadReplace})
			return err
		}},
		{allowSuccess: true, operation: func(ctx context.Context) error { return envelopeDownload(ctx, client, key+"o") }},
		{allowSuccess: true, operation: func(ctx context.Context) error { return envelopeDownload(ctx, client, key+"c") }},
		{headerBytes: envelopeControlHeaderBytes(), responseBytes: envelopeControlBytes, operation: func(ctx context.Context) error { return envelopeComplete(ctx, client, key+"m") }},
	} {
		runEnvelopePair(t, control, report, uint64(phase+1), test, base, sample)
	}
	if _, err := client.PresignGET(t.Context(), key, time.Second); err != nil {
		t.Fatalf("PresignGET() error = %v", err)
	}
	client.Close()
	base.close()
	if err := goleak.Find(leakBaseline); err != nil {
		t.Fatalf("child goroutine cleanup: %v", err)
	}

	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	writeEnvelopeRecord(t, report, envelopeRecord{
		kind:              envelopeRecordComplete,
		baselineRSS:       baseline.rss,
		peakRSS:           peak.rss,
		peakDelta:         peak.rss - baseline.rss,
		ceiling:           uint64(cfg.MaxWorkingMemoryBytes),
		construction:      construction,
		idle:              idle,
		vmRSS:             readVMRSS(t),
		heap:              memory.HeapAlloc,
		stack:             memory.StackInuse,
		baselineGoroutine: baselineGoroutines,
		finalGoroutine:    uint64(runtime.NumGoroutine()),
		activeTokens:      uint64(len(client.tokens)),
		activeConnections: uint64(base.active.Load()),
	})
}

func envelopeConfig(t *testing.T) Config {
	t.Helper()
	cfg := validConfig(ProviderAmazonS3)
	cfg.MaxObjectBytes = maximumPartCount * minimumMultipartChunk
	cfg.MaxOperationDuration = 30 * time.Second
	cfg.MaxResponseHeaderBytes = envelopeHeaderBytes
	cfg.MaxControlResponseBytes = envelopeControlBytes
	required, ok := cfg.requiredMemory()
	if !ok || required != 310_099_108 {
		t.Fatalf("envelope required memory = %d, %t", required, ok)
	}
	cfg.MaxWorkingMemoryBytes = required
	return cfg
}

type envelopePhase struct {
	operation     func(context.Context) error
	headerBytes   int64
	responseBytes int64
	allowSuccess  bool
	sourceReads   *atomic.Int64
}

func runEnvelopePair(t *testing.T, control *os.File, report *os.File, phase uint64, test envelopePhase, base *envelopeHTTPClient, sample func() uint64) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	results := make(chan error, 2)
	baselineHeaderBytes := base.headerBytes.Load()
	baselineResponseBytes := base.responseBytes.Load()
	baselineSourceReads := int64(0)
	if test.sourceReads != nil {
		baselineSourceReads = test.sourceReads.Load()
	}
	for range 2 {
		go func() { results <- test.operation(ctx) }()
	}
	writeEnvelopeRecord(t, report, envelopeRecord{kind: envelopeRecordPhase, phase: phase})
	var release [1]byte
	if _, err := io.ReadFull(control, release[:]); err != nil || release[0] != 1 {
		t.Fatalf("wait for controller phase %d: %v", phase, err)
	}
	if test.headerBytes != 0 {
		base.waitForHeaderBytes(t, baselineHeaderBytes+2*test.headerBytes)
	}
	if test.responseBytes != 0 {
		base.waitForResponseBytes(t, baselineResponseBytes+2*test.responseBytes)
	} else if got := base.responseBytes.Load(); got != baselineResponseBytes {
		t.Fatalf("phase %d buffered object bytes = %d", phase, got-baselineResponseBytes)
	}
	if test.sourceReads != nil && test.sourceReads.Load() != baselineSourceReads+2 {
		t.Fatalf("phase %d source reads = %d, want %d", phase, test.sourceReads.Load()-baselineSourceReads, 2)
	}
	_ = sample()
	cancel()
	for range 2 {
		if err := <-results; err == nil && !test.allowSuccess {
			t.Fatalf("phase %d completed before cancellation", phase)
		}
	}
}

type envelopeBlockingReader struct {
	done  chan struct{}
	reads *atomic.Int64
	once  sync.Once
}

func (r *envelopeBlockingReader) Read([]byte) (int, error) {
	r.reads.Add(1)
	<-r.done
	return 0, context.Canceled
}

func (r *envelopeBlockingReader) Close() error {
	r.once.Do(func() { close(r.done) })
	return nil
}

func envelopeDownload(ctx context.Context, client *Client, key string) error {
	result, err := client.Download(ctx, key)
	if err != nil {
		return err
	}
	<-ctx.Done()
	if err := result.Body.Close(); err != nil {
		return fmt.Errorf("close envelope download: %w", err)
	}
	return nil
}

func envelopeComplete(ctx context.Context, client *Client, key string) (err error) {
	call := client.telemetry.begin(ctx, telemetryOperationUpload)
	//nolint:contextcheck // finish ends the span without deriving a new context.
	defer func() { call.finish(err, 0) }()
	effective, release, err := client.admit(ctx, call)
	if err != nil {
		return err
	}
	defer release()
	etag := strings.Repeat("&", int(client.config.MaxResponseHeaderBytes))
	checksum := "AAAAAAAAAAA="
	parts := make([]types.CompletedPart, maximumPartCount)
	for index := range parts {
		partNumber := int32(index + 1)
		parts[index] = types.CompletedPart{PartNumber: &partNumber, ETag: &etag, ChecksumCRC64NVME: &checksum}
	}
	_, err = client.sdk.CompleteMultipartUpload(effective, &awss3.CompleteMultipartUploadInput{
		Bucket:            aws.String(client.config.Bucket),
		Key:               aws.String(key),
		UploadId:          aws.String("upload"),
		ChecksumCRC64NVME: aws.String(checksum),
		ChecksumType:      types.ChecksumTypeFullObject,
		MpuObjectSize:     aws.Int64(client.config.MaxObjectBytes),
		MultipartUpload:   &types.CompletedMultipartUpload{Parts: parts},
	})
	return err //nolint:wrapcheck // The envelope must preserve the SDK result for telemetry.
}

type envelopeHTTPClient struct {
	*http.Client

	active        atomic.Int64
	headerBytes   atomic.Int64
	responseBytes atomic.Int64
	transport     *http.Transport
}

func newEnvelopeHTTPClient(t *testing.T, roots *x509.CertPool, serverName, fixtureAddress string, headerLimit int64) *envelopeHTTPClient {
	t.Helper()
	if roots == nil {
		t.Fatal("envelope roots are nil")
	}
	result := &envelopeHTTPClient{}
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	transport := &http.Transport{
		DisableCompression:     true,
		DisableKeepAlives:      true,
		MaxConnsPerHost:        2,
		MaxIdleConnsPerHost:    0,
		MaxResponseHeaderBytes: headerLimit,
		Protocols:              protocols,
		TLSClientConfig:        &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var dialer net.Dialer
			connection, err := dialer.DialContext(ctx, network, fixtureAddress)
			if err != nil {
				return nil, fmt.Errorf("dial envelope fixture: %w", err)
			}
			result.active.Add(1)
			return &envelopeConn{Conn: connection, active: &result.active}, nil
		},
	}
	result.transport = transport
	result.Client = &http.Client{Transport: envelopeResponseTransport{base: transport, headers: &result.headerBytes, bytes: &result.responseBytes}}
	return result
}

func (c *envelopeHTTPClient) close() {
	c.transport.CloseIdleConnections()
}

func (c *envelopeHTTPClient) waitForResponseBytes(t *testing.T, want int64) {
	t.Helper()
	waittest.UntilFunc(t, 10*time.Second, func() bool { return c.responseBytes.Load() >= want }, func() string {
		return fmt.Sprintf("response bytes = %d, want at least %d", c.responseBytes.Load(), want)
	})
}

func (c *envelopeHTTPClient) waitForHeaderBytes(t *testing.T, want int64) {
	t.Helper()
	waittest.UntilFunc(t, 10*time.Second, func() bool { return c.headerBytes.Load() >= want }, func() string {
		return fmt.Sprintf("response header bytes = %d, want at least %d", c.headerBytes.Load(), want)
	})
}

type envelopeResponseTransport struct {
	base    http.RoundTripper
	headers *atomic.Int64
	bytes   *atomic.Int64
}

func (t envelopeResponseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if response != nil && response.Body != nil {
		t.headers.Add(envelopeHeaderSize(response.Header))
		response.Body = &envelopeResponseBody{ReadCloser: response.Body, bytes: t.bytes}
	}
	if err != nil {
		return response, fmt.Errorf("round trip envelope response: %w", err)
	}
	return response, nil
}

func envelopeHeaderSize(headers http.Header) int64 {
	var data bytes.Buffer
	if err := headers.Write(&data); err != nil {
		panic(err)
	}
	return int64(data.Len())
}

type envelopeResponseBody struct {
	io.ReadCloser

	bytes *atomic.Int64
}

func (b *envelopeResponseBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	b.bytes.Add(int64(n))
	return n, err //nolint:wrapcheck // Reader callers require io.EOF identity.
}

type envelopeConn struct {
	net.Conn

	active *atomic.Int64
	once   sync.Once
}

func (c *envelopeConn) Close() error {
	c.once.Do(func() { c.active.Add(-1) })
	if err := c.Conn.Close(); err != nil {
		return fmt.Errorf("close envelope connection: %w", err)
	}
	return nil
}

type smapsRollup struct {
	rss          uint64
	sharedClean  uint64
	sharedDirty  uint64
	privateClean uint64
	privateDirty uint64
}

func readSmapsRollup(t *testing.T) smapsRollup {
	t.Helper()
	data, err := os.ReadFile("/proc/self/smaps_rollup")
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseSmapsRollup(string(data))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func parseSmapsRollup(data string) (smapsRollup, error) {
	expected := map[string]struct{}{
		"Rss":           {},
		"Shared_Clean":  {},
		"Shared_Dirty":  {},
		"Private_Clean": {},
		"Private_Dirty": {},
	}
	values := make(map[string]uint64, len(expected))
	for line := range strings.SplitSeq(data, "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if _, ok := expected[name]; !ok {
			continue
		}
		if _, duplicate := values[name]; duplicate {
			return smapsRollup{}, fmt.Errorf("smaps_rollup duplicates %s", name)
		}
		fields := strings.Fields(rest)
		if len(fields) != 2 || fields[1] != "kB" {
			return smapsRollup{}, fmt.Errorf("smaps_rollup has invalid %s", name)
		}
		value, parseErr := strconv.ParseUint(fields[0], 10, 64)
		if parseErr != nil {
			return smapsRollup{}, fmt.Errorf("smaps_rollup parses %s: %w", name, parseErr)
		}
		if value > math.MaxUint64/1024 {
			return smapsRollup{}, fmt.Errorf("smaps_rollup converts %s", name)
		}
		values[name] = value * 1024
	}
	for name := range expected {
		if _, found := values[name]; !found {
			return smapsRollup{}, fmt.Errorf("smaps_rollup misses %s", name)
		}
	}
	result := smapsRollup{rss: values["Rss"], sharedClean: values["Shared_Clean"], sharedDirty: values["Shared_Dirty"], privateClean: values["Private_Clean"], privateDirty: values["Private_Dirty"]}
	resident := uint64(0)
	for _, value := range []uint64{result.sharedClean, result.sharedDirty, result.privateClean, result.privateDirty} {
		if value > math.MaxUint64-resident {
			return smapsRollup{}, errors.New("smaps_rollup sums resident categories")
		}
		resident += value
	}
	if result.rss == 0 || result.rss != resident {
		return smapsRollup{}, fmt.Errorf("smaps_rollup identity rss=%d shared_clean=%d shared_dirty=%d private_clean=%d private_dirty=%d", result.rss, result.sharedClean, result.sharedDirty, result.privateClean, result.privateDirty)
	}
	return result, nil
}

func TestSmapsRollupParserIsFailClosed(t *testing.T) {
	t.Parallel()
	valid := "Rss: 5 kB\nShared_Clean: 1 kB\nShared_Dirty: 1 kB\nPrivate_Clean: 1 kB\nPrivate_Dirty: 2 kB\n"
	if _, err := parseSmapsRollup(valid); err != nil {
		t.Fatalf("parseSmapsRollup(valid) error = %v", err)
	}
	for _, snapshot := range []string{
		"Rss: 5 kB\nShared_Clean: 1 kB\nShared_Dirty: 1 kB\nPrivate_Clean: 1 kB\n",
		valid + "Rss: 5 kB\n",
		"Rss: 5 kB\nShared_Clean: 1 bytes\nShared_Dirty: 1 kB\nPrivate_Clean: 1 kB\nPrivate_Dirty: 2 kB\n",
		"Rss: 5 kB\nShared_Clean: 18446744073709551615 kB\nShared_Dirty: 1 kB\nPrivate_Clean: 1 kB\nPrivate_Dirty: 2 kB\n",
		"Rss: 1 kB\nShared_Clean: 18014398509481983 kB\nShared_Dirty: 18014398509481983 kB\nPrivate_Clean: 1 kB\nPrivate_Dirty: 1 kB\n",
	} {
		if _, err := parseSmapsRollup(snapshot); err == nil {
			t.Fatalf("parseSmapsRollup(%q) error = nil", snapshot)
		}
	}
}

func readVMRSS(t *testing.T) uint64 {
	t.Helper()
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "VmRSS:" && fields[2] == "kB" {
			value, parseErr := strconv.ParseUint(fields[1], 10, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			return value * 1024
		}
	}
	t.Fatal("VmRSS is missing")
	return 0
}

func envelopeFile(t *testing.T, key string) *os.File {
	t.Helper()
	fd, err := strconv.Atoi(os.Getenv(key))
	if err != nil || fd < 3 {
		t.Fatalf("%s = %q", key, os.Getenv(key))
	}
	return os.NewFile(uintptr(fd), key)
}

func writeEnvelopeRecord(t *testing.T, file *os.File, record envelopeRecord) {
	t.Helper()
	data := make([]byte, envelopeRecordBytes)
	values := []uint64{record.kind, record.phase, record.baselineRSS, record.peakRSS, record.peakDelta, record.ceiling, record.construction, record.idle, record.vmRSS, record.heap, record.stack, record.baselineGoroutine, record.finalGoroutine, record.activeTokens, record.activeConnections}
	for index, value := range values {
		binary.LittleEndian.PutUint64(data[index*8:], value)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
}

func readEnvelopeRecord(file *os.File) (envelopeRecord, error) {
	data := make([]byte, envelopeRecordBytes)
	if _, err := io.ReadFull(file, data); err != nil {
		return envelopeRecord{}, fmt.Errorf("read envelope record: %w", err)
	}
	values := make([]uint64, envelopeRecordBytes/8)
	for index := range values {
		values[index] = binary.LittleEndian.Uint64(data[index*8:])
	}
	return envelopeRecord{kind: values[0], phase: values[1], baselineRSS: values[2], peakRSS: values[3], peakDelta: values[4], ceiling: values[5], construction: values[6], idle: values[7], vmRSS: values[8], heap: values[9], stack: values[10], baselineGoroutine: values[11], finalGoroutine: values[12], activeTokens: values[13], activeConnections: values[14]}, nil
}

type envelopeFixtureCounters struct {
	requests    int64
	bodies      int64
	connections int64
	handlers    int64
}

type envelopeFixture struct {
	address  string
	server   *http.Server
	listener net.Listener

	requests       atomic.Int64
	responses      atomic.Int64
	activeRequests atomic.Int64
	bodies         atomic.Int64
	connections    atomic.Int64
	handlers       atomic.Int64
	finished       chan struct{}
	stopHandlers   chan struct{}
}

func newEnvelopeFixture(t *testing.T, certificate tls.Certificate) *envelopeFixture {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &envelopeFixture{address: listener.Addr().String(), listener: listener, finished: make(chan struct{}, envelopeNetworkPhases*2), stopHandlers: make(chan struct{})}
	fixture.server = &http.Server{
		Handler: http.HandlerFunc(fixture.serve),
		ConnState: func(_ net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				fixture.connections.Add(1)
			case http.StateClosed, http.StateHijacked:
				fixture.connections.Add(-1)
			case http.StateActive, http.StateIdle:
			}
		},
	}
	go func() {
		_ = fixture.server.Serve(tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}}))
	}()
	t.Cleanup(func() {
		_ = fixture.server.Close()
		_ = listener.Close()
	})
	return fixture
}

func (f *envelopeFixture) serve(writer http.ResponseWriter, request *http.Request) {
	stopHandlers := f.stopHandlers
	f.requests.Add(1)
	f.activeRequests.Add(1)
	f.handlers.Add(1)
	defer func() {
		f.activeRequests.Add(-1)
		f.handlers.Add(-1)
		f.finished <- struct{}{}
	}()
	if request.Body != nil && request.Body != http.NoBody {
		f.bodies.Add(1)
		defer func() {
			f.bodies.Add(-1)
		}()
	}
	switch {
	case request.Method == http.MethodGet:
		length := int64(maximumPartCount) * minimumMultipartChunk
		if strings.HasSuffix(request.URL.Path, "c") {
			length = minimumMultipartChunk
		}
		writer.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		writer.Header().Set("X-Amz-Checksum-Crc64nvme", "AAAAAAAAAAA=")
		writer.Header().Set("X-Amz-Checksum-Type", "FULL_OBJECT")
		setEnvelopeHeader(writer.Header())
		writer.WriteHeader(http.StatusOK)
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
	case request.Method == http.MethodPut && strings.HasSuffix(request.URL.Path, "u"):
	case request.Method == http.MethodDelete || request.Method == http.MethodPut || request.Method == http.MethodPost && request.URL.Query().Get("uploadId") == "upload":
		f.writeControlError(writer)
	}
	select {
	case <-request.Context().Done():
	case <-stopHandlers:
	}
}

func (f *envelopeFixture) writeControlError(writer http.ResponseWriter) {
	body := envelopeErrorBody()
	writer.Header().Set("Content-Type", "application/xml")
	setEnvelopeControlHeader(writer.Header())
	writer.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(writer, body)
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
	f.responses.Add(1)
}

func setEnvelopeHeader(headers http.Header) {
	headers.Set("Date", "Thu, 01 Jan 1970 00:00:00 GMT")
	headers.Set("X-Envelope-Header", strings.Repeat("h", 128))
}

func envelopeControlHeaderBytes() int64 {
	return envelopeHeaderBytes - int64(len("HTTP/1.1 500 Internal Server Error\r\n")+len("Transfer-Encoding: chunked\r\n")+len("Connection: close\r\n")+len("\r\n"))
}

func setEnvelopeControlHeader(headers http.Header) {
	headers.Set("Date", "Thu, 01 Jan 1970 00:00:00 GMT")
	headers.Set("X-Envelope-Header", "")
	valueBytes := envelopeControlHeaderBytes() - envelopeHeaderSize(headers)
	if valueBytes < 0 {
		panic("T9 envelope control headers exceed their bound")
	}
	headers.Set("X-Envelope-Header", strings.Repeat("h", int(valueBytes)))
	if envelopeHeaderSize(headers) != envelopeControlHeaderBytes() {
		panic("T9 envelope control headers do not reach their wire bound")
	}
}

func envelopeErrorBody() string {
	const prefix = "<Error><Code>InternalError</Code><Message>"
	const suffix = "</Message></Error>"
	if envelopeControlBytes <= int64(len(prefix)+len(suffix)) {
		panic("T9 envelope control body is too small")
	}
	return prefix + strings.Repeat("e", int(envelopeControlBytes)-len(prefix)-len(suffix)) + suffix
}

func (f *envelopeFixture) stop() {
	close(f.stopHandlers)
	_ = f.server.Close()
	_ = f.listener.Close()
}

func (f *envelopeFixture) waitForRequests(t *testing.T, want int64) {
	t.Helper()
	waittest.UntilFunc(t, 10*time.Second, func() bool { return f.requests.Load() >= want }, func() string {
		return fmt.Sprintf("fixture requests = %d, want at least %d", f.requests.Load(), want)
	})
}

func (f *envelopeFixture) waitForResponses(t *testing.T, want int64) {
	t.Helper()
	waittest.UntilFunc(t, 10*time.Second, func() bool { return f.responses.Load() >= want }, func() string {
		return fmt.Sprintf("fixture responses = %d, want at least %d", f.responses.Load(), want)
	})
}

func (f *envelopeFixture) waitForFinished(t *testing.T, count int) {
	t.Helper()
	for range count {
		select {
		case <-f.finished:
		case <-time.After(10 * time.Second):
			t.Fatalf("fixture handler did not exit: active_requests=%d bodies=%d connections=%d handlers=%d total_requests=%d", f.activeRequests.Load(), f.bodies.Load(), f.connections.Load(), f.handlers.Load(), f.requests.Load())
		}
	}
}

func (f *envelopeFixture) waitForConnections(t *testing.T) {
	t.Helper()
	waittest.UntilFunc(t, 10*time.Second, func() bool { return f.connections.Load() == 0 }, func() string {
		return fmt.Sprintf("fixture connections = %d", f.connections.Load())
	})
}

func (f *envelopeFixture) active() envelopeFixtureCounters {
	return envelopeFixtureCounters{requests: f.activeRequests.Load(), bodies: f.bodies.Load(), connections: f.connections.Load(), handlers: f.handlers.Load()}
}

func envelopeLeafCertificate(t *testing.T, root *x509.Certificate, signer crypto.Signer) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "examplebucket.s3.us-east-1.amazonaws.com"},
		DNSNames:     []string{"examplebucket.s3.us-east-1.amazonaws.com"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(1<<31, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, root, &key.PublicKey, signer)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der, root.Raw}, PrivateKey: key}
}
