package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/worker/internal/bootstrap"
)

func main() {
	// Replace nil with the feature-owned duplicate-safe handler when selecting
	// this worker for a concrete initialized service.
	if err := bootstrap.Run(os.Args[1:], nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
