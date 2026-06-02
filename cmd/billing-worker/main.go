package main

import (
	"log/slog"
	"os"

	"github.com/Dankosik/billing-service/cmd/billing-worker/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(os.Args[1:]); err != nil {
		slog.Error("billing worker exited", "err", err)
		os.Exit(1)
	}
}
