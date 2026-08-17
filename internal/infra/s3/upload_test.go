package s3

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"net/http/httptrace"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/objectstorage"
)

type httpDoerFunc func(*http.Request) (*http.Response, error)

func (f httpDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUploadRejectsUnboundedContentType(t *testing.T) {
	t.Parallel()
	client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid Content-Type reached provider transport")
		return nil, errors.New("unreachable")
	})

	for _, contentType := range []string{"", "a", strings.Repeat("a", maximumContentTypeBytes)} {
		if err := client.validateUpload("object", uploadSource(strings.NewReader("x")), objectstorage.UploadOptions{
			ContentLength: 1,
			ContentType:   contentType,
			Intent:        objectstorage.UploadReplace,
		}); err != nil {
			t.Fatalf("Content-Type length %d validateUpload() error = %v", len(contentType), err)
		}
	}

	for _, contentType := range []string{
		strings.Repeat("a", maximumContentTypeBytes+1),
		"text/plain\r\nX-Injected: true",
		"text/plain\x00",
	} {
		source := &admissionReader{}
		if _, err := client.Upload(t.Context(), "object", source, objectstorage.UploadOptions{
			ContentLength: 1,
			ContentType:   contentType,
			Intent:        objectstorage.UploadReplace,
		}); objectstorage.Kind(err) != objectstorage.KindInvalid {
			t.Fatalf("Upload() kind = %q, want invalid", objectstorage.Kind(err))
		}
		if source.reads != 0 || source.closes != 1 || len(client.tokens) != 0 {
			t.Fatalf("invalid Content-Type read source %d times, closed it %d times, and held %d admissions", source.reads, source.closes, len(client.tokens))
		}
	}

	oversized := validConfig(ProviderAmazonS3)
	source := &admissionReader{}
	if _, err := client.Upload(t.Context(), "object", source, objectstorage.UploadOptions{
		ContentLength: oversized.MaxObjectBytes + 1,
		Intent:        objectstorage.UploadReplace,
	}); objectstorage.Kind(err) != objectstorage.KindTooLarge {
		t.Fatalf("oversized Upload() kind = %q, want too_large", objectstorage.Kind(err))
	}
	if source.reads != 0 || source.closes != 1 {
		t.Fatalf("oversized Upload() read source %d times and closed it %d times", source.reads, source.closes)
	}
}

func TestUploadCancellationClosesBlockedSourceAndReleasesAdmission(t *testing.T) {
	t.Parallel()
	cfg := validConfig(ProviderAmazonS3)
	cfg.MaxActiveOperations = 1
	cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
	source := newBlockingUploadSource()
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		httptrace.ContextClientTrace(request.Context()).WroteHeaders()
		_, err := io.Copy(io.Discard, request.Body)
		return nil, fmt.Errorf("copy blocked upload body: %w", err)
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := client.Upload(ctx, "object", source, objectstorage.UploadOptions{ContentLength: 1, Intent: objectstorage.UploadReplace})
		done <- err
	}()
	select {
	case <-source.entered:
	case <-time.After(time.Second):
		t.Fatal("upload source was not read")
	}
	cancel()
	select {
	case err := <-done:
		if objectstorage.Kind(err) != objectstorage.KindOutcomeUnknown {
			t.Fatalf("cancelled possible-send Upload() kind = %q, want outcome_unknown", objectstorage.Kind(err))
		}
	case <-time.After(time.Second):
		t.Fatal("Upload did not return after cancellation closed its source")
	}
	if got := source.closes.Load(); got != 1 {
		t.Fatalf("blocked upload source closes = %d, want 1", got)
	}
	if got := len(client.tokens); got != 0 {
		t.Fatalf("admission tokens after blocked upload cancellation = %d, want 0", got)
	}
}

type blockingUploadSource struct {
	entered     chan struct{}
	release     chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
	closes      atomic.Int32
}

func newBlockingUploadSource() *blockingUploadSource {
	return &blockingUploadSource{entered: make(chan struct{}), release: make(chan struct{})}
}

func (s *blockingUploadSource) Read([]byte) (int, error) {
	s.enterOnce.Do(func() { close(s.entered) })
	<-s.release
	return 0, errors.New("upload source closed")
}

func (s *blockingUploadSource) Close() error {
	s.closes.Add(1)
	s.releaseOnce.Do(func() { close(s.release) })
	return nil
}

