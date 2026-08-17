package postgreswebhook

import (
	"errors"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

func DecorrelatedJitter(previous, base, maxDelay time.Duration, random uint64) time.Duration {
	if base <= 0 || maxDelay < base {
		return 0
	}
	if previous < base {
		previous = base
	}
	var upper time.Duration
	if previous <= time.Duration(math.MaxInt64/3) {
		upper = previous * 3
	} else {
		upper = maxDelay
	}
	upper = min(upper, maxDelay)
	span, err := uint64Value(int64(upper - base))
	if err != nil {
		return 0
	}
	offset, err := durationValue(random % (span + 1))
	if err != nil {
		return 0
	}
	return base + offset
}

type TransportEvidence struct {
	StatusCode        int
	DefinitelyNotSent bool
	MayHaveSent       bool
	LocalDenial       bool
}

func ClassifyOutcome(evidence TransportEvidence) OutcomeClass {
	if evidence.LocalDenial {
		return OutcomeLocallyDenied
	}
	if evidence.StatusCode >= 200 && evidence.StatusCode <= 299 {
		return OutcomeHTTPAccepted
	}
	if evidence.StatusCode == http.StatusRequestTimeout || evidence.StatusCode == http.StatusTooEarly || evidence.StatusCode == http.StatusTooManyRequests || evidence.StatusCode >= 500 && evidence.StatusCode <= 599 && evidence.StatusCode != http.StatusNotImplemented && evidence.StatusCode != http.StatusHTTPVersionNotSupported {
		return OutcomeRetryableHTTPAmbiguous
	}
	if evidence.StatusCode >= 100 {
		return OutcomeHTTPRejected
	}
	if evidence.MayHaveSent {
		return OutcomeTransportAmbiguous
	}
	if evidence.DefinitelyNotSent {
		return OutcomeDefinitelyNotSentRetry
	}
	return OutcomeTransportAmbiguous
}

func ParseRetryAfter(raw, date string, attemptedAt time.Time, maxDelay time.Duration) (time.Duration, bool) {
	if maxDelay <= 0 {
		return 0, false
	}
	raw = strings.TrimSpace(raw)
	if raw != "" && strings.IndexFunc(raw, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		seconds, err := strconv.ParseUint(raw, 10, 63)
		if err != nil {
			return maxDelay, true
		}
		if seconds > uint64(math.MaxInt64/int64(time.Second)) {
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

func CumulativeSummary(previous OutcomeClass, attempts ...OutcomeClass) OutcomeClass {
	all := append([]OutcomeClass{previous}, attempts...)
	if slices.Contains(all, OutcomeHTTPAccepted) {
		return OutcomeHTTPAccepted
	}
	for _, outcome := range all {
		if outcome == OutcomeTransportAmbiguous || outcome == OutcomeRetryableHTTPAmbiguous || outcome == OutcomeUnknown {
			return OutcomeUnknown
		}
	}
	if previous == OutcomeClosedUnknown {
		return OutcomeClosedUnknown
	}
	for _, outcome := range slices.Backward(all) {
		if outcome != "" {
			return outcome
		}
	}
	return OutcomeAttemptsExhausted
}

func RetryDue(now, deadline time.Time, localDelay, hint time.Duration) (time.Time, error) {
	if !deadline.After(now) {
		return time.Time{}, errors.New("retry deadline exhausted")
	}
	due := now.Add(max(localDelay, hint))
	if !due.Before(deadline) {
		return time.Time{}, errors.New("retry deadline exhausted")
	}
	return due, nil
}
