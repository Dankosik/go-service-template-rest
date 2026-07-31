package oidcjwt

import "time"

type verifierTimer interface {
	C() <-chan time.Time
	Reset(duration time.Duration)
	Stop()
}

type timerFactory func(time.Duration) verifierTimer

type realVerifierTimer struct {
	timer *time.Timer
}

func newRealVerifierTimer(duration time.Duration) *realVerifierTimer {
	return &realVerifierTimer{timer: time.NewTimer(duration)}
}

func (t *realVerifierTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t *realVerifierTimer) Reset(duration time.Duration) {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(duration)
}

func (t *realVerifierTimer) Stop() {
	t.timer.Stop()
}
