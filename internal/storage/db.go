package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
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
	config, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	config.PreferSimpleProtocol = true

	sqldb := stdlib.OpenDB(*config)

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
	return nil // TODO How do I close bun?
}

func Notify(ctx context.Context, db IDB, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	q := `NOTIFY "jobs:rebalance", ?`
	_, err = db.ExecContext(ctx, q, string(b))
	return err
}

func InsertScheduledJob(ctx context.Context, db IDB, job entities.ScheduledJob) (entities.ScheduledJob, error) {
	j := FromScheduledJobEntity(job)
	_, err := db.NewInsert().Model(&j).ExcludeColumn("id").Returning("id").Exec(ctx)
	if err != nil {
		return job, err
	}
	if err := Notify(ctx, db, map[string]int{
		"partition": job.Partition,
	}); err != nil {
		return job, err
	}
	job = ToScheduledJobEntity(j)
	return job, err
}

func UpdateScheduledJobsPartitions(ctx context.Context, db IDB, job entities.ScheduledJob) error {
	j := FromScheduledJobEntity(job)
	_, err := db.NewUpdate().Model(&j).Column("partition").
		Where("id = ?", j.ID).Exec(ctx)
	return err
}

func UpdateJobsPartitions(ctx context.Context, db IDB, active []int, target int) error {
	exclude := make([]int, len(active)+1)
	exclude[0] = 0
	for i := range active {
		exclude[i+1] = active[i]
	}
	_, err := db.NewUpdate().
		Table("scheduled_jobs").
		Set("partition = ?", target).
		Where("partition NOT IN (?)", bun.In(exclude)).
		Where("status IN (?)", bun.In([]entities.ScheduledJobStatus{entities.Scheduled, entities.Pending})).
		Exec(ctx)
	return err
}

func AssignJobsToPartition(ctx context.Context, db IDB, target int, limit int) error {
	subq := db.NewSelect().
		Table("scheduled_jobs").
		Column("id").
		Where("partition = 0")
	if limit > 0 {
		subq = subq.Limit(limit)
	}

	_, err := db.NewUpdate().
		With("_data", subq).
		Table("scheduled_jobs").
		TableExpr("_data").
		Set("partition = ?", target).
		Where("scheduled_jobs.id = _data.id").
		Exec(ctx)
	return err
}

func RemovesJobsFromPartition(ctx context.Context, db IDB, current int, limit int) error {
	subq := db.NewSelect().
		Table("scheduled_jobs").
		Column("id").
		Where("partition = ?", current).
		Limit(limit)

	_, err := db.NewUpdate().
		With("_data", subq).
		Table("scheduled_jobs").
		TableExpr("_data").
		Set("partition = 0").
		Where("scheduled_jobs.id = _data.id").
		Exec(ctx)
	return err
}

func ReBalance(ctx context.Context, db IDB, active []int) error {
	//db.ExecContext(ctx, "LOCK TABLE games IN ACCESS EXCLUSIVE MODE")
	// first we put the orphan jobs to partition 0
	if err := UpdateJobsPartitions(ctx, db, active, 0); err != nil {
		return err
	}
	if len(active) == 0 {
		return nil
	}
	type group struct {
		Partition int
		Count     int
		ToAdd     int `bun:"-"`
	}
	var grps []group
	err := db.NewSelect().
		Model(&grps).
		ModelTableExpr("scheduled_jobs AS t1").
		Column("t1.partition").
		ColumnExpr("Count(t1.id) as count").
		Group("t1.partition").
		Scan(ctx)

	if err != nil {
		return err
	}

	var total int
	m := make(map[int]group)
	for i := range grps {
		total += grps[i].Count
		m[grps[i].Partition] = grps[i]
	}
	for _, partition := range active {
		if _, ok := m[partition]; !ok {
			m[partition] = group{Partition: partition}
		}
	}
	perBucket := total / len(active)
	for _, grp := range m {
		if grp.Partition == 0 {
			continue
		}
		grp.ToAdd = perBucket - grp.Count
		if grp.ToAdd < 0 {
			if err := RemovesJobsFromPartition(ctx, db, grp.Partition, -grp.ToAdd); err != nil {
				return err
			}
		}
		m[grp.Partition] = grp
	}
	var last group
	for _, grp := range m {
		if grp.Partition > 0 && grp.ToAdd > 0 {
			limit := grp.ToAdd
			if err := AssignJobsToPartition(ctx, db, grp.Partition, limit); err != nil {
				return err
			}
			last = grp
		}
	}
	if err := AssignJobsToPartition(ctx, db, last.Partition, 0); err != nil {
		return err
	}
	for _, v := range active {
		if err := Notify(ctx, db, map[string]int{"partition": v}); err != nil {
			return err
		}
	}
	return nil
}
