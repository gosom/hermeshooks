package rest

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"

	"github.com/gosom/hermeshooks/internal/entities"
)

type AuthService interface {
	Signup(ctx context.Context, username string) (string, error)
	AuthMiddleware(next bunrouter.HandlerFunc) bunrouter.HandlerFunc
	InternalApi(next bunrouter.HandlerFunc) bunrouter.HandlerFunc
}

type ScheduledJobService interface {
	Get(ctx context.Context, u entities.User, uid string) (entities.ScheduledJob, []entities.Execution, error)
	Schedule(ctx context.Context, job entities.ScheduledJob) (entities.ScheduledJob, error)
}

type WorkerService interface {
	Register(ctx context.Context, name string) (entities.WorkerMeta, error)
	UnRegister(ctx context.Context, name string) (entities.WorkerMeta, error)
	Health(ctx context.Context, name string) error
	UpSince(ctx context.Context) time.Time
	ActiveWorkers(ctx context.Context) int
	DbOk(ctx context.Context) bool
}

type RouterConfig struct {
	Log             zerolog.Logger
	ScheduledJobSrv ScheduledJobService
	WorkerSrv       WorkerService
	AuthSrv         AuthService
	PublicKey       *ecdsa.PublicKey
}

func NewRouter(cfg RouterConfig) *bunrouter.Router {
	router := bunrouter.New()

	router.WithGroup("/api/v1", func(g *bunrouter.Group) {
		g = g.Use(
			logHandler(cfg.Log),
			errorHandler,
			acceptedContentType("application/json"),
		)

		healthHandler := HealthHandler{
			log:       cfg.Log,
			workerSrv: cfg.WorkerSrv,
		}
		g.GET("/health", healthHandler.Get)

		g.WithGroup("/users", func(group *bunrouter.Group) {
			group = group.Use(cfg.AuthSrv.InternalApi)
			userHandler := UserHandler{
				log: cfg.Log,
				srv: cfg.AuthSrv,
			}
			group.POST("", userHandler.Create)
		})

		g.WithGroup("/workers", func(group *bunrouter.Group) {
			group = group.Use(cfg.AuthSrv.InternalApi)
			workerHandler := WorkerHandler{
				log:       cfg.Log,
				workerSrv: cfg.WorkerSrv,
			}
			group.POST("", workerHandler.Register)
			group.POST("/:name/health", workerHandler.HealthHandler)
			group.DELETE("/:name", workerHandler.UnRegister)
		})

		g.WithGroup("/meta", func(group *bunrouter.Group) {
			metaHandler := MetaHandler{
				log: cfg.Log,
			}
			group.GET("", metaHandler.Get)
		})

		g.WithGroup("/scheduledJobs", func(group *bunrouter.Group) {
			group = group.Use(cfg.AuthSrv.AuthMiddleware)
			scheduledJobsHandler := ScheduledJobsHandler{
				log: cfg.Log,
				srv: cfg.ScheduledJobSrv,
			}
			group.GET("/:uuid", scheduledJobsHandler.Get)
			group.POST("", scheduledJobsHandler.Create)
		})

	})
	return router
}
