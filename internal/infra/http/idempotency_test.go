package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/idempotency"
	"github.com/example/go-service-template-rest/internal/problem"
)

// testIdempotencyOutcomeTimeout bounds recording an outcome in tests. It is short
// on purpose: a test that hangs here would hang for the whole default budget.
const testIdempotencyOutcomeTimeout = 500 * time.Millisecond

// fakeIdempotencyStore is the single-process stand-in for the storage a service
// supplies. It is deliberately not offered as production code: a map is correct
// for exactly one replica, which is the version this seam exists to prevent
// teams from shipping.
type fakeIdempotencyStore struct {
	mu        sync.Mutex
	reserved  map[string]string
	completed map[string]idempotency.StoredResponse
	failWith  error
	releases  int
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{
		reserved:  map[string]string{},
		completed: map[string]idempotency.StoredResponse{},
	}
}

func (s *fakeIdempotencyStore) Reserve(_ context.Context, key, fingerprint string) (*idempotency.StoredResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	if existing, held := s.reserved[key]; held {
		if existing != fingerprint {
			return nil, idempotency.ErrKeyReused
		}
		if stored, done := s.completed[key]; done {
			return &stored, nil
		}
		return nil, idempotency.ErrInFlight
	}
	s.reserved[key] = fingerprint
	return nil, nil //nolint:nilnil // No response and no error is how this contract says the caller owns the key.
}

func (s *fakeIdempotencyStore) Complete(_ context.Context, key string, response idempotency.StoredResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed[key] = response
	return nil
}

func (s *fakeIdempotencyStore) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releases++
	delete(s.reserved, key)
	return nil
}

func (s *fakeIdempotencyStore) releaseCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.releases
}

// TestIdempotentReplaysTheFirstResult is the whole point: a client that retried
// because it never saw the answer gets the answer, not a second resource and not
// a conflict with its own earlier success.
func TestIdempotentReplaysTheFirstResult(t *testing.T) {
	t.Parallel()

	var runs int
	handler := idempotentTestHandler(t, newFakeIdempotencyStore(), func(w http.ResponseWriter, _ *http.Request) {
		runs++
		w.Header().Set("Location", "/api/v1/articles/created")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"slug":"created"}`))
	})

	first := postWithKey(handler, "key-1", `{"slug":"created"}`)
	second := postWithKey(handler, "key-1", `{"slug":"created"}`)

	if runs != 1 {
		t.Fatalf("handler ran %d times, want 1", runs)
	}
	for name, response := range map[string]*httptest.ResponseRecorder{"first": first, "second": second} {
		if response.Code != http.StatusCreated {
			t.Fatalf("%s status = %d, want %d", name, response.Code, http.StatusCreated)
		}
		if got := response.Header().Get("Location"); got != "/api/v1/articles/created" {
			t.Fatalf("%s Location = %q, want the created resource", name, got)
		}
		if body := strings.TrimSpace(response.Body.String()); body != `{"slug":"created"}` {
			t.Fatalf("%s body = %q, want the first attempt's body", name, body)
		}
	}
	if got := second.Header().Get("Idempotent-Replay"); got != "true" {
		t.Fatalf("replay header = %q, want true", got)
	}
	if got := first.Header().Get("Idempotent-Replay"); got != "" {
		t.Fatalf("first attempt carried a replay header = %q", got)
	}
}

func TestIdempotentRefusesAKeyReusedForDifferentIntent(t *testing.T) {
	t.Parallel()

	handler := idempotentTestHandler(t, newFakeIdempotencyStore(), createdHandler)

	postWithKey(handler, "key-1", `{"slug":"first"}`)
	response := postWithKey(handler, "key-1", `{"slug":"different"}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d for a key reused on a different request", response.Code, http.StatusConflict)
	}
	assertProblemCode(t, response, problem.CodeConflict)
}

