package scheduledjobs

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
)

type ServiceConfig struct {
	Log zerolog.Logger
	DB  *storage.DB
}

type Service struct {
	log zerolog.Logger
	db  *storage.DB
}

func New(cfg ServiceConfig) *Service {
	ans := Service{
		log: cfg.Log,
		db:  cfg.DB,
	}
	return &ans
}

func (s *Service) Schedule(ctx context.Context, job entities.ScheduledJob) (entities.ScheduledJob, error) {
	_, err := storage.InsertScheduledJob(ctx, s.db, job)
	return job, err
}
