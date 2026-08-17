package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestDownloadCompletesOnlyAtValidatedEOF(t *testing.T) {
	const data = "download"

	validHeader := func(checksum string) http.Header {
		return http.Header{
			"Content-Length":           []string{strconv.Itoa(len(data))},
			"X-Amz-Checksum-Crc64nvme": []string{checksum},
			"X-Amz-Checksum-Type":      []string{"FULL_OBJECT"},
		}
	}
	response := func(header http.Header, body io.ReadCloser) *http.Response {
		length := int64(-1)
		if value := header.Get("Content-Length"); value != "" {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			length = parsed
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200", Header: header, ContentLength: length, Body: body}
	}

	t.Run("range response is refused before body exposure", func(t *testing.T) {
		body := &countedReadCloser{reader: strings.NewReader(data)}
		header := validHeader(testCRC64NVME([]byte(data)))
		header.Set("Content-Range", "bytes 0-7/20")
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			response := response(header, body)
			response.StatusCode = http.StatusPartialContent
			return response, nil
		})
		result, err := client.Download(t.Context(), "object")
		if objectstorage.Kind(err) != objectstorage.KindIntegrityFailed || result.Body != nil {
			t.Fatalf("Download(range) = %#v, %v", result, err)
		}
		if got := body.closes.Load(); got != 1 {
			t.Fatalf("range body closes = %d, want 1", got)
		}
	})

	t.Run("only valid metadata exposes a body", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			header http.Header
			kind   objectstorage.ErrorKind
		}{
			{name: "oversized", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Set("Content-Length", strconv.FormatInt(validConfig(ProviderAmazonS3).MaxObjectBytes+1, 10))
				return h
			}(), kind: objectstorage.KindTooLarge},
			{name: "missing content length", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Del("Content-Length")
				return h
			}(), kind: objectstorage.KindIntegrityFailed},
			{name: "missing checksum", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Del("X-Amz-Checksum-Crc64nvme")
				return h
			}(), kind: objectstorage.KindIntegrityFailed},
			{name: "empty checksum", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Set("X-Amz-Checksum-Crc64nvme", "")
				return h
			}(), kind: objectstorage.KindIntegrityFailed},
			{name: "malformed checksum", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Set("X-Amz-Checksum-Crc64nvme", "not-base64")
				return h
			}(), kind: objectstorage.KindIntegrityFailed},
			{name: "composite checksum", header: func() http.Header {
				h := validHeader(testCRC64NVME([]byte(data)))
				h.Set("X-Amz-Checksum-Type", "COMPOSITE")
				return h
			}(), kind: objectstorage.KindIntegrityFailed},
		} {
			t.Run(test.name, func(t *testing.T) {
				body := &countedReadCloser{reader: strings.NewReader(data)}
				client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
					return response(test.header, body), nil
				})
				result, err := client.Download(t.Context(), "object")
				if result.Body != nil || objectstorage.Kind(err) != test.kind {
					t.Fatalf("Download() = %#v, %v; want no body and %q", result, err, test.kind)
				}
				if got := body.closes.Load(); got != 1 {
					t.Fatalf("underlying closes = %d, want 1", got)
				}
			})
		}
	})

	t.Run("validated EOF is the only success", func(t *testing.T) {
		body := &countedReadCloser{reader: strings.NewReader(data)}
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			if got := request.Header.Get("X-Amz-Checksum-Mode"); got != "ENABLED" {
				t.Fatalf("checksum mode = %q, want ENABLED", got)
			}
			return response(validHeader(testCRC64NVME([]byte(data))), body), nil
		})
		result, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(result.Body)
		if err != nil || string(got) != data {
			t.Fatalf("ReadAll() = %q, %v", got, err)
		}
		if got := body.closes.Load(); got != 1 {
			t.Fatalf("underlying closes at validated EOF = %d, want 1", got)
		}
		if err := result.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if got := body.closes.Load(); got != 1 {
			t.Fatalf("underlying closes = %d, want 1", got)
		}
	})

	t.Run("mismatch and terminal errors are stable", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			header http.Header
			reader io.Reader
			kind   objectstorage.ErrorKind
		}{
			{name: "mismatch", header: validHeader(testCRC64NVME([]byte("mismatch"))), reader: strings.NewReader(data), kind: objectstorage.KindIntegrityFailed},
			{name: "terminal mismatch", header: validHeader(testCRC64NVME([]byte(data))), reader: errReader{errors.New("provider secret")}, kind: objectstorage.KindIntegrityFailed},
		} {
			t.Run(test.name, func(t *testing.T) {
				body := &countedReadCloser{reader: test.reader}
				client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
					return response(test.header, body), nil
				})
				result, err := client.Download(t.Context(), "object")
				if err != nil {
					t.Fatal(err)
				}
				_, err = io.ReadAll(result.Body)
				if objectstorage.Kind(err) != test.kind {
					t.Fatalf("ReadAll() error = %v, kind = %q, want %q", err, objectstorage.Kind(err), test.kind)
				}
				if got := body.closes.Load(); got != 1 {
					t.Fatalf("underlying closes = %d, want 1", got)
				}
			})
		}
	})

	t.Run("declared maximum still bounds a longer body", func(t *testing.T) {
		cfg := validConfig(ProviderAmazonS3)
		cfg.MaxObjectBytes = cfg.MultipartChunkBytes
		cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
		for _, test := range []struct {
			name string
			body string
			kind objectstorage.ErrorKind
		}{
			{name: "exact limit", body: strings.Repeat("d", int(cfg.MaxObjectBytes))},
			{name: "overflow", body: strings.Repeat("d", int(cfg.MaxObjectBytes)) + "!", kind: objectstorage.KindTooLarge},
		} {
			t.Run(test.name, func(t *testing.T) {
				body := &countedReadCloser{reader: strings.NewReader(test.body)}
				header := validHeader(testCRC64NVME([]byte(test.body)))
				header.Set("Content-Length", strconv.FormatInt(cfg.MaxObjectBytes, 10))
				client := scriptedClientWithConfig(t, cfg, func(*http.Request) (*http.Response, error) {
					return response(header, body), nil
				})
				result, err := client.Download(t.Context(), "object")
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(result.Body)
				want := test.body
				if int64(len(want)) > cfg.MaxObjectBytes {
					want = want[:cfg.MaxObjectBytes]
				}
				if string(got) != want || test.kind == "" && err != nil || test.kind != "" && objectstorage.Kind(err) != test.kind {
					t.Fatalf("ReadAll() length = %d, %v; want bounded body and %q", len(got), err, test.kind)
				}
				if got := body.closes.Load(); got != 1 {
					t.Fatalf("underlying closes = %d, want 1", got)
				}
			})
		}
	})

	t.Run("close and cancellation release admission once", func(t *testing.T) {
		var calls atomic.Int32
		first := &countedReadCloser{reader: strings.NewReader(data)}
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return response(validHeader(testCRC64NVME([]byte(data))), first), nil
		})
		result, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatal(err)
		}
		if err := result.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if err := result.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if got := first.closes.Load(); got != 1 {
			t.Fatalf("early-close count = %d, want 1", got)
		}
		second, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatalf("second Download() error = %v", err)
		}
		_ = second.Body.Close()
		if got := calls.Load(); got != 2 {
			t.Fatalf("GET calls = %d, want 2", got)
		}

		ctx, cancel := context.WithCancel(t.Context())
		body := newBlockedReadCloser()
		cancelClient := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			return response(validHeader(testCRC64NVME([]byte(data))), body), nil
		})
		cancelled, err := cancelClient.Download(ctx, "object")
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		select {
		case <-body.closed:
		case <-time.After(time.Second):
			t.Fatal("cancelled download did not close its idle body")
		}
		_, err = cancelled.Body.Read(make([]byte, 1))
		if objectstorage.Kind(err) != objectstorage.KindCancelled || body.closes.Load() != 1 {
			t.Fatalf("cancelled read = %v; closes = %d", err, body.closes.Load())
		}
		next, err := cancelClient.Download(t.Context(), "object")
		if err != nil {
			t.Fatalf("Download() after context cancellation = %v, want admission release", err)
		}
		_ = next.Body.Close()
	})

	t.Run("close releases a blocked read", func(t *testing.T) {
		cfg := validConfig(ProviderAmazonS3)
		cfg.MaxActiveOperations = 1
		cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
		body := newBlockedReadCloser()
		client := scriptedClientWithConfig(t, cfg, func(*http.Request) (*http.Response, error) {
			return response(validHeader(testCRC64NVME([]byte(data))), body), nil
		})
		result, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatal(err)
		}
		readDone := make(chan error, 1)
		go func() {
			_, err := result.Body.Read(make([]byte, 1))
			readDone <- err
		}()
		<-body.entered
		if err := result.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-readDone; objectstorage.Kind(err) != objectstorage.KindInternal {
			t.Fatalf("blocked Read() error = %v, want internal after incomplete Close", err)
		}
		if got := body.closes.Load(); got != 1 {
			t.Fatalf("underlying closes = %d, want 1", got)
		}
		next, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatalf("Download() after Close = %v, want admission release", err)
		}
		_ = next.Body.Close()
	})

	t.Run("cancellation releases a blocked read", func(t *testing.T) {
		cfg := validConfig(ProviderAmazonS3)
		cfg.MaxActiveOperations = 1
		cfg.MaxWorkingMemoryBytes, _ = cfg.requiredMemory()
		body := newBlockedReadCloser()
		client := scriptedClientWithConfig(t, cfg, func(*http.Request) (*http.Response, error) {
			return response(validHeader(testCRC64NVME([]byte(data))), body), nil
		})
		ctx, cancel := context.WithCancel(t.Context())
		result, err := client.Download(ctx, "object")
		if err != nil {
			t.Fatal(err)
		}
		readDone := make(chan error, 1)
		go func() {
			_, err := result.Body.Read(make([]byte, 1))
			readDone <- err
		}()
		<-body.entered
		cancel()
		if err := <-readDone; objectstorage.Kind(err) != objectstorage.KindCancelled {
			t.Fatalf("blocked cancelled Read() error = %v, want cancelled", err)
		}
		if got := body.closes.Load(); got != 1 {
			t.Fatalf("underlying closes = %d, want 1", got)
		}
		if got := len(client.tokens); got != 0 {
			t.Fatalf("admission tokens after blocked download cancellation = %d, want 0", got)
		}
	})

	t.Run("expired deadline starts no GET", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()
		calls := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("unexpected GET")
		})
		_, err := client.Download(ctx, "object")
		if objectstorage.Kind(err) != objectstorage.KindDeadlineExceeded || calls != 0 {
			t.Fatalf("Download() = %v; calls = %d", err, calls)
		}
	})
}

