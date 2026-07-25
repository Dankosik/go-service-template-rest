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
)

// fakeIdempotencyStore is the single-process stand-in for the storage a service
// supplies. It is deliberately not offered as production code: a map is correct
// for exactly one replica, which is the version this seam exists to prevent
// teams from shipping.
type fakeIdempotencyStore struct {
	mu        sync.Mutex
	reserved  map[string]string
	completed map[string]StoredResponse
	failWith  error
	releases  int
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{
		reserved:  map[string]string{},
		completed: map[string]StoredResponse{},
	}
}

func (s *fakeIdempotencyStore) Reserve(_ context.Context, key, fingerprint string) (*StoredResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	if existing, held := s.reserved[key]; held {
		if existing != fingerprint {
			return nil, ErrIdempotencyKeyReused
		}
		if stored, done := s.completed[key]; done {
			return &stored, nil
		}
		return nil, ErrIdempotencyInFlight
	}
	s.reserved[key] = fingerprint
	return nil, nil //nolint:nilnil // No response and no error is how this contract says the caller owns the key.
}

func (s *fakeIdempotencyStore) Complete(_ context.Context, key string, response StoredResponse) error {
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
	assertProblemCode(t, response, problemCodeConflict)
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
	handler := Recover(slog.New(slog.DiscardHandler), Idempotent(store, slog.New(slog.DiscardHandler),
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
	assertProblemCode(t, response, problemCodeBadRequest)
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

	if got := Idempotent(nil, nil, terminal); fmt.Sprintf("%p", got) != fmt.Sprintf("%p", terminal) {
		t.Fatal("Idempotent(nil, ...) wrapped the handler; a service without a store must pay nothing")
	}
}

func idempotentTestHandler(tb testing.TB, store IdempotencyStore, handler http.HandlerFunc) http.Handler {
	tb.Helper()
	return Idempotent(store, slog.New(slog.DiscardHandler), handler)
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
