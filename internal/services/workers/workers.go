package workers

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/rs/zerolog"
)

type WorkerServiceConfig struct {
	Log        zerolog.Logger
	DB         *storage.DB
	HealthFreq time.Duration
}

type workerService struct {
	log        zerolog.Logger
	db         *storage.DB
	lock       sync.RWMutex
	pnum       int
	registry   map[string]entities.WorkerMeta
	ch         chan struct{}
	healthFreq time.Duration
}

func New(cfg WorkerServiceConfig) *workerService {
	rand.Seed(time.Now().UnixNano())
	if cfg.HealthFreq == 0 {
		cfg.HealthFreq = 10 * time.Second
	}
	ans := workerService{
		log:        cfg.Log,
		db:         cfg.DB,
		registry:   make(map[string]entities.WorkerMeta),
		pnum:       0,
		ch:         make(chan struct{}, 1),
		healthFreq: cfg.HealthFreq,
	}
	return &ans
}

func (o *workerService) RLock() {
	o.lock.RLock()
}

func (o *workerService) RUnlock() {
	o.lock.RUnlock()
}

func (o *workerService) StartReBalancer(ctx context.Context) {
	if err := o.rebalance(ctx); err != nil {
		panic(err)
	}
	for _ = range o.ch {
		if err := o.rebalance(ctx); err != nil {
			panic(err)
		}
	}
}

func (o *workerService) triggerRebalance() bool {
	select {
	case o.ch <- struct{}{}:
		return true
	default:
	}
	return false
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
	ticker := time.NewTicker(5 * time.Minute)
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
	o.triggerRebalance()
	return nil
}

func (o *workerService) getCurrentPartitions() []int {
	items := make([]int, 0, len(o.registry))
	for _, v := range o.registry {
		items = append(items, v.Partition)
	}
	return items
}

// rebalance happens AFTER registration/unregistration
func (o *workerService) rebalance(ctx context.Context) error {
	// We should get all the jobs that have a partition not in the worker list
	// and allocate them to workers
	o.lock.RLock()
	t0 := time.Now()
	o.log.Info().Msgf("Starting rebalance")
	defer func() {
		o.log.Info().Msgf("rebalance time %s\n", time.Now().Sub(t0))
	}()
	active := o.getCurrentPartitions()
	o.lock.RUnlock()
	tx, err := o.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := storage.ReBalance(ctx, tx, active); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (o *workerService) StartWorkerMonitor(ctx context.Context, w entities.WorkerMeta) {
	ticker := time.NewTicker(o.healthFreq)
	o.log.Info().Msgf("next tick in %s", o.healthFreq)
	defer func() {
		o.log.Info().Msgf("worker %s has been de-registered", w.Name)
	}()
	defer ticker.Stop()
	defer o.workerCleanUp(w)
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
	if time.Now().UTC().Sub(w.LastHealthCheck) <= 2*o.healthFreq {
		o.log.Info().Msgf("worker %s is healthy", w.Name)
		return w, true
	}
	return w, false
}

func (o *workerService) Register(ctx context.Context, name string) (entities.WorkerMeta, error) {
	o.lock.Lock()
	select {
	case <-ctx.Done():
		return entities.WorkerMeta{}, ctx.Err()
	default:
	}
	w, ok := o.registry[name]
	if ok {
		w.LastHealthCheck = time.Now().UTC()
		o.registry[name] = w
		o.lock.Unlock()
		return w, nil
	}
	o.pnum++
	w.Name = name
	w.Partition = o.pnum
	var workerCtx context.Context
	workerCtx, w.CancelFunc = context.WithCancel(context.Background())
	w.LastHealthCheck = time.Now().UTC()
	o.registry[name] = w
	o.lock.Unlock()

	o.triggerRebalance()
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

func (o *workerService) Health(ctx context.Context, name string) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	w, ok := o.registry[name]
	if !ok {
		return errors.New("resource not found")
	}
	w.LastHealthCheck = time.Now().UTC()
	o.registry[name] = w
	return nil
}