func TestSingleUploadStreamsCRC64NVMEAndExactLength(t *testing.T) {
	t.Parallel()
	data := []byte("the declared bytes")
	buffer := bytes.NewBuffer(append(slices.Clone(data), '!'))
	source := &transportGatedReader{t: t, reader: buffer}
	client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
		source.allow.Store(true)
		if request.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", request.Method)
		}
		if got := request.Header.Get("If-None-Match"); got != "*" {
			t.Fatalf("If-None-Match = %q, want *", got)
		}
		if got := request.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("Content-Type = %q, want text/plain", got)
		}
		if got := request.Header.Get("X-Amz-Decoded-Content-Length"); got != strconv.Itoa(len(data)) {
			t.Fatalf("decoded content length = %q, want %d", got, len(data))
		}
		if got := request.Header.Get("X-Amz-Sdk-Checksum-Algorithm"); got != "CRC64NVME" {
			t.Fatalf("checksum algorithm = %q", got)
		}
		payload, trailers := decodeAWSChunked(t, request.Body)
		if !bytes.Equal(payload, data) {
			t.Fatalf("uploaded payload = %q, want %q", payload, data)
		}
		if got := trailers.Get("X-Amz-Checksum-Crc64nvme"); got != testCRC64NVME(data) {
			t.Fatalf("trailer CRC64NVME = %q, want %q", got, testCRC64NVME(data))
		}
		return s3Response(http.StatusOK, http.Header{
			"X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(data)},
			"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
		}, ""), nil
	})

	result, err := client.Upload(context.Background(), "object", source, objectstorage.UploadOptions{
		ContentLength: int64(len(data)), ContentType: "text/plain", Intent: objectstorage.UploadCreateOnly,
	})
	if err != nil || result.Cleanup != objectstorage.CleanupNone {
		t.Fatalf("Upload() = %#v, %v", result, err)
	}
	if got := source.closes.Load(); got != 1 {
		t.Fatalf("successful upload source closes = %d, want 1", got)
	}
	if sentinel, err := buffer.ReadByte(); err != nil || sentinel != '!' {
		t.Fatalf("byte after declared length = %q, %v; want unread sentinel", sentinel, err)
	}

	t.Run("short source fails", func(t *testing.T) {
		t.Parallel()
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = io.Copy(io.Discard, request.Body)
			return s3Response(http.StatusOK, http.Header{
				"X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(data[:len(data)-1])},
				"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
			}, ""), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data[:len(data)-1])), objectstorage.UploadOptions{
			ContentLength: int64(len(data)), Intent: objectstorage.UploadReplace,
		})
		if err == nil {
			t.Fatal("short source unexpectedly succeeded")
		}
	})

	t.Run("mismatched confirmation fails", func(t *testing.T) {
		t.Parallel()
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = decodeAWSChunked(t, request.Body)
			return s3Response(http.StatusOK, http.Header{
				"X-Amz-Checksum-Crc64nvme": []string{"not-the-checksum"},
				"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
			}, ""), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data)), objectstorage.UploadOptions{
			ContentLength: int64(len(data)), Intent: objectstorage.UploadReplace,
		})
		if got := objectstorage.Kind(err); got != objectstorage.KindIntegrityFailed {
			t.Fatalf("mismatched confirmation kind = %q, want integrity_failed", got)
		}
	})

	t.Run("missing confirmation fails", func(t *testing.T) {
		t.Parallel()
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = decodeAWSChunked(t, request.Body)
			return s3Response(http.StatusOK, http.Header{"X-Amz-Checksum-Type": []string{"FULL_OBJECT"}}, ""), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data)), objectstorage.UploadOptions{ContentLength: int64(len(data)), Intent: objectstorage.UploadReplace})
		if got := objectstorage.Kind(err); got != objectstorage.KindIntegrityFailed {
			t.Fatalf("missing confirmation kind = %q, want integrity_failed", got)
		}
	})

	t.Run("threshold is still single", func(t *testing.T) {
		t.Parallel()
		threshold := validConfig(ProviderAmazonS3).MultipartChunkBytes
		data := bytes.Repeat([]byte("t"), int(threshold))
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPut {
				t.Fatalf("threshold request method = %s, want PUT", request.Method)
			}
			payload, _ := decodeAWSChunked(t, request.Body)
			if !bytes.Equal(payload, data) {
				t.Fatal("threshold request changed source bytes")
			}
			return s3Response(http.StatusOK, http.Header{
				"X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(data)},
				"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
			}, ""), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data)), objectstorage.UploadOptions{
			ContentLength: threshold, Intent: objectstorage.UploadReplace,
		})
		if err != nil {
			t.Fatalf("threshold Upload() error = %v", err)
		}
	})
}

