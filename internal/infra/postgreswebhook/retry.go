package postgreswebhook

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type transportEvidence struct {
	StatusCode        int
	DefinitelyNotSent bool
	MayHaveSent       bool
	LocalDenial       bool
}

func ParseRetryAfter(raw, date string, attemptedAt time.Time, maxDelay time.Duration) (time.Duration, bool) {
	if maxDelay <= 0 {
		return 0, false
	}
	raw = strings.TrimSpace(raw)
	if raw != "" && strings.IndexFunc(raw, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		seconds, err := strconv.ParseUint(raw, 10, 63)
		if err != nil || seconds > uint64(math.MaxInt64/int64(time.Second)) {
			return maxDelay, true
		}
		return min(time.Duration(seconds)*time.Second, maxDelay), true
	}
	when, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	base := attemptedAt
	if parsedDate, parseErr := http.ParseTime(date); parseErr == nil {
		base = parsedDate
	}
	if !when.After(base) {
		return 0, false
	}
	return min(when.Sub(base), maxDelay), true
}
