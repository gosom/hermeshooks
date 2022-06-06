package workers

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/rs/zerolog"
)

type WorkerServiceConfig struct {
	Log zerolog.Logger
	DB  *storage.DB
}

type workerService struct {
	log      zerolog.Logger
	db       *storage.DB
	lock     sync.RWMutex
	pnum     int
	registry map[string]entities.WorkerMeta
}

func New(cfg WorkerServiceConfig) *workerService {
	rand.Seed(time.Now().UnixNano())
	ans := workerService{
		log:      cfg.Log,
		db:       cfg.DB,
		registry: make(map[string]entities.WorkerMeta),
		pnum:     0,
	}
	return &ans
}

func (o *workerService) RLock() {
	o.lock.RLock()
}

func (o *workerService) RUnlock() {
	o.lock.RUnlock()
}

// Pick returns a random partition. Makes sure a read lock is acquired
func (o *workerService) Pick() int {
	if len(o.registry) == 0 {
		return 0
	}
	r := rand.Intn(len(o.registry))
	for _, v := range o.registry {
		if r == 0 {
			return v.Partition
		}
		r--
	}
	return 0
}

func (o *workerService) StatsPrinter(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
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

func (o *workerService) workerCleanUp(w entities.WorkerMeta) error {
	// update db to reassign
	// clean up
	o.lock.Lock()
	delete(o.registry, w.Name)
	o.lock.Unlock()
	// maybe it's good to balance assignments among workers
	// for now just pick one
	o.lock.RLock()
	defer o.lock.RUnlock()
	return storage.UpdateJobsPartitions(context.Background(), o.db, w.Partition, o.Pick())
}

func (o *workerService) getCurrentPartitions() []int {
	items := make([]int, 0, len(o.registry))
	for _, v := range o.registry {
		items = append(items, v.Partition)
	}
	return items
}

func (o *workerService) StartWorkerMonitor(ctx context.Context, w entities.WorkerMeta) {
	defaultWaitDuration := 10 * time.Second
	ticker := time.NewTicker(defaultWaitDuration)
	o.log.Info().Msgf("next tick in %s", defaultWaitDuration)
	defer func() {
		o.log.Info().Msgf("worker %s has been de-registered", w.Name)
	}()
	defer ticker.Stop()
	defer o.workerCleanUp(w)
	current := o.getCurrentPartitions()
	if err := storage.UpdateOrhanJobs(ctx, o.db, current, w.Partition); err != nil {
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.log.Info().Msgf("tick")
				_, ok := o.workerHealthCheck(w.Name)
				if !ok {
					return
				}
			}
		}
	}()
	<-done
}

func (o *workerService) workerHealthCheck(name string) (entities.WorkerMeta, bool) {
	o.log.Info().Msgf("healthcheck for %s", name)
	o.lock.RLock()
	defer o.lock.RUnlock()
	w, ok := o.registry[name]
	if !ok {
		o.log.Info().Msgf("worker %s is not registered", w.Name)
		return w, false
	}
	if time.Now().UTC().Sub(w.LastHealthCheck) <= 20*time.Second {
		o.log.Info().Msgf("worker %s is healthy", w.Name)
		return w, true
	}
	return w, false
}

func (o *workerService) Register(ctx context.Context, name string) (entities.WorkerMeta, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	w, ok := o.registry[name]
	if !ok {
		o.pnum++
		w.Name = name
		w.Partition = o.pnum
	} else {
		w.CancelFunc() // we cancel the context of the running worker
	}
	var workerCtx context.Context
	workerCtx, w.CancelFunc = context.WithCancel(context.Background())
	w.LastHealthCheck = time.Now().UTC()
	o.registry[name] = w
	go o.StartWorkerMonitor(workerCtx, w)
	o.log.Info().Msgf("registered worker %s", w.Name)
	return w, nil
}

func (o *workerService) UnRegister(ctx context.Context, name string) (entities.WorkerMeta, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	w, ok := o.registry[name]
	if !ok {
		return w, nil
	}
	w.CancelFunc()
	return w, nil
}