type transportGatedReader struct {
	t      *testing.T
	reader io.Reader
	allow  atomic.Bool
	closes atomic.Int32
}

func (r *transportGatedReader) Read(p []byte) (int, error) {
	if !r.allow.Load() {
		r.t.Error("upload source was read before the transport started")
		return 0, errors.New("source read before transport")
	}
	return r.reader.Read(p) //nolint:wrapcheck // Reader callers require the source EOF identity.
}

func (r *transportGatedReader) Close() error {
	r.closes.Add(1)
	return nil
}

func TestMultipartUploadIsSerialAndConfirmsWholeChecksum(t *testing.T) {
	t.Parallel()
	cfg := validConfig(ProviderAmazonS3)
	cfg.MaxObjectBytes = 3*cfg.MultipartChunkBytes + 1
	cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
	data := bytes.Repeat([]byte("p"), int(cfg.MultipartChunkBytes)*2+3)
	part := 0
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		switch {
		case request.Method == http.MethodPost && query.Has("uploads"):
			return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>"), nil
		case request.Method == http.MethodPut && query.Get("partNumber") != "":
			part++
			payload, _ := decodeAWSChunked(t, request.Body)
			start := (part - 1) * int(cfg.MultipartChunkBytes)
			end := min(start+int(cfg.MultipartChunkBytes), len(data))
			if !bytes.Equal(payload, data[start:end]) {
				t.Fatalf("part %d bytes do not match declared source", part)
			}
			if got := request.Header.Get("X-Amz-Sdk-Checksum-Algorithm"); got != "CRC64NVME" {
				t.Fatalf("part %d algorithm = %q", part, got)
			}
			return s3Response(http.StatusOK, http.Header{
				"Etag":                     []string{fmt.Sprintf("part-%d", part)},
				"X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(payload)},
			}, ""), nil
		case request.Method == http.MethodPost && query.Get("uploadId") == "private-upload":
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := request.Header.Get("X-Amz-Checksum-Crc64nvme"), testCRC64NVME(data); got != want {
				t.Fatalf("completion checksum = %q, want %q", got, want)
			}
			if got := request.Header.Get("X-Amz-Checksum-Type"); got != "FULL_OBJECT" {
				t.Fatalf("completion checksum type = %q", got)
			}
			if got := request.Header.Get("X-Amz-Mp-Object-Size"); got != strconv.Itoa(len(data)) {
				t.Fatalf("completion object size = %q, want %d", got, len(data))
			}
			lastDescriptor := -1
			for index, checksum := range []string{
				testCRC64NVME(data[:cfg.MultipartChunkBytes]),
				testCRC64NVME(data[cfg.MultipartChunkBytes : 2*cfg.MultipartChunkBytes]),
				testCRC64NVME(data[2*cfg.MultipartChunkBytes:]),
			} {
				want := fmt.Sprintf("<ChecksumCRC64NVME>%s</ChecksumCRC64NVME><ETag>part-%d</ETag><PartNumber>%d</PartNumber>", checksum, index+1, index+1)
				position := bytes.Index(body, []byte(want))
				if position == -1 || position <= lastDescriptor {
					t.Fatalf("completion is missing ordered part descriptor %q", want)
				}
				lastDescriptor = position
			}
			return s3Response(http.StatusOK, nil, "<CompleteMultipartUploadResult><ChecksumCRC64NVME>"+testCRC64NVME(data)+"</ChecksumCRC64NVME><ChecksumType>FULL_OBJECT</ChecksumType></CompleteMultipartUploadResult>"), nil
		default:
			t.Fatalf("unexpected request %s %s", request.Method, request.URL)
			return nil, errors.New("unreachable")
		}
	})

	result, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data)), objectstorage.UploadOptions{
		ContentLength: int64(len(data)), Intent: objectstorage.UploadReplace,
	})
	if err != nil || result.Cleanup != objectstorage.CleanupNone {
		t.Fatalf("Upload() = %#v, %v", result, err)
	}
	if part != 3 {
		t.Fatalf("uploaded parts = %d, want 3", part)
	}

	t.Run("C+1", func(t *testing.T) {
		t.Parallel()
		assertMultipartSuccess(t, cfg, cfg.MultipartChunkBytes+1)
	})
	t.Run("exact multiple", func(t *testing.T) {
		t.Parallel()
		assertMultipartSuccess(t, cfg, 2*cfg.MultipartChunkBytes)
	})
	if got := partCount(cfg.MultipartChunkBytes*maximumPartCount, cfg.MultipartChunkBytes); got != maximumPartCount {
		t.Fatalf("maximum multipart count = %d, want %d", got, maximumPartCount)
	}
	assertMultipartCompletionFailures(t)
}

