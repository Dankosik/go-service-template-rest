//go:build !jobs_test_worker

package jobsworker

import (
	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
)

// A concrete feature replaces this nil builder in its selected role. The
// generic pack has no default kind and must therefore fail before database I/O.
var buildWorkers bootstrap.WorkersBuilder
