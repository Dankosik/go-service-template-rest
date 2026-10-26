// Package worker is the messaging worker role's process entry. Only
// cmd/entrypoint imports it, and it imports no other role. Every linked role's
// package initializers run in every role's process, so role code keeps I/O,
// registration, and logging out of package initializers.
package worker

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/worker/internal/bootstrap"
)

// Main runs the messaging worker role with args, the process arguments after
// argv[0], and returns the process exit status.
func Main(args []string) int {
	// Replace nil with a role-local bootstrap.HandlerBuilder that returns a
	// typed natsjs.Registry. Nil fails before config or broker I/O.
	if err := bootstrap.Run(args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
