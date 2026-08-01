package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/cmd/worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func main() {
	if err := bootstrap.Run(os.Args[1:], buildFeatureHandler); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildFeatureHandler is the binary-local composition point for one
// duplicate-safe feature handler and its dependencies. Replace the rejection
// with concrete feature construction; transport lifecycle stays in bootstrap.
func buildFeatureHandler(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(), error) {
	return nil, nil, fmt.Errorf("%w: worker feature handler is not registered", natsjs.ErrRejected)
}
