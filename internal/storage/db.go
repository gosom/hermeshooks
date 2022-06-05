package storage

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/gosom/hermeshooks/internal/entities"
)

type DbConfig struct {
	DSN          string
	MaxOpenConns int
	Debug        bool
}

type DB struct {
	*bun.DB
	sqldb *sql.DB
}

func New(cfg DbConfig) (*DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DSN)))
	sqldb.SetMaxIdleConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxOpenConns)
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))
	ans := DB{
		DB:    db,
		sqldb: sqldb,
	}
	return &ans, nil
}

func (o *DB) Close() error {
	if err := o.sqldb.Close(); err != nil {
		return err
	}
	return o.Close()
}

func InsertScheduledJob(ctx context.Context, db bun.IDB, job entities.ScheduledJob) (entities.ScheduledJob, error) {
	j := FromScheduledJobEntity(job)
	_, err := db.NewInsert().Model(&j).ExcludeColumn("id").Returning("id").Exec(ctx)
	job = ToScheduledJobEntity(j)
	return job, err
}
