// Package outboxrelay is the outbox relay role's process entry. Only
// cmd/entrypoint imports it, and it imports no other role. Every linked role's
// package initializers run in every role's process, so role code keeps I/O,
// registration, and logging out of package initializers.
package outboxrelay

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/outbox-relay/internal/bootstrap"
)

// Main runs the outbox relay role with args, the process arguments after
// argv[0], and returns the process exit status.
func Main(args []string) int {
	if err := bootstrap.Run(args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
