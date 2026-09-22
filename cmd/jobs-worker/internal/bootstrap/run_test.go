package bootstrap

import (
	"context"
	"strings"
	"testing"
)

func TestJobsWorkerRunRejectsNilBuilderBeforeConfig(t *testing.T) {
	err := run(context.Background(), []string{"--config", "/does/not/exist"}, nil)
	if err == nil || !strings.Contains(err.Error(), "worker builder") {
		t.Fatalf("run() error = %v, want missing builder before config access", err)
	}
}
