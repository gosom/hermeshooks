package worker

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
)

type monitor struct {
	log zerolog.Logger
	iq  <-chan struct{}
	p   int
	db  *storage.DB
}

func (m monitor) start(ctx context.Context) (<-chan entities.ScheduledJob, <-chan error) {
	buffSize := 100
	outc := make(chan entities.ScheduledJob, buffSize)
	errc := make(chan error, 1)
	go func() {
		m.log.Info().Msgf("starting monitor")
		defer func() {
			m.log.Warn().Msgf("exiting monitor")
		}()
		defer close(errc)
		defer close(outc)
		defaultWaitDuration := 5 * time.Minute
		timer := time.NewTimer(defaultWaitDuration)
		defer timer.Stop()
		for {
			m.log.Info().Msgf("monito waiting stop")
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			now := time.Now().UTC()
			jobs, next, err := storage.SelectJobsForExecution(
				ctx, m.db, m.p, buffSize, now,
			)
			if err != nil {
				errc <- err
				return
			}
			m.log.Info().Int("jobsCount", len(jobs)).Msg("monitor selected jobs")
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer func() {
					m.log.Info().Int("jobsCount", len(jobs)).Msg("jobs pushed")
				}()
				defer wg.Done()
				for i := range jobs {
					outc <- jobs[i]
				}
			}()
			m.log.Info().Str("monitor", "yes").Msgf("next job %+b", next)
			waitTime := defaultWaitDuration
			if !next.RunAt.IsZero() {
				switch wt := next.RunAt.Sub(now); {
				case wt > 0:
					waitTime = wt
				default:
					waitTime = time.Nanosecond
				}
			}
			timer.Reset(waitTime)
			m.log.Info().Dur("waitTime", waitTime).Msg("monitor waits")
			// TODO select jobs from partition
			// sent the ones in the past for execution
			// wait until next jobs triggers or until a refresh event
			select {
			case <-timer.C:
			case <-m.iq:
				m.log.Info().Msg("monitor received refresh commands")
			case <-ctx.Done():
				m.log.Info().Msg("monitor ctx is done, wait push")
				wg.Wait()
				return
			}
			m.log.Info().Msgf("monitor timer triggered")
		}
	}()
	return outc, errc
}
