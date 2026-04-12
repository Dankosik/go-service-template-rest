package httpx

import (
	"context"
	"errors"
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
	Addr              string
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
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
	}
}

func (s *Server) Run() error {
	if err := s.requireInitialized(); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
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
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.requireInitialized(); err != nil {
		return err
	}
	return s.srv.Shutdown(ctx)
}

func (s *Server) requireInitialized() error {
	if s == nil || s.srv == nil {
		return ErrUninitializedServer
	}
	return nil
}
