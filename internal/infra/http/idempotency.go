package httpx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"
)

// IdempotencyKeyHeader is the request header this middleware keys on.
const IdempotencyKeyHeader = "Idempotency-Key"

// idempotencyRetryAfter is the hint on a key that is still being processed. It is
// short because the first attempt is running now, not queued behind an outage.
const idempotencyRetryAfter = time.Second

// maxIdempotentResponseBytes bounds what is held for replay.
//
// A response past the cap is recorded as used-but-not-replayable rather than
// dropped: the safety property is that the work runs once, and giving up on the
// exact bytes must not give up on that.
const maxIdempotentResponseBytes = 1 << 20

var (
	// ErrIdempotencyInFlight reports that another attempt with the same key has
	// not finished. The caller retries; it must not be told the work conflicts.
	ErrIdempotencyInFlight = errors.New("idempotency key is in flight")

	// ErrIdempotencyKeyReused reports that the key was already spent on a
	// different request. This is a client defect — the same key must identify the
	// same intent — and it is the one case where replay would be dangerous.
	ErrIdempotencyKeyReused = errors.New("idempotency key was used for a different request")
)

// StoredResponse is a completed response held for replay.
type StoredResponse struct {
	Status int
	Header http.Header
	Body   []byte
	// Replayable is false when the response was too large to hold. A repeat then
	// learns the key is spent instead of re-running the work.
	Replayable bool
}

// IdempotencyStore holds one outcome per key.
//
// A correct implementation is the part that cannot be written here, because it
// depends on storage this template does not choose. Two properties are not
// negotiable. Reserve must be a single atomic claim — for PostgreSQL, an
// INSERT ... ON CONFLICT DO NOTHING against a primary key, so two concurrent
// attempts cannot both win — and entries must expire, or the table grows for the
// life of the service. An in-memory map satisfies neither the moment a second
// replica exists, which is the version this seam is here to prevent.
type IdempotencyStore interface {
	// Reserve claims key for this attempt.
	//
	// It returns a stored response when a previous attempt with the same
	// fingerprint completed, ErrIdempotencyInFlight when another attempt holds
	// the key, and ErrIdempotencyKeyReused when the key was spent on a different
	// request. A nil response with a nil error means this attempt owns the key.
	Reserve(ctx context.Context, key, fingerprint string) (*StoredResponse, error)

	// Complete records the outcome for replay.
	Complete(ctx context.Context, key string, response StoredResponse) error

	// Release abandons a reservation whose attempt produced no final answer, so a
	// retry is not refused for work that never happened.
	Release(ctx context.Context, key string) error
}

// Idempotent makes a repeated unsafe request answer with the first attempt's
// result instead of doing the work twice.
//
// Clients retry POST. A request whose 201 was lost to a spent request budget, a
// dropped connection, or a drain is indistinguishable to the caller from one that
// never arrived, so its library sends it again — and without a key the service
// either creates a second resource or reports a conflict for the caller's own
// earlier success. Neither is what happened.
//
// Requests with no Idempotency-Key are untouched, so this is opt-in per caller
// and adding it breaks nothing. Safe methods are untouched, because replaying a
// GET is what a cache is for.
//
// A 5xx is never stored. A server fault is the one outcome a client should be
// free to retry, and holding it would turn one bad minute into a key that
// answers 500 until it expires.
func Idempotent(store IdempotencyStore, log *slog.Logger, next http.Handler) http.Handler {
	if store == nil {
		return next
	}
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" || !idempotentMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if !validRequestID(key) {
			writeProblem(w, r, problemResponse{
				code:   problemCodeBadRequest,
				detail: "idempotency key must be 1..128 unreserved characters",
			})
			return
		}

		fingerprint, err := requestFingerprint(r)
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				writeProblem(w, r, problemResponse{code: problemCodeRequestEntityTooLarge, detail: "request body exceeds limit"})
				return
			}
			writeMalformedRequestProblem(w, r)
			return
		}

		stored, err := store.Reserve(r.Context(), key, fingerprint)
		if err != nil {
			rejectIdempotencyReservation(w, r, log, err)
			return
		}
		if stored != nil {
			replayStoredResponse(w, r, *stored)
			return
		}

		// Detached from the request: the outcome has to be recorded, or released,
		// even when the budget that admitted the request has since expired.
		outcomeCtx := context.WithoutCancel(r.Context())

		recorder := newIdempotentRecorder(w)
		completed := false
		defer func() {
			// A panic unwinding through here, or a handler that committed
			// nothing, must not leave the key claimed: the work did not finish,
			// so a retry has to be allowed to run it.
			if completed {
				return
			}
			if releaseErr := store.Release(outcomeCtx, key); releaseErr != nil {
				log.Error("idempotency_release_failed", "err", releaseErr)
			}
		}()

		next.ServeHTTP(recorder, r)

		if recorder.status >= http.StatusInternalServerError || recorder.status == 0 {
			return
		}
		if completeErr := store.Complete(outcomeCtx, key, recorder.stored()); completeErr != nil {
			// The response already reached the client, so this cannot fail the
			// request. It is logged because it means a retry will re-run work
			// that already happened, which is the exact outcome this exists to
			// prevent and an operator has to be able to see it.
			log.Error("idempotency_complete_failed", "err", completeErr)
			return
		}
		completed = true
	})
}

