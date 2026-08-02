package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/worker/internal/bootstrap"
)

func main() {
	// Replace nil with a binary-local bootstrap.HandlerBuilder when this worker
	// is selected for a concrete feature. Nil fails before config or broker I/O.
	if err := bootstrap.Run(os.Args[1:], nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