func assertMultipartSuccess(t *testing.T, cfg Config, length int64) {
	t.Helper()
	data := bytes.Repeat([]byte("b"), int(length))
	part := 0
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		switch {
		case request.Method == http.MethodPost && query.Has("uploads"):
			return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>"), nil
		case request.Method == http.MethodPut:
			part++
			if got := query.Get("partNumber"); got != strconv.Itoa(part) {
				t.Fatalf("part order = %q, want %d", got, part)
			}
			payload, _ := decodeAWSChunked(t, request.Body)
			start := (part - 1) * int(cfg.MultipartChunkBytes)
			end := min(start+int(cfg.MultipartChunkBytes), len(data))
			if !bytes.Equal(payload, data[start:end]) {
				t.Fatalf("part %d bytes changed", part)
			}
			return s3Response(http.StatusOK, http.Header{"Etag": []string{fmt.Sprintf("part-%d", part)}, "X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(payload)}}, ""), nil
		case request.Method == http.MethodPost && query.Get("uploadId") == "private-upload":
			_, _ = io.Copy(io.Discard, request.Body)
			return s3Response(http.StatusOK, nil, "<CompleteMultipartUploadResult><ChecksumCRC64NVME>"+testCRC64NVME(data)+"</ChecksumCRC64NVME><ChecksumType>FULL_OBJECT</ChecksumType></CompleteMultipartUploadResult>"), nil
		default:
			t.Fatalf("unexpected request %s %s", request.Method, request.URL)
			return nil, errors.New("unreachable")
		}
	})
	_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(data)), objectstorage.UploadOptions{ContentLength: length, Intent: objectstorage.UploadReplace})
	if err != nil || part != int(partCount(length, cfg.MultipartChunkBytes)) {
		t.Fatalf("Upload() = %v; parts=%d", err, part)
	}
}

