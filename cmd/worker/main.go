package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/worker/internal/bootstrap"
)

func main() {
	// Replace nil with a binary-local bootstrap.HandlerBuilder that returns a
	// typed natsjs.Registry. Nil fails before config or broker I/O.
	if err := bootstrap.Run(os.Args[1:], nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
