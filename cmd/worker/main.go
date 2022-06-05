package main

import (
	"context"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		panic(err)
	}

	logger := common.NewLogger()

	if err := run(ctx, logger, cfg); err != nil {
		logger.Panic().AnErr("error", err).Msg("exiting with error")
	}
}

type config struct {
	Debug bool   `envconfig:"DEBUG" default:"false"`
	Node  string `envconfig:"NODE" default:"http://localhost:8000"`
	DSN   string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
}

func run(ctx context.Context, logger zerolog.Logger, cfg config) error {
	wc := worker.WorkerConfig{
		Log:  logger,
		Node: cfg.Node,
	}

	w, err := worker.NewWorker(wc)
	if err != nil {
		return err
	}
	return w.Start(ctx)
}
