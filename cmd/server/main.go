package main

import (
	"context"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/rest"
	"github.com/gosom/hermeshooks/internal/services/scheduledjobs"
	"github.com/gosom/hermeshooks/internal/services/workers"
	"github.com/gosom/hermeshooks/internal/storage"
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
	DSN   string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
}

func run(ctx context.Context, logger zerolog.Logger, cfg config) error {
	// ----------------- db stuff --------------------------------------
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
	})
	if err != nil {
		return err
	}
	defer db.Close()

	// -----------------------------------------------------------------
	jobSrv := scheduledjobs.New(
		scheduledjobs.ServiceConfig{
			Log: logger,
			DB:  db,
		},
	)
	wSrv := workers.New(workers.WorkerServiceConfig{
		Log: logger,
	})
	go func() {
		wSrv.StatsPrinter(ctx)
	}()

	// -------------------------------------------------------------------
	routerCfg := rest.RouterConfig{
		Log:             logger,
		ScheduledJobSrv: jobSrv,
		WorkerSrv:       wSrv,
	}
	router := rest.NewRouter(routerCfg)
	srvConfig := rest.ServerConfig{
		Log:     logger,
		Handler: router,
	}
	// -------------------------------------------------------------------
	rest, err := rest.New(srvConfig)
	if err != nil {
		return err
	}
	return rest.Run(ctx)
}
