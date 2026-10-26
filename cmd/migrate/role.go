// Package migrate is the migration role's process entry. Only cmd/entrypoint
// imports it, and it imports no other role. Every linked role's package
// initializers run in every role's process, so role code keeps I/O,
// registration, and logging out of package initializers.
package migrate

import (
	"os"
)

// Main runs the migration role with args, the process arguments after argv[0],
// and returns the process exit status.
func Main(args []string) int {
	// run already logged a classified terminal record. The error text is not
	// printed because it can carry the DSN or failing SQL.
	if err := run(args, os.Stdout); err != nil {
		return 1
	}
	return 0
}