func TestMultipartCleanupIsBoundedAndConservative(t *testing.T) {
	t.Parallel()
	for _, sent := range []bool{false, true} {
		t.Run(fmt.Sprintf("lost create sent=%t", sent), func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(ProviderAmazonS3)
			calls := 0
			client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
				calls++
				if sent {
					httptrace.ContextClientTrace(request.Context()).WroteHeaders()
				}
				return nil, errors.New("lost CreateMultipartUpload response")
			})
			result, err := client.Upload(t.Context(), "object", uploadSource(bytes.NewReader(make([]byte, cfg.MultipartChunkBytes+1))), objectstorage.UploadOptions{
				ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace,
			})
			wantCleanup := objectstorage.CleanupDisposition("")
			if sent {
				wantCleanup = objectstorage.CleanupPending
			}
			wantKind := objectstorage.KindInternal
			if sent {
				wantKind = objectstorage.KindOutcomeUnknown
			}
			if objectstorage.Kind(err) != wantKind || result.Cleanup != wantCleanup || calls != 1 {
				t.Fatalf("lost create = %#v, %v; calls=%d, want cleanup %q and one call", result, err, calls, wantCleanup)
			}
		})
	}

	t.Run("missing upload ID is visible pending", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig(ProviderAmazonS3)
		calls := 0
		client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
			calls++
			if request.Method != http.MethodPost || !request.URL.Query().Has("uploads") {
				t.Fatalf("unexpected request %s %s", request.Method, request.URL)
			}
			return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult></InitiateMultipartUploadResult>"), nil
		})
		result, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(bytes.Repeat([]byte("p"), int(cfg.MultipartChunkBytes)+1))), objectstorage.UploadOptions{ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace})
		if objectstorage.Kind(err) != objectstorage.KindInternal || result.Cleanup != objectstorage.CleanupPending || calls != 1 {
			t.Fatalf("malformed Create = %#v, %v; calls=%d", result, err, calls)
		}
	})

	t.Run("pagination stops at ten advancing pages", func(t *testing.T) {
		t.Parallel()
		lists := 0
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			lists++
			wantMarker := ""
			if lists > 1 {
				wantMarker = strconv.Itoa(lists - 1)
			}
			if got := request.URL.Query().Get("part-number-marker"); got != wantMarker {
				t.Fatalf("page %d marker = %q, want %q", lists, got, wantMarker)
			}
			return s3Response(http.StatusOK, nil, "<ListPartsResult><IsTruncated>true</IsTruncated><NextPartNumberMarker>"+strconv.Itoa(lists)+"</NextPartNumberMarker></ListPartsResult>"), nil
		})
		empty, complete := client.multipartPartsEmpty(t.Context(), "object", "private-upload")
		if empty || complete || lists != maximumCleanupListPages {
			t.Fatalf("pagination result = %t, %t; lists=%d, want both false and %d", empty, complete, lists, maximumCleanupListPages)
		}
	})

	cfg := validConfig(ProviderAmazonS3)
	var aborts, lists int
	var calls int
	var bodies []*trackedBody
	var returned atomic.Bool
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		if returned.Load() {
			t.Fatal("cleanup issued a request after Upload returned")
		}
		calls++
		query := request.URL.Query()
		switch {
		case request.Method == http.MethodPost && query.Has("uploads"):
			return trackedResponse(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>", &bodies), nil
		case request.Method == http.MethodPut && query.Get("partNumber") == "1":
			_, _ = io.Copy(io.Discard, request.Body)
			return trackedResponse(http.StatusOK, http.Header{"Etag": []string{"part-1"}}, "", &bodies), nil
		case request.Method == http.MethodDelete && query.Get("uploadId") == "private-upload":
			aborts++
			return trackedResponse(http.StatusNoContent, nil, "", &bodies), nil
		case request.Method == http.MethodGet && query.Get("uploadId") == "private-upload":
			lists++
			return trackedResponse(http.StatusOK, nil, "<ListPartsResult><IsTruncated>false</IsTruncated></ListPartsResult>", &bodies), nil
		default:
			t.Fatalf("unexpected cleanup request %s %s", request.Method, request.URL)
			return nil, errors.New("unreachable")
		}
	})

	result, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(bytes.Repeat([]byte("p"), int(cfg.MultipartChunkBytes)+1))), objectstorage.UploadOptions{
		ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace,
	})
	if got := objectstorage.Kind(err); got != objectstorage.KindIntegrityFailed {
		t.Fatalf("primary error = %q, want integrity_failed", got)
	}
	if result.Cleanup != objectstorage.CleanupPending || aborts != 1 || lists != 1 {
		t.Fatalf("cleanup = %#v, aborts=%d, lists=%d", result, aborts, lists)
	}
	if strings.Contains(err.Error(), "private-upload") {
		t.Fatalf("error leaked upload ID: %q", err)
	}
	for index, body := range bodies {
		if !body.closed.Load() {
			t.Fatalf("response body %d was not closed", index)
		}
	}
	returnedCalls := calls
	returned.Store(true)
	for range 8 {
		runtime.Gosched()
	}
	if calls != returnedCalls {
		t.Fatal("cleanup started work after Upload returned")
	}

	for _, scenario := range []struct {
		name       string
		provider   Provider
		abortCode  int
		listBody   string
		cancelPart bool
	}{
		{name: "abort failure", provider: ProviderAmazonS3, abortCode: http.StatusInternalServerError},
		{name: "non-empty page", provider: ProviderAmazonS3, abortCode: http.StatusNoContent, listBody: "<ListPartsResult><IsTruncated>false</IsTruncated><Part><PartNumber>1</PartNumber></Part></ListPartsResult>"},
		{name: "truncated page", provider: ProviderAmazonS3, abortCode: http.StatusNoContent, listBody: "<ListPartsResult><IsTruncated>true</IsTruncated><NextPartNumberMarker>1</NextPartNumberMarker></ListPartsResult>"},
		{name: "malformed page", provider: ProviderAmazonS3, abortCode: http.StatusNoContent, listBody: "<ListPartsResult>"},
		{name: "list failure", provider: ProviderAmazonS3, abortCode: http.StatusNoContent, listBody: "<Error><Code>InternalError</Code></Error>"},
		{name: "cancelled", provider: ProviderAmazonS3, cancelPart: true},
		{name: "R2 remains pending", provider: ProviderCloudflare, abortCode: http.StatusNoContent},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			result, aborts, lists, err := failedMultipartCleanup(t, scenario)
			if got := objectstorage.Kind(err); got != objectstorage.KindIntegrityFailed {
				t.Fatalf("primary error = %q, want integrity_failed", got)
			}
			if result.Cleanup != objectstorage.CleanupPending {
				t.Fatalf("cleanup = %q, want pending", result.Cleanup)
			}
			if aborts > maximumCleanupCycles || lists > maximumCleanupCycles*maximumCleanupListPages {
				t.Fatalf("cleanup exceeded bounded calls: aborts=%d lists=%d", aborts, lists)
			}
			switch scenario.name {
			case "non-empty page":
				if aborts != maximumCleanupCycles || lists != maximumCleanupCycles {
					t.Fatalf("non-empty cleanup = aborts:%d lists:%d, want %d each", aborts, lists, maximumCleanupCycles)
				}
			case "truncated page":
				if aborts != 1 || lists != 2 {
					t.Fatalf("non-advancing pagination = aborts:%d lists:%d, want 1 and 2", aborts, lists)
				}
			case "R2 remains pending":
				if aborts != 1 || lists != 0 {
					t.Fatalf("R2 cleanup = aborts:%d lists:%d, want 1 and 0", aborts, lists)
				}
			}
		})
	}
}

