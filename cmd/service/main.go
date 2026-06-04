package main

import (
	"fmt"
	"os"

	"github.com/example/go-service-template-rest/cmd/service/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
