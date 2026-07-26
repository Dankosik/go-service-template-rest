//go:build integration

package integration_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/idempotency"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
)

const (
	idempotencyTestRetention = time.Hour
	idempotencyTestAttempts  = 32
)

// newIdempotencyPool migrates a fresh database with this repository's own
// migrations and returns a pool on it.
//
// The migrations directory is the real one rather than a fixture, so this also
// proves the shipped schema is what the store's statements are written against: a
// column renamed in one and not the other is a runtime failure no unit test sees.
func newIdempotencyPool(t *testing.T) *postgres.Pool {
	t.Helper()

	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       idempotencyTestAttempts,
		AcquireTimeout:     5 * time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newIdempotencyStore(t *testing.T, retention time.Duration) *postgres.IdempotencyStore {
	t.Helper()

	store, err := postgres.NewIdempotencyStore(newIdempotencyPool(t), retention)
	if err != nil {
		t.Fatalf("create idempotency store: %v", err)
	}
	return store
}

// TestIdempotencyStoreClaimIsAtomic is the property this store exists for, and the
// one no fake can prove: of many concurrent attempts on one key, exactly one may
// own it and every other must be told it is in flight.
//
// The hand-rolled version a service writes instead — SELECT, then INSERT if absent
// — passes a single-threaded test and fails here: both attempts see no row, both
// insert, one dies on the primary key, and the retry it triggers does the work
// twice.
func TestIdempotencyStoreClaimIsAtomic(t *testing.T) {
	store := newIdempotencyStore(t, idempotencyTestRetention)

	const (
		key         = "concurrent-claim"
		fingerprint = "fingerprint"
	)

	var (
		mu      sync.Mutex
		owners  int
		flights int
		other   []error
		wg      sync.WaitGroup
	)
	for range idempotencyTestAttempts {
		wg.Go(func() {
			stored, err := store.Reserve(t.Context(), key, fingerprint)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && stored == nil:
				owners++
			case errors.Is(err, idempotency.ErrInFlight):
				flights++
			default:
				other = append(other, err)
			}
		})
	}
	wg.Wait()

	if len(other) > 0 {
		t.Fatalf("unexpected Reserve outcomes: %v", other)
	}
	if owners != 1 {
		t.Fatalf("owners = %d, want exactly 1 of %d concurrent attempts", owners, idempotencyTestAttempts)
	}
	if flights != idempotencyTestAttempts-1 {
		t.Fatalf("in-flight results = %d, want %d", flights, idempotencyTestAttempts-1)
	}
}

// TestIdempotencyStoreReplaysCompletedOutcome walks the lifecycle against a real
// database, so the JSONB header round trip and the completed-versus-in-flight
// distinction are proved by the statements rather than by a fake.
func TestIdempotencyStoreReplaysCompletedOutcome(t *testing.T) {
	store := newIdempotencyStore(t, idempotencyTestRetention)

	const (
		key         = "replayed-key"
		fingerprint = "fingerprint"
	)

	if _, err := store.Reserve(t.Context(), key, fingerprint); err != nil {
		t.Fatalf("Reserve(first) error = %v", err)
	}

	want := idempotency.StoredResponse{
		Status:     http.StatusCreated,
		Header:     http.Header{"Location": []string{"/api/v1/articles/a"}},
		Body:       []byte(`{"slug":"a"}`),
		Replayable: true,
	}
	if err := store.Complete(t.Context(), key, want); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	stored, err := store.Reserve(t.Context(), key, fingerprint)
	if err != nil {
		t.Fatalf("Reserve(replay) error = %v", err)
	}
	if stored == nil {
		t.Fatal("Reserve(replay) = nil, want the completed outcome")
	}
	if stored.Status != want.Status {
		t.Fatalf("replayed status = %d, want %d", stored.Status, want.Status)
	}
	if got := stored.Header.Get("Location"); got != "/api/v1/articles/a" {
		t.Fatalf("replayed Location = %q, want %q", got, "/api/v1/articles/a")
	}
	if string(stored.Body) != string(want.Body) {
		t.Fatalf("replayed body = %q, want %q", stored.Body, want.Body)
	}

	// A different request under the same key is the one case replay would be
	// dangerous, and it must be refused rather than answered.
	if _, err := store.Reserve(t.Context(), key, "different-fingerprint"); !errors.Is(err, idempotency.ErrKeyReused) {
		t.Fatalf("Reserve(reused key) error = %v, want %v", err, idempotency.ErrKeyReused)
	}
}

// TestIdempotencyStoreReleaseFreesIncompleteReservationOnly keeps a late release
// from erasing an outcome another attempt is entitled to replay.
func TestIdempotencyStoreReleaseFreesIncompleteReservationOnly(t *testing.T) {
	store := newIdempotencyStore(t, idempotencyTestRetention)

	const fingerprint = "fingerprint"

	if _, err := store.Reserve(t.Context(), "abandoned", fingerprint); err != nil {
		t.Fatalf("Reserve(abandoned) error = %v", err)
	}
	if err := store.Release(t.Context(), "abandoned"); err != nil {
		t.Fatalf("Release(abandoned) error = %v", err)
	}
	if _, err := store.Reserve(t.Context(), "abandoned", fingerprint); err != nil {
		t.Fatalf("Reserve after release error = %v, want the key to be claimable again", err)
	}

	if _, err := store.Reserve(t.Context(), "finished", fingerprint); err != nil {
		t.Fatalf("Reserve(finished) error = %v", err)
	}
	if err := store.Complete(t.Context(), "finished", idempotency.StoredResponse{
		Status:     http.StatusOK,
		Replayable: true,
	}); err != nil {
		t.Fatalf("Complete(finished) error = %v", err)
	}
	if err := store.Release(t.Context(), "finished"); err != nil {
		t.Fatalf("Release(finished) error = %v", err)
	}
	stored, err := store.Reserve(t.Context(), "finished", fingerprint)
	if err != nil {
		t.Fatalf("Reserve(finished after release) error = %v", err)
	}
	if stored == nil {
		t.Fatal("Release deleted a completed outcome; the replay it owed a retry is gone")
	}
}

