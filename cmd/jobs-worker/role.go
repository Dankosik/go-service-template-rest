// Package jobsworker is the jobs worker role's process entry and its
// WorkersBuilder seam. Only cmd/entrypoint imports it, and it imports no other
// role. Every linked role's package initializers run in every role's process,
// so role code keeps I/O, registration, and logging out of package
// initializers; an init here may only assign the role's builder.
package jobsworker

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
)

// Main runs the jobs worker role with args, the process arguments after
// argv[0], and returns the process exit status.
func Main(args []string) int {
	if err := bootstrap.Run(args, buildWorkers); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
