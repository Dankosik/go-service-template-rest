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

	"github.com/example/go-service-template-rest/internal/idempotency"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/felixge/httpsnoop"
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

// IdempotencyStore holds one outcome per key.
//
// It is declared here, by the consumer, so an adapter satisfies it structurally
// rather than by importing this package. internal/infra/postgres.IdempotencyStore
// is the implementation this repository ships.
//
// Two properties are not negotiable, and both are storage decisions. Reserve must
// be a single atomic claim, so two concurrent attempts cannot both win — for
// PostgreSQL that is INSERT ... ON CONFLICT DO NOTHING against a primary key. And
// entries must expire, or the table grows for the life of the service. An
// in-memory map satisfies neither the moment a second replica exists, which is the
// version this seam exists to prevent.
type IdempotencyStore interface {
	// Reserve claims key for this attempt.
	//
	// It returns a stored response when a previous attempt with the same
	// fingerprint completed, idempotency.ErrInFlight when another attempt holds
	// the key, and idempotency.ErrKeyReused when the key was spent on a different
	// request. A nil response with a nil error means this attempt owns the key.
	Reserve(ctx context.Context, key, fingerprint string) (*idempotency.StoredResponse, error)

	// Complete records the outcome for replay.
	Complete(ctx context.Context, key string, response idempotency.StoredResponse) error

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
// This belongs inside the generated router's per-operation middleware list,
// ahead of the request validator so the validator wraps it. Installed further
// out — in the root chain, where it looks like it belongs — it claims keys for
// unauthenticated callers and unrouted paths, and records the resulting
// rejection as the key's outcome. See storableOutcome.
//
// A 5xx is never stored. A server fault is the one outcome a client should be
// free to retry, and holding it would turn one bad minute into a key that
// answers 500 until it expires.
//
// outcomeTimeout bounds recording the outcome after the client has been answered;
// see idempotencyOutcomeContext for why that call needs a bound of its own.
func Idempotent(store IdempotencyStore, outcomeTimeout time.Duration, log *slog.Logger, next http.Handler) http.Handler {
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
				code:   problem.CodeBadRequest,
				detail: "idempotency key must be 1..128 unreserved characters",
			})
			return
		}

		fingerprint, err := requestFingerprint(r)
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				writeProblem(w, r, problemResponse{code: problem.CodeRequestEntityTooLarge, detail: "request body exceeds limit"})
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

		recorder := newIdempotentRecorder(w)
		completed := false
		// The release derives its own context from the request rather than taking
		// one: the point of this budget is that it outlives the request that
		// earned it, so a spent handler budget cannot stop the compensating write.
		//nolint:contextcheck // idempotencyOutcomeContext detaches from r.Context() deliberately; see its documentation.
		defer func() {
			// A panic unwinding through here, or a handler that committed
			// nothing, must not leave the key claimed: the work did not finish,
			// so a retry has to be allowed to run it.
			if completed {
				return
			}
			outcomeCtx, cancelOutcome := idempotencyOutcomeContext(r.Context(), outcomeTimeout)
			defer cancelOutcome()
			if releaseErr := store.Release(outcomeCtx, key); releaseErr != nil {
				log.ErrorContext(outcomeCtx, "idempotency_release_failed", "err", releaseErr)
			}
		}()

		next.ServeHTTP(recorder.writer, r)

		if !storableOutcome(recorder.status) {
			return
		}
		outcomeCtx, cancelOutcome := idempotencyOutcomeContext(r.Context(), outcomeTimeout)
		defer cancelOutcome()
		if completeErr := store.Complete(outcomeCtx, key, recorder.stored()); completeErr != nil {
			// The response already reached the client, so this cannot fail the
			// request. It is logged because it means a retry will re-run work
			// that already happened, which is the exact outcome this exists to
			// prevent and an operator has to be able to see it.
			log.ErrorContext(outcomeCtx, "idempotency_complete_failed", "err", completeErr)
			return
		}
		completed = true
	})
}

