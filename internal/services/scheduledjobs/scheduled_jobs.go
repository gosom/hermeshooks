package scheduledjobs

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/gosom/hermeshooks/internal/entities"
)

type ServiceConfig struct {
	Log zerolog.Logger
	DB  *bun.DB
}

type Service struct {
	log zerolog.Logger
}

func New(cfg ServiceConfig) *Service {
	ans := Service{
		log: cfg.Log,
	}
	return &ans
}

func (s *Service) Schedule(ctx context.Context, job entities.ScheduledJob) error {
	// Here we should assing to a partition and save to db

	return nil
}