func TestDownloadAcquisitionRetryIsBounded(t *testing.T) {
	const data = "download"
	validResponse := func() *http.Response {
		response := s3Response(http.StatusOK, http.Header{
			"Content-Length":           {strconv.Itoa(len(data))},
			"X-Amz-Checksum-Crc64nvme": {testCRC64NVME([]byte(data))},
			"X-Amz-Checksum-Type":      {"FULL_OBJECT"},
		}, data)
		response.ContentLength = int64(len(data))
		return response
	}

	t.Run("transient acquisition succeeds on third attempt", func(t *testing.T) {
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return s3Response(http.StatusServiceUnavailable, nil, `<Error><Code>SlowDown</Code></Error>`), nil
			}
			return validResponse(), nil
		})
		result, err := client.Download(t.Context(), "object")
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(result.Body)
		if err != nil || string(got) != data || attempts != 3 {
			t.Fatalf("Download() = %q, %v, attempts=%d; want %q, nil, 3", got, err, attempts, data)
		}
	})

	t.Run("transient acquisition exhausts at three attempts", func(t *testing.T) {
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			return s3Response(http.StatusServiceUnavailable, nil, `<Error><Code>SlowDown</Code></Error>`), nil
		})
		result, err := client.Download(t.Context(), "object")
		if objectstorage.Kind(err) != objectstorage.KindTemporary || result.Body != nil || attempts != 3 {
			t.Fatalf("Download() = %#v, %v, attempts=%d; want temporary, no body, 3", result, err, attempts)
		}
	})

	t.Run("cancellation prevents another acquisition attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			cancel()
			return s3Response(http.StatusServiceUnavailable, nil, `<Error><Code>SlowDown</Code></Error>`), nil
		})
		result, err := client.Download(ctx, "object")
		if objectstorage.Kind(err) != objectstorage.KindCancelled || result.Body != nil || attempts != 1 {
			t.Fatalf("Download() = %#v, %v, attempts=%d; want canceled, no body, 1", result, err, attempts)
		}
	})
}

type countedReadCloser struct {
	reader io.Reader
	closes atomic.Int32
}

func (b *countedReadCloser) Read(p []byte) (int, error) {
	return b.reader.Read(p) //nolint:wrapcheck // Reader callers require the source EOF identity.
}

func (b *countedReadCloser) Close() error {
	b.closes.Add(1)
	return nil
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

type blockedReadCloser struct {
	entered chan struct{}
	closed  chan struct{}
	closes  atomic.Int32
}

func newBlockedReadCloser() *blockedReadCloser {
	return &blockedReadCloser{entered: make(chan struct{}), closed: make(chan struct{})}
}

func (b *blockedReadCloser) Read([]byte) (int, error) {
	select {
	case <-b.entered:
	default:
		close(b.entered)
	}
	<-b.closed
	return 0, io.EOF
}

func (b *blockedReadCloser) Close() error {
	if b.closes.Add(1) == 1 {
		close(b.closed)
	}
	return nil
}
