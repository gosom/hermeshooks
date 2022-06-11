package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type ServerConfig struct {
	Log          zerolog.Logger
	Addr         string
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	IdleTimeout  time.Duration
	Handler      http.Handler
}

type server struct {
	log zerolog.Logger
	srv *http.Server
}

func (s *server) Run(ctx context.Context) error {
	// TODO graceful exit
	s.log.Info().Msgf("starting server %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func New(cfg ServerConfig) (*server, error) {
	if cfg.Handler == nil {
		return nil, errors.New("please provide a Handler")
	}
	// put some sane defaults
	if len(cfg.Addr) == 0 {
		cfg.Addr = "127.0.0.1:8000"
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = time.Second * 10
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = time.Second * 5
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = time.Second * 120
	}

	ans := server{
		log: cfg.Log,
		srv: &http.Server{
			Addr:         cfg.Addr,
			WriteTimeout: cfg.WriteTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			Handler:      cfg.Handler,
		},
	}
	return &ans, nil
}
