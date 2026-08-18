package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/outbox-relay/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
