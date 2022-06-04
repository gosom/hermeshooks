package main

import (
	"context"
	"database/sql"
	"os"
	"runtime"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/gosom/hermeshooks/internal/rest"
	"github.com/gosom/hermeshooks/internal/services/scheduledjobs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		panic(err)
	}

	logger := newLogger()

	if err := run(ctx, logger, cfg); err != nil {
		logger.Panic().AnErr("error", err).Msg("exiting with error")
	}
}

type config struct {
	Debug bool   `envconfig:"DEBUG" default:"false"`
	DSN   string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
}

func newLogger() zerolog.Logger {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return logger
}

func run(ctx context.Context, logger zerolog.Logger, cfg config) error {
	// ----------------- db stuff --------------------------------------
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DSN)))

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxIdleConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	db := bun.NewDB(sqldb, pgdialect.New())
	defer func() {
		db.Close()
		sqldb.Close()
	}()

	// -----------------------------------------------------------------
	jobSrv := scheduledjobs.New(
		scheduledjobs.ServiceConfig{
			Log: logger,
			DB:  db,
		},
	)

	// -------------------------------------------------------------------
	routerCfg := rest.RouterConfig{
		Log:             logger,
		ScheduledJobSrv: jobSrv,
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