// TestIdempotencyStoreSweepDeletesExpiredKeysOnly covers the other half of the
// contract: without the sweep the table keeps every key any client ever sent.
//
// Retention is set to the smallest value the constructor accepts and waited out,
// rather than faked. The clock that decides expiry is the database server's, so
// there is nothing here for a fake clock to control.
func TestIdempotencyStoreSweepDeletesExpiredKeysOnly(t *testing.T) {
	const shortRetention = time.Millisecond

	pool := newIdempotencyPool(t)
	expiring, err := postgres.NewIdempotencyStore(pool, shortRetention)
	if err != nil {
		t.Fatalf("create short-retention store: %v", err)
	}
	lasting, err := postgres.NewIdempotencyStore(pool, idempotencyTestRetention)
	if err != nil {
		t.Fatalf("create long-retention store: %v", err)
	}

	if _, err := lasting.Reserve(t.Context(), "live", "fingerprint"); err != nil {
		t.Fatalf("Reserve(live) error = %v", err)
	}
	if removed, err := lasting.Sweep(t.Context()); err != nil || removed != 0 {
		t.Fatalf("Sweep() = (%d, %v), want (0, nil) while the reservation is inside its retention", removed, err)
	}

	if _, err := expiring.Reserve(t.Context(), "expiring", "fingerprint"); err != nil {
		t.Fatalf("Reserve(expiring) error = %v", err)
	}
	time.Sleep(50 * shortRetention)

	removed, err := expiring.Sweep(t.Context())
	if err != nil {
		t.Fatalf("Sweep(expired) error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Sweep() removed %d rows, want the one expired key", removed)
	}
	if _, err := lasting.Reserve(t.Context(), "live", "fingerprint"); !errors.Is(err, idempotency.ErrInFlight) {
		t.Fatalf("Reserve(live) after sweep error = %v, want the live reservation intact", err)
	}
}

// TestIdempotentMiddlewareReplaysThroughPostgres is the end-to-end seam neither
// side proves alone: the unit tests drive the middleware with a fake store, and the
// tests above drive the real store with no middleware. What is only provable here is
// that the shipped store satisfies the port the transport consumes *behaviorally* —
// a handler runs once, the retry is answered from PostgreSQL, and the replay is
// labeled.
func TestIdempotentMiddlewareReplaysThroughPostgres(t *testing.T) {
	store := newIdempotencyStore(t, idempotencyTestRetention)

	// The outcome budget is shorter than the handler on purpose. It is the shape
	// every real write endpoint has, and the one that used to fail: the budget was
	// built before the handler ran, so by the time the outcome reached PostgreSQL
	// the context had already expired, the row kept completed_at NULL, and the
	// retry below was answered 409 in_flight rather than the first attempt's 201.
	const (
		outcomeBudget = 250 * time.Millisecond
		handlerWork   = 2 * outcomeBudget
	)

	var handled atomic.Int32
	handler := httpx.Idempotent(
		store,
		outcomeBudget,
		slog.New(slog.DiscardHandler),
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handled.Add(1)
			time.Sleep(handlerWork)
			w.Header().Set("Location", "/api/v1/articles/created")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"slug":"created"}`))
		}),
	)

	post := func() *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			"/api/v1/articles",
			strings.NewReader(`{"slug":"created"}`),
		)
		request.Header.Set(httpx.IdempotencyKeyHeader, "middleware-key")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	first := post()
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusCreated)
	}
	if first.Header().Get("Idempotent-Replay") != "" {
		t.Fatal("the first attempt is labeled as a replay")
	}

	replay := post()
	if replay.Code != http.StatusCreated {
		t.Fatalf("replay status = %d, want %d", replay.Code, http.StatusCreated)
	}
	if replay.Header().Get("Idempotent-Replay") != "true" {
		t.Fatalf("replay is not labeled: headers = %v", replay.Header())
	}
	if got := replay.Header().Get("Location"); got != "/api/v1/articles/created" {
		t.Fatalf("replayed Location = %q, want the first attempt's header", got)
	}
	if replay.Body.String() != first.Body.String() {
		t.Fatalf("replayed body = %q, want %q", replay.Body.String(), first.Body.String())
	}
	if got := handled.Load(); got != 1 {
		t.Fatalf("handler ran %d times, want exactly 1", got)
	}

	// The same key with different work must be refused rather than answered with
	// somebody else's result.
	reused := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/articles",
		strings.NewReader(`{"slug":"different"}`),
	)
	reused.Header.Set(httpx.IdempotencyKeyHeader, "middleware-key")
	reusedResponse := httptest.NewRecorder()
	handler.ServeHTTP(reusedResponse, reused)
	if reusedResponse.Code != http.StatusConflict {
		t.Fatalf("reused-key status = %d, want %d", reusedResponse.Code, http.StatusConflict)
	}
	if got := handled.Load(); got != 1 {
		t.Fatalf("handler ran %d times after a reused key, want it never re-run", got)
	}
}
