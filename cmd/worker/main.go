package main

import (
	"context"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/storage"
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
	// ----------------- db stuff --------------------------------------
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
		PgDriver:     true,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	wc := worker.WorkerConfig{
		Log:  logger,
		Node: cfg.Node,
		DB:   db,
	}

	w, err := worker.NewWorker(wc)
	if err != nil {
		return err
	}
	return w.Start(ctx)
}
