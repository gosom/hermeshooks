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