func assertMultipartCompletionFailures(t *testing.T) {
	t.Helper()
	for _, scenario := range []struct {
		name string
		body string
	}{
		{name: "corrupt checksum", body: "<CompleteMultipartUploadResult><ChecksumCRC64NVME>wrong</ChecksumCRC64NVME><ChecksumType>FULL_OBJECT</ChecksumType></CompleteMultipartUploadResult>"},
		{name: "embedded error", body: "<Error><Code>InternalError</Code><Message>private detail</Message></Error>"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig(ProviderAmazonS3)
			var aborts, lists int
			client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
				q := request.URL.Query()
				switch {
				case request.Method == http.MethodPost && q.Has("uploads"):
					return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>"), nil
				case request.Method == http.MethodPut:
					payload, _ := decodeAWSChunked(t, request.Body)
					return s3Response(http.StatusOK, http.Header{"Etag": []string{"part"}, "X-Amz-Checksum-Crc64nvme": []string{testCRC64NVME(payload)}}, ""), nil
				case request.Method == http.MethodPost && q.Get("uploadId") == "private-upload":
					_, _ = io.Copy(io.Discard, request.Body)
					return s3Response(http.StatusOK, nil, scenario.body), nil
				case request.Method == http.MethodDelete:
					aborts++
					return s3Response(http.StatusNoContent, nil, ""), nil
				case request.Method == http.MethodGet:
					lists++
					return s3Response(http.StatusOK, nil, "<ListPartsResult><IsTruncated>false</IsTruncated></ListPartsResult>"), nil
				default:
					t.Fatalf("unexpected request %s %s", request.Method, request.URL)
					return nil, errors.New("unreachable")
				}
			})
			result, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBuffer(bytes.Repeat([]byte("p"), int(cfg.MultipartChunkBytes)+1))), objectstorage.UploadOptions{ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace})
			if err == nil || result.Cleanup != objectstorage.CleanupPending || aborts != 1 || lists != 1 {
				t.Fatalf("failure cleanup = %#v, %v; aborts=%d lists=%d", result, err, aborts, lists)
			}
		})
	}
}

