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

type IDB = bun.IDB

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

func InsertScheduledJob(ctx context.Context, db IDB, job entities.ScheduledJob) (entities.ScheduledJob, error) {
	j := FromScheduledJobEntity(job)
	_, err := db.NewInsert().Model(&j).ExcludeColumn("id").Returning("id").Exec(ctx)
	job = ToScheduledJobEntity(j)
	return job, err
}

func UpdateScheduledJobsPartitions(ctx context.Context, db IDB, job entities.ScheduledJob) error {
	j := FromScheduledJobEntity(job)
	_, err := db.NewUpdate().Model(&j).Column("partition").
		Where("id = ?", j.ID).Exec(ctx)
	return err
}

func UpdateJobsPartitions(ctx context.Context, db IDB, current int, target int) error {
	_, err := db.NewUpdate().
		Table("scheduled_jobs").Set("partition = ?", target).
		Where("status IN (?)", bun.In([]entities.ScheduledJobStatus{entities.Scheduled, entities.Pending})).
		Where("partition = ?", current).
		Exec(ctx)
	return err
}

func UpdateOrhanJobs(ctx context.Context, db IDB, existing []int, target int) error {
	_, err := db.NewUpdate().
		Table("scheduled_jobs").Set("partition = ?", target).
		Where("status IN (?)", bun.In([]entities.ScheduledJobStatus{entities.Scheduled, entities.Pending})).
		Where("partition NOT IN (?)", bun.In(existing)).
		Exec(ctx)
	return err
}
