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

// ErrNilListener indicates Serve received a nil listener.
var ErrNilListener = errors.New("nil listener")

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
	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
}

func (s *Server) Serve(listener net.Listener) error {
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
	return s.srv.Shutdown(ctx)
}
