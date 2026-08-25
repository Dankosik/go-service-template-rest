package httpx

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// KeyedRateLimiter is one token bucket per key, held in a bounded map.
//
// It is per instance, not per fleet: N replicas admit up to N times the
// configured rate, and sizing it as limit/replicas breaks under a rolling deploy.
// It is the wrong shape for a contractual quota, which needs shared state behind
// this same RateLimiter interface.
//
// The key map is bounded because keys are caller-controlled: an unbounded
// map[string]*rate.Limiter keyed on anything an attacker can vary is a memory
// leak with a rate limiter attached.
type KeyedRateLimiter struct {
	limit rate.Limit
	burst int
	// maxKeys bounds one generation. Two are live at a time, so the real ceiling
	// is twice this.
	maxKeys int

	mu sync.Mutex
	// current and previous are generations rather than an LRU: eviction is one
	// pointer swap instead of per-entry bookkeeping on the hot path. A key still
	// in use is promoted out of previous on its next request; an idle one is
	// dropped a generation later.
	current  map[string]*rate.Limiter
	previous map[string]*rate.Limiter
}

const (
	defaultRateLimitMaxKeys = 8192

	// rateLimitInitialKeys caps the preallocation, so a large maxKeys does not
	// reserve the whole map up front for a service that has three callers.
	rateLimitInitialKeys = 256
)

// NewKeyedRateLimiter builds a per-key token bucket admitting perSecond requests
// with room for burst of them arriving at once, tracking at most maxKeys callers
// per generation. A non-positive maxKeys uses defaultRateLimitMaxKeys.
//
// Unusable settings are an error rather than a nil limiter: a nil
// *KeyedRateLimiter stored in a RateLimiter interface is a non-nil interface
// holding a nil pointer, so the middleware would install itself and panic on the
// first request that carried a key. A service that wants no limiting leaves the
// field unset.
func NewKeyedRateLimiter(perSecond float64, burst, maxKeys int) (*KeyedRateLimiter, error) {
	if perSecond <= 0 || math.IsNaN(perSecond) {
		return nil, errors.New("keyed rate limiter: requests per second must be > 0")
	}
	if burst <= 0 {
		return nil, errors.New("keyed rate limiter: burst must be > 0")
	}
	if maxKeys <= 0 {
		maxKeys = defaultRateLimitMaxKeys
	}

	return &KeyedRateLimiter{
		limit:   rate.Limit(perSecond),
		burst:   burst,
		maxKeys: maxKeys,
		current: make(map[string]*rate.Limiter, min(maxKeys, rateLimitInitialKeys)),
	}, nil
}

func (l *KeyedRateLimiter) Allow(_ context.Context, key string) (bool, time.Duration) {
	// Reserve and Cancel must be one operation: x/time/rate cannot fully undo an
	// earlier reservation after a later one has changed the same limiter.
	// ponytail: one lock also owns the generations; split per bucket only if
	// measured contention makes that necessary.
	l.mu.Lock()
	defer l.mu.Unlock()
	limiter := l.bucketLocked(key)

	// Reserve rather than Allow, so a rejected caller can be told how long to
	// wait. Cancel returns the token the reservation took, which keeps a
	// rejection from consuming budget the caller never got to use.
	reservation := limiter.Reserve()
	if !reservation.OK() {
		return false, time.Second
	}
	if delay := reservation.Delay(); delay > 0 {
		reservation.Cancel()
		return false, delay
	}
	return true, 0
}

// bucketLocked requires l.mu to be held by the caller.
func (l *KeyedRateLimiter) bucketLocked(key string) *rate.Limiter {
	if limiter, ok := l.current[key]; ok {
		return limiter
	}
	limiter, ok := l.previous[key]
	if !ok {
		limiter = rate.NewLimiter(l.limit, l.burst)
	}
	// A promoted caller keeps the previous limiter — and therefore its
	// accumulated debt — when a generation rolls over.
	if len(l.current) >= l.maxKeys {
		l.previous = l.current
		l.current = make(map[string]*rate.Limiter, min(l.maxKeys, rateLimitInitialKeys))
	}
	l.current[key] = limiter
	return limiter
}
