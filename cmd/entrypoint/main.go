// Command entrypoint is the image's only executable. Each process runs exactly
// one role, chosen from the final element of argv[0], so the image's entry
// paths are links to this one file. The selection reads no configuration,
// environment, or flags; each role owns all of that behind its Main.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/example/go-service-template-rest/cmd/service"
	// profile:messaging-nats-jetstream:start
	"github.com/example/go-service-template-rest/cmd/worker"
	// profile:messaging-nats-jetstream:end
	// profile:jobs-postgres:start
	jobsworker "github.com/example/go-service-template-rest/cmd/jobs-worker"
	// profile:jobs-postgres:end
	// profile:outbox-postgres:start
	outboxrelay "github.com/example/go-service-template-rest/cmd/outbox-relay"
	// profile:outbox-postgres:end
	// profile:database-postgres:start
	"github.com/example/go-service-template-rest/cmd/migrate"
	// profile:database-postgres:end
)

// pinnedRole, when set with -ldflags -X main.pinnedRole=<name>, replaces the
// argv[0] name. Only local make artifacts set it, because their file names
// cannot match a role name; the shipped image never does.
var pinnedRole string

type role struct {
	name string
	main func(args []string) int
}

// roles lists the selected roles in entry-path order.
var roles = []role{
	{name: "service", main: service.Main},
	// profile:messaging-nats-jetstream:start
	{name: "worker", main: worker.Main},
	// profile:messaging-nats-jetstream:end
	// profile:jobs-postgres:start
	{name: "jobs-worker", main: jobsworker.Main},
	// profile:jobs-postgres:end
	// profile:outbox-postgres:start
	{name: "outbox-relay", main: outboxrelay.Main},
	// profile:outbox-postgres:end
	// profile:database-postgres:start
	{name: "migrate", main: migrate.Main},
	// profile:database-postgres:end
}

func main() {
	os.Exit(run(pinnedRole, os.Args, roles, os.Stderr))
}

// run starts the role named by pinned, or by the final element of argv[0] when
// pinned is empty, and returns its exit status. A name that matches no role
// exactly gets one stderr line and status 2, and no role code runs.
func run(pinned string, argv []string, table []role, stderr io.Writer) int {
	name := invokedName(pinned, argv)
	selected, ok := lookup(table, name)
	if !ok {
		_, _ = fmt.Fprintln(stderr, rejection(name, table))
		return 2
	}
	var args []string
	if len(argv) > 0 {
		args = argv[1:]
	}
	return selected.main(args)
}

func invokedName(pinned string, argv []string) string {
	if pinned != "" {
		return pinned
	}
	if len(argv) == 0 {
		return ""
	}
	return argv[0][strings.LastIndexByte(argv[0], '/')+1:]
}

func lookup(table []role, name string) (role, bool) {
	for _, candidate := range table {
		if candidate.name == name {
			return candidate, true
		}
	}
	return role{}, false
}

func rejection(name string, table []role) string {
	names := make([]string, 0, len(table))
	for _, candidate := range table {
		names = append(names, candidate.name)
	}
	return fmt.Sprintf("entrypoint: unknown role %q; selected roles: %s", name, strings.Join(names, ", "))
}
