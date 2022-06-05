package workers

import (
	"context"
	"sync"
	"time"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/rs/zerolog"
)

type WorkerServiceConfig struct {
	Log zerolog.Logger
}

type workerService struct {
	log      zerolog.Logger
	lock     sync.RWMutex
	pnum     int
	registry map[string]entities.WorkerMeta
}

func New(cfg WorkerServiceConfig) *workerService {
	ans := workerService{
		log:      cfg.Log,
		registry: make(map[string]entities.WorkerMeta),
		pnum:     0,
	}
	return &ans
}

func (o *workerService) StatsPrinter(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.lock.RLock()
				wn := len(o.registry)
				o.lock.RUnlock()
				o.log.Info().Int("workersNum", wn).Msg("worker stats")
			}
		}
	}()
	<-done
	return nil
}

func (o *workerService) Register(ctx context.Context, name string) (entities.WorkerMeta, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	w, ok := o.registry[name]
	if !ok {
		o.pnum++
		w.Name = name
		w.Partition = o.pnum
	}
	w.LastHealthCheck = time.Now().UTC()
	w.Healthy = true
	o.registry[name] = w
	return w, nil
}

func (o *workerService) Unregister(ctx context.Context, name string) (entities.WorkerMeta, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	w, ok := o.registry[name]
	if !ok {
		return w, nil
	}
	// TODO reassign jobs from partition to other workers partitions
	return w, nil
}
