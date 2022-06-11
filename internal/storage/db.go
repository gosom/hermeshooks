package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
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
	PgDriver     bool
}

type DB struct {
	*bun.DB
	sqldb *sql.DB
}

func New(cfg DbConfig) (*DB, error) {

	var sqldb *sql.DB
	if cfg.PgDriver {
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DSN)))
	} else {
		config, err := pgx.ParseConfig(cfg.DSN)
		if err != nil {
			return nil, err
		}
		config.PreferSimpleProtocol = true
		sqldb = stdlib.OpenDB(*config)
	}

	sqldb.SetMaxIdleConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxOpenConns)

	db := bun.NewDB(sqldb, pgdialect.New())
	if cfg.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}
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

func (o *DB) Listen(ctx context.Context, outc chan<- struct{}, p int) error {
	ln := pgdriver.NewListener(o.DB)
	if err := ln.Listen(ctx, "jobs:rebalance"); err != nil {
		return err
	}
	for notif := range ln.Channel() {
		tmp := map[string]int{}
		if err := json.Unmarshal([]byte(notif.Payload), &tmp); err != nil {
			return err
		}
		if tmp["partition"] == p {
			select {
			case outc <- struct{}{}:
			default:
			}
		}
	}
	return nil
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

func SelectJobsForExecution(ctx context.Context, db *DB, partition int, limit int, now time.Time) ([]entities.ScheduledJob, entities.ScheduledJob, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, entities.ScheduledJob{}, err
	}
	defer tx.Rollback()
	var current []ScheduledJob
	if err := tx.NewSelect().
		Model(&current).
		Where("status = ?", entities.Scheduled).
		Where("partition = ?", partition).
		Where("run_at <= ?", now).
		Order("run_at").
		Limit(limit).
		For("update").
		Scan(ctx); err != nil {
		return nil, entities.ScheduledJob{}, err
	}
	toUpdate := make([]int64, len(current), len(current))
	for i := range current {
		toUpdate[i] = current[i].ID
	}
	if len(toUpdate) > 0 {
		if _, err := tx.NewUpdate().
			Table("scheduled_jobs").
			Set("status = ?", entities.Pending).
			Where("id IN (?)", bun.In(toUpdate)).
			Exec(ctx); err != nil {
			return nil, entities.ScheduledJob{}, err
		}
	}
	var nexts []ScheduledJob
	afterTime := now
	if len(current) > 0 {
		afterTime = current[len(current)-1].RunAt
	}
	if err := tx.NewSelect().
		Model(&nexts).
		Where("status = ?", entities.Scheduled).
		Where("partition = ?", partition).
		Where("run_at >= ?", afterTime).
		Order("run_at").
		Limit(1).
		Scan(ctx); err != nil {
		return nil, entities.ScheduledJob{}, err
	}
	var next entities.ScheduledJob
	if len(nexts) > 0 {
		next = ToScheduledJobEntity(nexts[0])
	}
	items := make([]entities.ScheduledJob, len(current), len(current))
	for i := range current {
		items[i] = ToScheduledJobEntity(current[i])
		items[i].Status = entities.Pending
	}
	return items, next, tx.Commit()
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
	q := db.NewUpdate().
		Table("scheduled_jobs").
		Set("partition = ?", target).
		Set("status = ?", entities.Scheduled)
	if len(active) > 0 {
		q = q.Where("partition NOT IN (?)", bun.In(active))
	}
	q = q.Where("status IN (?)", bun.In([]entities.ScheduledJobStatus{entities.Scheduled, entities.Pending}))

	_, err := q.Exec(ctx)
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
		Where("status IN (?)", bun.In([]entities.ScheduledJobStatus{entities.Scheduled, entities.Pending})).
		Group("t1.partition").
		Scan(ctx)

	if err != nil {
		return err
	}

	isActive := func(p int) bool {
		for _, v := range active {
			if p == v {
				return true
			}
		}
		return false
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
		if isActive(grp.Partition) {
			grp.ToAdd = perBucket - grp.Count
		} else {
			grp.ToAdd = -grp.Count
		}
		if grp.ToAdd < 0 {
			if err := RemovesJobsFromPartition(ctx, db, grp.Partition, -grp.ToAdd); err != nil {
				return err
			}
		}
		m[grp.Partition] = grp
	}
	var last group
	for _, grp := range m {
		if grp.Partition > 0 && grp.ToAdd > 0 && isActive(grp.Partition) {
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

func UpdateJobStatus(ctx context.Context, db IDB, job entities.ScheduledJob) error {
	j := FromScheduledJobEntity(job)
	_, err := db.NewUpdate().
		Model(&j).
		Column("status").
		Column("updated_at").
		Where("id = ?", j.ID).
		Exec(ctx)
	return err
}

func InsertExecution(ctx context.Context, db IDB, job entities.Execution) (entities.Execution, error) {
	e := FromEntitiesExecution(job)
	if _, err := db.NewInsert().
		Model(&e).
		ExcludeColumn("id").
		Returning("id").
		Exec(ctx); err != nil {
		return entities.Execution{}, err
	}
	ans := ToEntitiesExecution(e)
	return ans, nil

}

func InsertUser(ctx context.Context, db IDB, u entities.User) (entities.User, error) {
	su := FromEntitiesUser(u)
	if _, err := db.NewInsert().
		Model(&su).
		ExcludeColumn("id").
		Returning("id").
		Exec(ctx); err != nil {
		return entities.User{}, err
	}
	u = ToEntitiesUser(su)
	return u, nil
}

func UserExists(ctx context.Context, db IDB, username string) (bool, error) {
	exists, err := db.NewSelect().
		Model((*User)(nil)).Where("username = ?", username).
		Exists(ctx)
	return exists, err
}

func GetUserByUserName(ctx context.Context, db IDB, username string) (entities.User, error) {
	var u User
	if err := db.NewSelect().
		Model(&u).
		Where("username = ?", username).
		Scan(ctx); err != nil {
		return entities.User{}, err
	}
	return ToEntitiesUser(u), nil
}