// idempotencyOutcomeContext bounds recording or releasing a reservation after the
// handler has answered.
//
// It must be built at the point of use, once the handler has returned. The
// deadline is absolute from the moment of the call, so a context built before
// next.ServeHTTP would spend its whole budget on the handler: with the shipped
// defaults, every request slower than one second reached Complete with an already
// expired context, failed instantly, then failed the compensating Release the same
// way — leaving the key claimed and uncompleted until it expired. A client
// retrying such a request was answered 409 in_flight for the whole retention
// window and never learned the outcome of work that had already run.
//
// Detached from the request, because the outcome has to be recorded even when the
// budget that admitted the request has since expired — and bounded, because
// context.WithoutCancel returns a context with no deadline at all. Without the
// bound these were the only two calls in the request path with no time limit: a
// store whose row lock was held, or whose connection was wedged mid-retransmit,
// blocked the handler goroutine here forever, after the client had already been
// answered. With http.max_in_flight at its default, 256 such requests exhaust the
// shedding semaphore permanently, so the service answers 503 to everything —
// including requests that touch no store — while readiness keeps reporting ok,
// because a cached PostgreSQL ping still succeeds. Nothing recovers it but a
// restart.
//
// A spent budget here costs one duplicated unit of work on a retry, and says so in
// the log. That is the cheaper failure by a wide margin.
func idempotencyOutcomeContext(base context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	detached := context.WithoutCancel(base)
	if timeout <= 0 {
		return context.WithCancel(detached)
	}
	return context.WithTimeout(detached, timeout)
}

func idempotentMethod(method string) bool {
	return slices.Contains([]string{http.MethodPost, http.MethodPatch}, method)
}

// storableOutcome reports whether a status is a result worth replaying.
//
// A key stands for one unit of work, so only an answer that reflects whether
// that work happened may be stored. The refusals below do not:
//
//   - 401 and 403 describe the credential presented on this attempt, not the
//     request. Storing them meant a client that refreshed an expired token and
//     retried the byte-identical request was replayed its own 401 until the key
//     expired, because the fingerprint covers the request and not the credential.
//   - 408 and 429 are explicit invitations to try again, so recording either as
//     the final answer contradicts what the service just told the client.
//
// 5xx is refused for the same reason it always was: a server fault is the one
// outcome a client should be free to retry. A zero status means the handler
// committed nothing at all, which is not an outcome either. Everything left is
// a decision about the request itself — a 201, a validation 400, a domain 409 —
// and repeating it is exactly what the key is for.
func storableOutcome(status int) bool {
	switch status {
	case 0,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusRequestTimeout,
		http.StatusTooManyRequests:
		return false
	}
	return status < http.StatusInternalServerError
}

func rejectIdempotencyReservation(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, idempotency.ErrInFlight):
		w.Header().Set("Retry-After", strconv.Itoa(int(idempotencyRetryAfter.Seconds())))
		writeProblem(w, r, problemResponse{
			code:   problem.CodeConflict,
			detail: "a request with this idempotency key is still being processed",
		})
	case errors.Is(err, idempotency.ErrKeyReused):
		writeProblem(w, r, problemResponse{
			code:   problem.CodeConflict,
			detail: "this idempotency key was already used for a different request",
		})
	default:
		log.ErrorContext(r.Context(), "idempotency_reserve_failed", "err", err)
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "request failed"})
	}
}

