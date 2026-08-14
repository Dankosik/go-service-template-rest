package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(os.Args[1:], buildRegistry); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