func TestIdempotentReportsAnInFlightAttemptAsRetryable(t *testing.T) {
	t.Parallel()

	store := newFakeIdempotencyStore()
	handler := idempotentTestHandler(t, store, func(w http.ResponseWriter, _ *http.Request) {
		// Reserved but never completed, which is what a concurrent attempt looks
		// like to the second caller.
		w.WriteHeader(http.StatusCreated)
	})
	if _, err := store.Reserve(context.Background(), "key-1", fingerprintFor(`{"slug":"first"}`)); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	response := postWithKey(handler, "key-1", `{"slug":"first"}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if got := response.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want a retry hint: the first attempt is running, not failed", got)
	}
}

// TestIdempotentDoesNotHoldAServerFault keeps one bad minute from becoming a key
// that answers 500 until it expires. A 5xx is the outcome a client is entitled to
// retry.
func TestIdempotentDoesNotHoldAServerFault(t *testing.T) {
	t.Parallel()

	store := newFakeIdempotencyStore()
	var runs int
	handler := idempotentTestHandler(t, store, func(w http.ResponseWriter, _ *http.Request) {
		runs++
		if runs == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	if first := postWithKey(handler, "key-1", `{"slug":"first"}`); first.Code != http.StatusInternalServerError {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusInternalServerError)
	}
	second := postWithKey(handler, "key-1", `{"slug":"first"}`)

	if second.Code != http.StatusCreated {
		t.Fatalf("second status = %d, want the retry to run", second.Code)
	}
	if runs != 2 {
		t.Fatalf("handler ran %d times, want the retry admitted after a server fault", runs)
	}
	if store.releaseCount() == 0 {
		t.Fatal("reservation was not released after a server fault")
	}
}

// TestIdempotentReleasesTheKeyOnPanic keeps a crash from spending a key on work
// that never happened. Recover sits outside this middleware, so the panic unwinds
// through it first.
func TestIdempotentReleasesTheKeyOnPanic(t *testing.T) {
	t.Parallel()

	store := newFakeIdempotencyStore()
	handler := Recover(slog.New(slog.DiscardHandler), Idempotent(store, testIdempotencyOutcomeTimeout, slog.New(slog.DiscardHandler),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("handler exploded")
		})))

	response := postWithKey(handler, "key-1", `{"slug":"first"}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if store.releaseCount() != 1 {
		t.Fatalf("release count = %d, want the key freed for a retry", store.releaseCount())
	}
}

func TestIdempotentIgnoresRequestsItDoesNotOwn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		method string
		key    string
	}{
		{name: "no key", method: http.MethodPost},
		{name: "safe method", method: http.MethodGet, key: "key-1"},
		{name: "safe method without key", method: http.MethodGet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var runs int
			handler := idempotentTestHandler(t, newFakeIdempotencyStore(), func(w http.ResponseWriter, _ *http.Request) {
				runs++
				w.WriteHeader(http.StatusOK)
			})

			for range 2 {
				request := httptest.NewRequest(tc.method, "/api/v1/articles", strings.NewReader(""))
				if tc.key != "" {
					request.Header.Set(IdempotencyKeyHeader, tc.key)
				}
				handler.ServeHTTP(httptest.NewRecorder(), request)
			}

			if runs != 2 {
				t.Fatalf("handler ran %d times, want both requests passed through untouched", runs)
			}
		})
	}
}

