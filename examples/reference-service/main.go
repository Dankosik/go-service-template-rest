package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/httpapi"
)

const (
	listenAddress   = ":8080"
	shutdownTimeout = 5 * time.Second

	// writeTokenEnv supplies the demonstration credential for protected
	// operations. It is not an authentication design; see
	// docs/first-production-feature.md before building a real one.
	writeTokenEnv = "REFERENCE_WRITE_TOKEN"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, log); err != nil {
		log.Error("reference_service_stopped", "err", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, log *slog.Logger) error {
	repository, err := memory.New([]article.Article{{
		Slug:      "clear-owners",
		Title:     "Keep responsibilities with their owner",
		Summary:   "A transport maps HTTP, a use case owns behavior, and an adapter owns storage details.",
		Published: true,
	}})
	if err != nil {
		return fmt.Errorf("build article repository: %w", err)
	}
	articles, err := article.NewService(repository)
	if err != nil {
		return fmt.Errorf("build article service: %w", err)
	}
	// The demonstration credential is required rather than defaulted, so the
	// example cannot start with a guessable write token baked into source.
	writeToken := os.Getenv(writeTokenEnv)
	if strings.TrimSpace(writeToken) == "" {
		return fmt.Errorf("%s is required to run the reference example", writeTokenEnv)
	}
	handler, err := httpapi.NewRouter(articles, writeToken)
	if err != nil {
		return fmt.Errorf("build reference router: %w", err)
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen for reference HTTP: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	log.Info("reference_service_started", "http.addr", listener.Addr().String())
	return serve(ctx, listener, handler)
}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve reference HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		closeErr := server.Close()
		serveErr := <-serverErr
		var stopErr error
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			stopErr = fmt.Errorf("stop reference HTTP: %w", serveErr)
		}
		return errors.Join(
			fmt.Errorf("shutdown reference HTTP: %w", err),
			closeErr,
			stopErr,
		)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("stop reference HTTP: %w", err)
	}
	return nil
}
