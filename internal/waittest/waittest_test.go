package waittest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type fatalPanic struct{}

type recordingTB struct {
	testing.TB

	message string
}

func (*recordingTB) Helper() {}

func (tb *recordingTB) Fatalf(format string, args ...any) {
	tb.message = fmt.Sprintf(format, args...)
	panic(fatalPanic{})
}

func expectFatal(t *testing.T, tb *recordingTB, want string, run func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != (fatalPanic{}) {
			t.Fatalf("panic = %v, want fatalPanic", recovered)
		}
		if !strings.Contains(tb.message, want) {
			t.Fatalf("Fatalf() = %q, want it to contain %q", tb.message, want)
		}
	}()
	run()
	t.Fatal("run returned without Fatalf")
}

func TestUntilFuncBoundsBlockingPredicate(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		tb := &recordingTB{TB: t}
		expectFatal(t, tb, "timed out waiting for blocked predicate", func() {
			UntilFunc(tb, time.Second, func(ctx context.Context) bool {
				<-ctx.Done()
				return false
			}, func() string { return "blocked predicate" })
		})
	})
}

func TestReceiveRejectsClosedValueChannel(t *testing.T) {
	t.Parallel()

	values := make(chan int)
	close(values)
	tb := &recordingTB{TB: t}
	expectFatal(t, tb, "channel closed while waiting for value", func() {
		Receive(tb, values, time.Second, "value")
	})
}

func TestReceiveSignalAcceptsClose(t *testing.T) {
	t.Parallel()

	signal := make(chan struct{})
	close(signal)
	ReceiveSignal(t, signal, time.Second, "close signal")
}
