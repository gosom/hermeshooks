package rest

import (
	"context"
	"crypto/ecdsa"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type ScheduledJobService interface {
	Schedule(ctx context.Context, job entities.ScheduledJob) (entities.ScheduledJob, error)
}

type WorkerService interface {
	Register(ctx context.Context, name string) (entities.WorkerMeta, error)
	UnRegister(ctx context.Context, name string) (entities.WorkerMeta, error)
}

type RouterConfig struct {
	Log             zerolog.Logger
	ScheduledJobSrv ScheduledJobService
	WorkerSrv       WorkerService
	PublicKey       *ecdsa.PublicKey
}

func NewRouter(cfg RouterConfig) *bunrouter.Router {
	router := bunrouter.New()

	router.WithGroup("/api/v1", func(g *bunrouter.Group) {
		g = g.Use(logHandler(cfg.Log), errorHandler)

		g.WithGroup("/workers", func(group *bunrouter.Group) {
			workerHandler := WorkerHandler{
				log:       cfg.Log,
				workerSrv: cfg.WorkerSrv,
			}
			group.POST("", workerHandler.Register)
			group.DELETE("/:name", workerHandler.UnRegister)
		})

		g.WithGroup("/meta", func(group *bunrouter.Group) {
			metaHandler := MetaHandler{
				log: cfg.Log,
			}
			group.GET("", metaHandler.Get)
		})

		g.WithGroup("/scheduledJobs", func(group *bunrouter.Group) {
			scheduledJobsHandler := ScheduledJobsHandler{
				log: cfg.Log,
				srv: cfg.ScheduledJobSrv,
			}
			group.POST("", scheduledJobsHandler.Create)
		})

	})
	return router
}