func failedMultipartCleanup(t *testing.T, scenario struct {
	name       string
	provider   Provider
	abortCode  int
	listBody   string
	cancelPart bool
},
) (objectstorage.UploadResult, int, int, error) {
	t.Helper()
	cfg := validConfig(scenario.provider)
	var aborts, lists int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := scriptedClientWithConfig(t, cfg, func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		switch {
		case request.Method == http.MethodPost && query.Has("uploads"):
			return s3Response(http.StatusOK, nil, "<InitiateMultipartUploadResult><UploadId>private-upload</UploadId></InitiateMultipartUploadResult>"), nil
		case request.Method == http.MethodPut && query.Get("partNumber") == "1":
			_, _ = io.Copy(io.Discard, request.Body)
			if scenario.cancelPart {
				cancel()
			}
			return s3Response(http.StatusOK, http.Header{"Etag": []string{"part-1"}}, ""), nil
		case request.Method == http.MethodDelete && query.Get("uploadId") == "private-upload":
			aborts++
			return s3Response(scenario.abortCode, nil, ""), nil
		case request.Method == http.MethodGet && query.Get("uploadId") == "private-upload":
			lists++
			return s3Response(http.StatusOK, nil, scenario.listBody), nil
		default:
			t.Fatalf("unexpected cleanup request %s %s", request.Method, request.URL)
			return nil, errors.New("unreachable")
		}
	})
	result, err := client.Upload(ctx, "object", uploadSource(bytes.NewBuffer(bytes.Repeat([]byte("p"), int(cfg.MultipartChunkBytes)+1))), objectstorage.UploadOptions{
		ContentLength: cfg.MultipartChunkBytes + 1, Intent: objectstorage.UploadReplace,
	})
	return result, aborts, lists, err
}

func scriptedClient(t *testing.T, script httpDoerFunc) *Client {
	t.Helper()
	return scriptedClientWithConfig(t, validConfig(ProviderAmazonS3), script)
}

func scriptedClientWithConfig(t *testing.T, cfg Config, script httpDoerFunc) *Client {
	t.Helper()
	client, err := newClient(cfg, testImageRootSource(t))
	if err != nil {
		t.Fatal(err)
	}
	client.transport.base = httpDoerFunc(func(request *http.Request) (*http.Response, error) {
		got := request.Header.Get("X-Amz-Expected-Bucket-Owner")
		if cfg.Provider == ProviderAmazonS3 && got != cfg.ExpectedBucketOwner {
			t.Fatalf("expected bucket owner header = %q, want %q", got, cfg.ExpectedBucketOwner)
		}
		if cfg.Provider == ProviderCloudflare && got != "" {
			t.Fatalf("R2 request included expected bucket owner %q", got)
		}
		if request.Method == http.MethodGet && request.URL.Query().Get("uploadId") != "" && request.URL.Query().Get("max-parts") != strconv.Itoa(maximumCleanupListParts) {
			t.Fatalf("ListParts max-parts = %q, want %d", request.URL.Query().Get("max-parts"), maximumCleanupListParts)
		}
		return script(request)
	})
	return client
}

func s3Response(status int, header http.Header, body string) *http.Response {
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{StatusCode: status, Status: strconv.Itoa(status), Header: header, Body: io.NopCloser(strings.NewReader(body))}
}

type trackedBody struct {
	io.Reader

	closed atomic.Bool
}

func (b *trackedBody) Close() error {
	b.closed.Store(true)
	return nil
}

func trackedResponse(status int, header http.Header, body string, bodies *[]*trackedBody) *http.Response {
	tracked := &trackedBody{Reader: strings.NewReader(body)}
	*bodies = append(*bodies, tracked)
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{StatusCode: status, Status: strconv.Itoa(status), Header: header, Body: tracked}
}

func decodeAWSChunked(t *testing.T, body io.ReadCloser) ([]byte, http.Header) {
	t.Helper()
	defer func() {
		if err := body.Close(); err != nil {
			t.Errorf("close AWS chunked body: %v", err)
		}
	}()
	reader := bufio.NewReader(body)
	var payload bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		sizeText, _, _ := strings.Cut(strings.TrimSpace(line), ";")
		size, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		if size == 0 {
			trailers := make(http.Header)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				line = strings.TrimRight(line, "\r\n")
				if line == "" {
					return payload.Bytes(), trailers
				}
				name, value, ok := strings.Cut(line, ":")
				if !ok {
					t.Fatalf("invalid trailer %q", line)
				}
				trailers.Add(name, strings.TrimSpace(value))
			}
		}
		chunk := make([]byte, size)
		if _, err := io.ReadFull(reader, chunk); err != nil {
			t.Fatal(err)
		}
		payload.Write(chunk)
		if line, err := reader.ReadString('\n'); err != nil || line != "\r\n" {
			t.Fatalf("chunk terminator = %q, %v", line, err)
		}
	}
}

func testCRC64NVME(data []byte) string {
	sum := crc64.Checksum(data, crc64.MakeTable(0x9a6c9329ac4bc9b5))
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], sum)
	return base64.StdEncoding.EncodeToString(raw[:])
}
