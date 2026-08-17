package bootstrap

import (
	"context"
	"strings"
	"testing"
)

func TestJobsWorkerRunRejectsNilBuilderBeforeConfig(t *testing.T) {
	t.Parallel()
	err := run(context.Background(), []string{"--config", "/does/not/exist"}, nil)
	if err == nil || !strings.Contains(err.Error(), "registry builder") {
		t.Fatalf("run() error = %v, want missing builder before config access", err)
	}
}
