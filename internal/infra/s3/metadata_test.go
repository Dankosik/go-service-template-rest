package s3

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestMetadataAndDeleteExposePortableResults(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	modified := time.Date(2026, 8, 12, 10, 11, 12, 0, time.FixedZone("offset", 3*60*60))
	client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		switch request.Method {
		case http.MethodHead:
			return s3Response(http.StatusOK, http.Header{
				"Content-Length": []string{"7"},
				"Content-Type":   []string{"text/plain"},
				"Last-Modified":  []string{modified.UTC().Format(http.TimeFormat)},
			}, ""), nil
		case http.MethodDelete:
			return s3Response(http.StatusNoContent, nil, ""), nil
		default:
			t.Fatalf("method = %s", request.Method)
			return nil, http.ErrAbortHandler
		}
	})

	metadata, err := client.Metadata(context.Background(), "object")
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if metadata.Size != 7 || metadata.ContentType != "text/plain" || !metadata.LastModified.Equal(modified.UTC()) || metadata.LastModified.Location() != time.UTC {
		t.Fatalf("Metadata() = %#v, want portable UTC fields", metadata)
	}
	if err := client.Delete(context.Background(), "missing-object"); err != nil {
		t.Fatalf("Delete(absent) error = %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests = %d, want 2", requests.Load())
	}

	for _, test := range []struct {
		name string
		code int
		body string
		want objectstorage.ErrorKind
	}{
		{name: "admitted absence", code: http.StatusNotFound, body: `<Error><Code>NoSuchKey</Code></Error>`, want: objectstorage.KindNotFound},
		{name: "concealed absence", code: http.StatusForbidden, body: `<Error><Code>AccessDenied</Code></Error>`, want: objectstorage.KindDenied},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
				return s3Response(test.code, nil, test.body), nil
			})
			_, err := client.Metadata(context.Background(), "object")
			if got := objectstorage.Kind(err); got != test.want {
				t.Fatalf("Kind(Metadata()) = %q, want %q", got, test.want)
			}
		})
	}
}
