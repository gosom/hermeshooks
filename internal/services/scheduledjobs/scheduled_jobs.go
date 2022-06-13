package scheduledjobs

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
)

type Partitioner interface {
	RLock()
	Pick() int
	RUnlock()
}

type ServiceConfig struct {
	Log         zerolog.Logger
	DB          *storage.DB
	Partitioner Partitioner
}

type Service struct {
	log         zerolog.Logger
	db          *storage.DB
	partitioner Partitioner
}

func New(cfg ServiceConfig) *Service {
	ans := Service{
		log:         cfg.Log,
		db:          cfg.DB,
		partitioner: cfg.Partitioner,
	}
	return &ans
}

func (s *Service) Schedule(ctx context.Context, job entities.ScheduledJob) (entities.ScheduledJob, error) {
	s.partitioner.RLock()
	defer s.partitioner.RUnlock()
	job.Partition = s.partitioner.Pick()
	return storage.InsertScheduledJob(ctx, s.db, job)
}
func (s *Service) Get(ctx context.Context, u entities.User, uid string) (entities.ScheduledJob, []entities.Execution, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return entities.ScheduledJob{}, nil, err
	}
	job, err := storage.GetScheduledJob(ctx, tx, uid, u.ID)
	if err != nil {
		return entities.ScheduledJob{}, nil, err
	}
	defer tx.Rollback()
	executions, err := storage.SelectExecutions(ctx, tx, job.ID)
	if err != nil {
		return job, nil, err
	}
	return job, executions, tx.Commit()
}