func TestIdempotentRejectsAMalformedKey(t *testing.T) {
	t.Parallel()

	handler := idempotentTestHandler(t, newFakeIdempotencyStore(), createdHandler)

	response := postWithKey(handler, "key with spaces/and-slashes", `{"slug":"first"}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertProblemCode(t, response, problem.CodeBadRequest)
}

// TestIdempotentSurfacesAStoreFailureAsInternal keeps a broken store from
// silently letting duplicate work through.
func TestIdempotentSurfacesAStoreFailureAsInternal(t *testing.T) {
	t.Parallel()

	store := newFakeIdempotencyStore()
	store.failWith = errors.New("store unavailable")
	var runs int
	handler := idempotentTestHandler(t, store, func(w http.ResponseWriter, _ *http.Request) {
		runs++
		w.WriteHeader(http.StatusCreated)
	})

	response := postWithKey(handler, "key-1", `{"slug":"first"}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if runs != 0 {
		t.Fatal("handler ran while the idempotency store was unavailable")
	}
}

// TestIdempotentPreservesTheBodyForTheHandler keeps fingerprinting from consuming
// what the operation is supposed to decode.
func TestIdempotentPreservesTheBodyForTheHandler(t *testing.T) {
	t.Parallel()

	var seen string
	handler := idempotentTestHandler(t, newFakeIdempotencyStore(), func(w http.ResponseWriter, r *http.Request) {
		body, err := readAllBody(r)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		seen = body
		w.WriteHeader(http.StatusCreated)
	})

	postWithKey(handler, "key-1", `{"slug":"kept"}`)

	if seen != `{"slug":"kept"}` {
		t.Fatalf("handler observed body %q, want the original request body", seen)
	}
}

// TestIdempotentNilStoreLeavesTheChainUnchanged keeps the seam free for services
// that do not need it.
func TestIdempotentNilStoreLeavesTheChainUnchanged(t *testing.T) {
	t.Parallel()

	terminal := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	if got := Idempotent(nil, testIdempotencyOutcomeTimeout, nil, terminal); fmt.Sprintf("%p", got) != fmt.Sprintf("%p", terminal) {
		t.Fatal("Idempotent(nil, ...) wrapped the handler; a service without a store must pay nothing")
	}
}

func idempotentTestHandler(tb testing.TB, store IdempotencyStore, handler http.HandlerFunc) http.Handler {
	tb.Helper()
	return Idempotent(store, testIdempotencyOutcomeTimeout, slog.New(slog.DiscardHandler), handler)
}

func createdHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func postWithKey(handler http.Handler, key, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	request.Header.Set(IdempotencyKeyHeader, key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func fingerprintFor(body string) string {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	fingerprint, err := requestFingerprint(request)
	if err != nil {
		panic(err)
	}
	return fingerprint
}

func readAllBody(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err //nolint:wrapcheck // Test helper reporting the read failure verbatim.
	}
	return string(body), nil
}

// blockingIdempotencyStore never returns from Complete. It stands in for the
// failure the outcome budget exists for: a row lock held by a long transaction,
// or a connection wedged mid-retransmit.
type blockingIdempotencyStore struct {
	released chan struct{}
	observed chan context.Context
}

func newBlockingIdempotencyStore() *blockingIdempotencyStore {
	return &blockingIdempotencyStore{
		released: make(chan struct{}),
		observed: make(chan context.Context, 1),
	}
}

//nolint:nilnil // A nil response with a nil error is the contract: this attempt owns the key.
func (s *blockingIdempotencyStore) Reserve(context.Context, string, string) (*idempotency.StoredResponse, error) {
	return nil, nil
}

func (s *blockingIdempotencyStore) Complete(ctx context.Context, _ string, _ idempotency.StoredResponse) error {
	select {
	case s.observed <- ctx:
	default:
	}
	select {
	case <-ctx.Done():
		//nolint:wrapcheck // The test asserts on the context error identity.
		return ctx.Err()
	case <-s.released:
		return nil
	}
}

func (s *blockingIdempotencyStore) Release(context.Context, string) error { return nil }

// TestIdempotentBoundsOutcomeRecording is the regression anchor for the defect
// that made this budget necessary. The outcome context is detached from the
// request so a spent budget cannot stop the record from being written — and
// context.WithoutCancel returns a context with no deadline at all, so before this
// bound existed a stalled store held the handler goroutine and its in-flight slot
// for the life of the process, after the client had already been answered.
func TestIdempotentBoundsOutcomeRecording(t *testing.T) {
	t.Parallel()

	store := newBlockingIdempotencyStore()
	t.Cleanup(func() { close(store.released) })
	handler := idempotentTestHandler(t, store, createdHandler)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = postWithKey(handler, "outcome-budget", `{"slug":"a"}`)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not return: recording the outcome is unbounded")
	}

	observed := <-store.observed
	if _, hasDeadline := observed.Deadline(); !hasDeadline {
		t.Fatal("outcome context carries no deadline; a stalled store would hold the handler goroutine forever")
	}
}

// TestIdempotentRecorderPreservesOptionalWriterInterfaces pins the property the
// hand-rolled recorder lost. Embedding http.ResponseWriter in a struct promotes
// only its three methods, so enabling idempotency handed every handler beneath it
// a writer that could not flush — which broke streamed responses on a deploy that
// changed no handler.
func TestIdempotentRecorderPreservesOptionalWriterInterfaces(t *testing.T) {
	t.Parallel()

	var (
		sawFlusher bool
		flushErr   error
	)
	handler := idempotentTestHandler(t, newFakeIdempotencyStore(), func(w http.ResponseWriter, _ *http.Request) {
		_, sawFlusher = w.(http.Flusher)
		w.WriteHeader(http.StatusCreated)
		flushErr = http.NewResponseController(w).Flush()
	})

	postWithKey(handler, "streaming-key", `{"slug":"a"}`)

	if !sawFlusher {
		t.Fatal("recorder is not an http.Flusher; a streaming handler beneath it cannot flush")
	}
	if flushErr != nil {
		t.Fatalf("http.ResponseController.Flush() error = %v, want nil", flushErr)
	}
}

// TestIdempotentRecorderCapturesReadFromBodies keeps the replay honest for a
// handler that hands the writer a reader. Those bytes bypass Write, so without the
// ReadFrom hook the store held an empty body and still reported it replayable —
// and the retry was answered 200 with nothing in it.
func TestIdempotentRecorderCapturesReadFromBodies(t *testing.T) {
	t.Parallel()

	const body = `{"slug":"copied"}`
	store := newFakeIdempotencyStore()
	handler := idempotentTestHandler(t, store, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := io.Copy(w, strings.NewReader(body)); err != nil {
			t.Errorf("copy response body: %v", err)
		}
	})

	first := postWithKey(handler, "readfrom-key", `{"slug":"a"}`)
	if first.Body.String() != body {
		t.Fatalf("first response body = %q, want %q", first.Body.String(), body)
	}

	replay := postWithKey(handler, "readfrom-key", `{"slug":"a"}`)
	if replay.Header().Get("Idempotent-Replay") != "true" {
		t.Fatalf("replay is not labeled: headers = %v", replay.Header())
	}
	if replay.Body.String() != body {
		t.Fatalf("replayed body = %q, want %q", replay.Body.String(), body)
	}
}
