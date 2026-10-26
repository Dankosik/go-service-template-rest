// Package service is the service role's process entry. Only cmd/entrypoint
// imports it, and it imports no other role. Every linked role's package
// initializers run in every role's process, so role code keeps I/O,
// registration, and logging out of package initializers.
package service

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/service/internal/bootstrap"
)

// Main runs the service role with args, the process arguments after argv[0],
// and returns the process exit status.
func Main(args []string) int {
	if err := bootstrap.Run(args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}
