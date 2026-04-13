package httpx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Server struct {
	srv *http.Server
}

var (
	// ErrNilListener indicates Serve received a nil listener.
	ErrNilListener = errors.New("nil listener")

	// ErrUninitializedServer indicates an exported Server method was called before New.
	ErrUninitializedServer = errors.New("uninitialized server")
)

type Config struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

func New(cfg Config, handler http.Handler) *Server {
	if handler == nil {
		handler = http.NotFoundHandler()
	}

	return &Server{
		srv: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
	}
}

func (s *Server) Serve(listener net.Listener) error {
	if err := s.requireInitialized(); err != nil {
		return err
	}
	if listener == nil {
		return ErrNilListener
	}
	err := s.srv.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve http: %w", err)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.requireInitialized(); err != nil {
		return err
	}
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

func (s *Server) requireInitialized() error {
	if s == nil || s.srv == nil {
		return ErrUninitializedServer
	}
	return nil
}
