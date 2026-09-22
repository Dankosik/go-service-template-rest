package runtimeopts

import (
	"errors"
	"testing"
	"testing/synctest"
)

func TestStoppedBeforeReturnTracksRuntimeCompletion(t *testing.T) {
	t.Parallel()

	stopped := make(chan struct{})
	close(stopped)
	if !StoppedBeforeReturn(nil, stopped) {
		t.Fatal("completed graceful stop was not safe to clean up")
	}
	if !StoppedBeforeReturn(errors.New("forced"), stopped) {
		t.Fatal("completed forced stop was not safe to clean up")
	}
	if StoppedBeforeReturn(errors.New("forced"), make(chan struct{})) {
		t.Fatal("running forced runtime was marked safe to clean up")
	}
}

func TestStoppedBeforeReturnWaitsForSuccessfulStop(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		stopped := make(chan struct{})
		returned := make(chan bool, 1)
		go func() {
			returned <- StoppedBeforeReturn(nil, stopped)
		}()

		synctest.Wait()
		select {
		case <-returned:
			t.Fatal("successful stop returned before runtime completion")
		default:
		}

		close(stopped)
		if !<-returned {
			t.Fatal("completed successful stop was not safe to clean up")
		}
	})
}
