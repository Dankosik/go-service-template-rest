package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestJobsWorkerRunRejectsNilBuilderBeforeConfig(t *testing.T) {
	err := run(context.Background(), []string{"--config", "/does/not/exist"}, nil)
	if err == nil || !strings.Contains(err.Error(), "worker builder") {
		t.Fatalf("run() error = %v, want missing builder before config access", err)
	}
}

func TestRiverCleanupSafetyTracksClientExit(t *testing.T) {
	t.Parallel()

	stopped := make(chan struct{})
	close(stopped)
	if !riverStoppedBeforeReturn(nil, stopped) {
		t.Fatal("completed graceful stop was not safe to clean up")
	}
	if !riverStoppedBeforeReturn(errors.New("forced"), stopped) {
		t.Fatal("completed forced stop was not safe to clean up")
	}
	if riverStoppedBeforeReturn(errors.New("forced"), make(chan struct{})) {
		t.Fatal("running River client was marked safe to clean up")
	}
}
