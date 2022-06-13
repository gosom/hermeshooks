package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/rest"
	"github.com/gosom/hermeshooks/internal/services/auth"
	"github.com/gosom/hermeshooks/internal/services/scheduledjobs"
	"github.com/gosom/hermeshooks/internal/services/workers"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/gosom/hermeshooks/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &cli.App{
		Name:     "hermeshooks",
		HelpName: "scheduling webhooks",
		Commands: []*cli.Command{
			serverTask(ctx),
			workerTask(ctx),
			fixturesTask(ctx),
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

// ============================================================================

type serverConfig struct {
	Addr           string `envconfig:"ADDR" default:"localhost:8000"`
	Debug          bool   `envconfig:"DEBUG" default:"false"`
	DSN            string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
	RapidApiKey    string `envconfig:"RAPID_API_KEY" default:"secret"`
	InternalApiKey string `envconfig:"INTERNAL_API_KEY" default:"secret"`
	Domain         string `envconfig:"DOMAIN" default:""`
}

func serverTask(ctx context.Context) *cli.Command {
	var cfg serverConfig
	if err := envconfig.Process("", &cfg); err != nil {
		panic(err)
	}

	logger := common.NewLogger(cfg.Debug)
	cmd := cli.Command{
		Name:  "server",
		Usage: "starts the webserver",
		Action: func(c *cli.Context) error {
			return runServer(ctx, logger, cfg)
		},
	}
	return &cmd
}

func runServer(ctx context.Context, logger zerolog.Logger, cfg serverConfig) error {
	// ----------------- db stuff --------------------------------------
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
		Debug:        cfg.Debug,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	// -----------------------------------------------------------------
	aSrv, err := auth.New(auth.Config{
		Log:            logger,
		DB:             db,
		RapidApiKey:    cfg.RapidApiKey,
		InternalApiKey: cfg.InternalApiKey,
	})
	if err != nil {
		return err
	}
	wSrv := workers.New(workers.WorkerServiceConfig{
		Log: logger,
		DB:  db,
	})
	go func() {
		wSrv.StartReBalancer(ctx)
	}()
	go func() {
		wSrv.StatsPrinter(ctx)
	}()

	jobSrv := scheduledjobs.New(
		scheduledjobs.ServiceConfig{
			Log:         logger,
			DB:          db,
			Partitioner: wSrv,
		},
	)

	// -------------------------------------------------------------------
	routerCfg := rest.RouterConfig{
		Log:             logger,
		ScheduledJobSrv: jobSrv,
		WorkerSrv:       wSrv,
		AuthSrv:         aSrv,
	}
	router := rest.NewRouter(routerCfg)
	srvConfig := rest.ServerConfig{
		Addr:    cfg.Addr,
		Log:     logger,
		Handler: router,
		Domain:  cfg.Domain,
	}
	// -------------------------------------------------------------------
	rest, err := rest.New(srvConfig)
	if err != nil {
		return err
	}
	return rest.Run(ctx)
}

// ============================================================================

type workerConfig struct {
	Debug          bool   `envconfig:"DEBUG" default:"false"`
	Node           string `envconfig:"NODE" default:"http://localhost:8000"`
	DSN            string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
	InternalApiKey string `envconfig:"INTERNAL_API_KEY" default:"secret"`
}

func workerTask(ctx context.Context) *cli.Command {
	var cfg workerConfig
	if err := envconfig.Process("", &cfg); err != nil {
		panic(err)
	}

	logger := common.NewLogger(cfg.Debug)
	cmd := cli.Command{
		Name:  "worker",
		Usage: "starts a worker",
		Action: func(c *cli.Context) error {
			return runWorker(ctx, logger, cfg)
		},
	}
	return &cmd
}

// ============================================================================

func runWorker(ctx context.Context, logger zerolog.Logger, cfg workerConfig) error {
	// ----------------- db stuff --------------------------------------
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
		PgDriver:     true,
		Debug:        cfg.Debug,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	wc := worker.WorkerConfig{
		Log:    logger,
		Node:   cfg.Node,
		DB:     db,
		ApiKey: cfg.InternalApiKey,
	}

	w, err := worker.NewWorker(wc)
	if err != nil {
		return err
	}
	return w.Start(ctx)
}

// ============================================================================
type fixturesConfig struct {
	Num   int    `envconfig:"NUM" default:"1000"`
	Debug bool   `envconfig:"DEBUG" default:"false"`
	DSN   string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
}

func fixturesTask(ctx context.Context) *cli.Command {
	var cfg fixturesConfig
	if err := envconfig.Process("", &cfg); err != nil {
		panic(err)
	}
	logger := common.NewLogger(cfg.Debug)
	cmd := cli.Command{
		Name:  "fixtures",
		Usage: "adds some fixtures to the database",
		Action: func(c *cli.Context) error {
			return runFixtures(ctx, logger, cfg)
		},
	}
	return &cmd
}

func runFixtures(ctx context.Context, logger zerolog.Logger, cfg fixturesConfig) error {
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
		Debug:        cfg.Debug,
	})
	if err != nil {
		return err
	}
	_ = db
	//defer db.Close()
	jobs := make([]entities.ScheduledJob, 0, cfg.Num)
	now := time.Now().UTC()
	fmt.Println(cfg.Num)
	for i := 0; i < cfg.Num; i++ {
		j := entities.ScheduledJob{
			UID:         uuid.New(),
			Name:        common.RandomString(5),
			Description: common.RandomString(10),
			Url:         "http://localhost:8081/wh",
			Payload:     "{}",
			Signature:   "",
			RunAt:       now, // TODO
			Retries:     0,
			Status:      entities.Scheduled,
			Partition:   0,
			CreatedAt:   time.Now().UTC(),
		}
		jobs = append(jobs, j)
		if len(jobs) > 10000 {
			if _, err = db.NewInsert().Model(&jobs).ExcludeColumn("id").Exec(ctx); err != nil {
				return err
			}
			jobs = jobs[:0]
		}
	}
	if len(jobs) > 0 {
		if _, err = db.NewInsert().Model(&jobs).ExcludeColumn("id").Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
