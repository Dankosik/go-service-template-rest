package main

import (
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
)

func TestJobsWorkerMainRejectsNilBuilder(t *testing.T) {
	t.Parallel()
	err := bootstrap.Run(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "registry builder") {
		t.Fatalf("Run(nil) error = %v, want missing registry builder", err)
	}
}
