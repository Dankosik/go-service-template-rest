package worker

import (
	"context"
	"testing"
	"time"
)

func TestWorkerStopsAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runWorker(ctx, func() {
			time.Sleep(5 * time.Millisecond)
		})
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	default:
		t.Fatal("worker did not stop")
	}
}
