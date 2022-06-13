package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"
)

type ServerConfig struct {
	Log          zerolog.Logger
	Addr         string
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	IdleTimeout  time.Duration
	Handler      http.Handler
	Domain       string
}

type server struct {
	log    zerolog.Logger
	srv    *http.Server
	domain string
}

func (s *server) Run(ctx context.Context) error {
	// TODO graceful exit
	if s.domain == "localhost" {
		s.log.Info().Msgf("starting server %s", s.srv.Addr)
		return s.srv.ListenAndServe()
	}

	s.log.Info().Msgf("starting server https://%s", s.domain)
	return s.srv.Serve(autocert.NewListener(s.domain))
}

func New(cfg ServerConfig) (*server, error) {
	if cfg.Handler == nil {
		return nil, errors.New("please provide a Handler")
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
	if len(cfg.Domain) == 0 {
		cfg.Domain = "localhost"
	}
	bindAddr := "127.0.1:8000"

	if cfg.Domain == "localhost" {
		if len(cfg.Addr) == 0 {
			cfg.Addr = "127.0.0.1:8000"
		}
		bindAddr = cfg.Addr
	} else { // we have a domain server TLS
		bindAddr = ":443"
	}

	ans := server{
		log:    cfg.Log,
		domain: cfg.Domain,
		srv: &http.Server{
			Addr:         bindAddr,
			WriteTimeout: cfg.WriteTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			Handler:      cfg.Handler,
		},
	}
	return &ans, nil
}