func replayStoredResponse(w http.ResponseWriter, r *http.Request, stored idempotency.StoredResponse) {
	if !stored.Replayable {
		writeProblem(w, r, problemResponse{
			code:   problem.CodeConflict,
			detail: "this idempotency key was already used and its response is no longer available",
		})
		return
	}

	// Replaced rather than added. stored.Header carries only what the handler
	// itself produced, so a key present here is one the handler owned, and the
	// value it chose is the one the replay has to reproduce.
	for name, values := range stored.Header {
		w.Header().Del(name)
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
// The query is part of the intent and was the expensive omission. A contract
// that expresses a variant in the query rather than the body — ?format=pdf,
// ?dryRun=true, ?version=2 — produced one fingerprint for every variant, so the
// second request with the same key was answered with the first one's stored
// 201 and its Location header. That is a silent wrong answer: the client is
// told the work succeeded and the work it asked for never ran. It is
// canonicalized through url.Values.Encode, which sorts by key, so a client that
// reorders its parameters between attempts is still the same request.
//
// Content-Type is included for the same reason: the same bytes mean different
// things as JSON and as CSV.
//
// The body is read into memory and put back. That is bounded: RequestBodyLimit
// has already wrapped it in a MaxBytesReader, and the generated validator reads
// it the same way for any operation that declares one.
func requestFingerprint(r *http.Request) (string, error) {
	digest := sha256.New()
	_, _ = io.WriteString(digest, r.Method+"\n"+r.URL.Path+"\n")
	_, _ = io.WriteString(digest, r.URL.Query().Encode()+"\n")
	_, _ = io.WriteString(digest, r.Header.Get("Content-Type")+"\n")

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
//
// The capture is built on httpsnoop rather than by embedding http.ResponseWriter
// in a struct. Embedding the interface promotes only its three methods, so the
// handler beneath received a writer that was not an http.Flusher, not an
// http.Hijacker, and opaque to http.ResponseController — meaning enabling
// idempotency silently broke every streamed response and every SetWriteDeadline
// in the service, on a deploy that changed no handler. httpsnoop composes the
// wrapper from the interfaces the underlying writer actually implements, which is
// why it is already a dependency here: trackResponseCommit needs the same
// property.
type idempotentRecorder struct {
	writer     http.ResponseWriter
	status     int
	body       bytes.Buffer
	overflowed bool
	// inherited is the response header as it stood before the handler ran.
	//
	// This middleware is the innermost one, so by the time it looks at the
	// header map, RequestCorrelation has already written X-Request-ID into it
	// and SecurityHeaders has already written X-Content-Type-Options — into the
	// same map, because a ResponseWriter has exactly one. Storing those as if
	// the handler produced them made every replay carry two X-Request-ID values,
	// the second naming the request that happened to run first, and a duplicated
	// nosniff. Only keys the handler actually changed are stored; see
	// storedHeader.
	inherited http.Header
}

func newIdempotentRecorder(w http.ResponseWriter) *idempotentRecorder {
	recorder := &idempotentRecorder{inherited: w.Header().Clone()}
	recorder.writer = httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(status int) {
				if recorder.status == 0 {
					recorder.status = status
				}
				next(status)
			}
		},
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				recorder.observeImplicitStatus()
				recorder.capture(b)
				return next(b)
			}
		},
		ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return func(src io.Reader) (int64, error) {
				// A handler that hands the writer a reader — io.Copy, or
				// http.ServeContent — bypasses Write entirely. Without this hook
				// the recorder would store an empty body while still reporting it
				// replayable, so the retry would be answered 200 with nothing in
				// it. Teeing keeps the capture honest under the same cap.
				recorder.observeImplicitStatus()
				return next(io.TeeReader(src, captureWriter{recorder: recorder}))
			}
		},
	})
	return recorder
}

// observeImplicitStatus records the status net/http will send for a body write
// that was not preceded by WriteHeader.
func (r *idempotentRecorder) observeImplicitStatus() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
}

func (r *idempotentRecorder) capture(b []byte) {
	if r.overflowed {
		return
	}
	if r.body.Len()+len(b) > maxIdempotentResponseBytes {
		r.overflowed = true
		r.body.Reset()
		return
	}
	r.body.Write(b)
}

func (r *idempotentRecorder) stored() idempotency.StoredResponse {
	return idempotency.StoredResponse{
		Status:     r.status,
		Header:     r.storedHeader(),
		Body:       slices.Clone(r.body.Bytes()),
		Replayable: !r.overflowed,
	}
}

// storedHeader keeps the response headers this handler owns and drops the ones
// the middleware outside it had already set.
//
// The comparison is by value rather than by key, so a handler that deliberately
// overrides an inherited header still has that override replayed. A key whose
// values are untouched belongs to an outer middleware, which will set it again
// for the request being replayed to — with that request's own identifiers.
func (r *idempotentRecorder) storedHeader() http.Header {
	current := r.writer.Header()
	stored := make(http.Header, len(current))
	for name, values := range current {
		if slices.Equal(r.inherited[name], values) {
			continue
		}
		stored[name] = slices.Clone(values)
	}
	return stored
}

// captureWriter adapts the recorder's capture to io.Writer for the ReadFrom tee.
// It never fails: a capture that cannot hold the bytes marks the response
// unreplayable rather than breaking the response the client is receiving.
type captureWriter struct {
	recorder *idempotentRecorder
}

func (w captureWriter) Write(b []byte) (int, error) {
	w.recorder.capture(b)
	return len(b), nil
}