func idempotentMethod(method string) bool {
	return slices.Contains([]string{http.MethodPost, http.MethodPatch}, method)
}

func rejectIdempotencyReservation(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, ErrIdempotencyInFlight):
		w.Header().Set("Retry-After", strconv.Itoa(int(idempotencyRetryAfter.Seconds())))
		writeProblem(w, r, problemResponse{
			code:   problemCodeConflict,
			detail: "a request with this idempotency key is still being processed",
		})
	case errors.Is(err, ErrIdempotencyKeyReused):
		writeProblem(w, r, problemResponse{
			code:   problemCodeConflict,
			detail: "this idempotency key was already used for a different request",
		})
	default:
		log.Error("idempotency_reserve_failed", "err", err)
		writeProblem(w, r, problemResponse{code: problemCodeInternalError, detail: "request failed"})
	}
}

func replayStoredResponse(w http.ResponseWriter, r *http.Request, stored StoredResponse) {
	if !stored.Replayable {
		writeProblem(w, r, problemResponse{
			code:   problemCodeConflict,
			detail: "this idempotency key was already used and its response is no longer available",
		})
		return
	}

	for name, values := range stored.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	// The replay is labeled, so a client and an operator can both tell a repeat
	// from the original without diffing bodies.
	w.Header().Set("Idempotent-Replay", "true")
	w.WriteHeader(stored.Status)
	_, _ = w.Write(stored.Body)
}

// requestFingerprint identifies the intent a key stands for, so the same key
// presented with a different request is refused rather than answered with
// somebody else's result.
//
// The body is read into memory and put back. That is bounded: RequestBodyLimit
// has already wrapped it in a MaxBytesReader, and the generated validator reads
// it the same way for any secured operation.
func requestFingerprint(r *http.Request) (string, error) {
	digest := sha256.New()
	_, _ = io.WriteString(digest, r.Method+"\n"+r.URL.Path+"\n")

	if r.Body == nil || r.Body == http.NoBody {
		return hex.EncodeToString(digest.Sum(nil)), nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Passed through unchanged so the caller can still classify a
		// MaxBytesError as 413 rather than a malformed request.
		return "", err //nolint:wrapcheck // Classification by the caller requires the original error identity.
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	_, _ = digest.Write(body)

	return hex.EncodeToString(digest.Sum(nil)), nil
}

// idempotentRecorder captures a response while passing it through, so the client
// is answered at handler speed rather than after a store round trip.
type idempotentRecorder struct {
	http.ResponseWriter
	status     int
	body       bytes.Buffer
	overflowed bool
}

func newIdempotentRecorder(w http.ResponseWriter) *idempotentRecorder {
	return &idempotentRecorder{ResponseWriter: w}
}

func (r *idempotentRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *idempotentRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if !r.overflowed {
		if r.body.Len()+len(b) > maxIdempotentResponseBytes {
			r.overflowed = true
			r.body.Reset()
		} else {
			r.body.Write(b)
		}
	}
	written, err := r.ResponseWriter.Write(b)
	if err != nil {
		return written, err //nolint:wrapcheck // Passing the writer's own error through unchanged is the contract.
	}
	return written, nil
}

func (r *idempotentRecorder) stored() StoredResponse {
	return StoredResponse{
		Status:     r.status,
		Header:     r.Header().Clone(),
		Body:       slices.Clone(r.body.Bytes()),
		Replayable: !r.overflowed,
	}
}
