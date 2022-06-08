package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
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
	Num   int    `envconfig:"NUM" default:"1000"`
	Debug bool   `envconfig:"DEBUG" default:"false"`
	DSN   string `envconfig:"DSN" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
}

func run(ctx context.Context, logger zerolog.Logger, cfg config) error {
	db, err := storage.New(storage.DbConfig{
		DSN:          cfg.DSN,
		MaxOpenConns: 4 * runtime.GOMAXPROCS(0),
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
			Name:        randomString(5),
			Description: randomString(10),
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

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}
