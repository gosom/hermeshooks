package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
)

type executor struct {
	log     zerolog.Logger
	db      *storage.DB
	iq      <-chan entities.ScheduledJob
	threads int
	client  common.HTTPClient
}

func (e executor) start(ctx context.Context) error {
	e.log.Info().Msg("starting executor")
	sem := make(chan bool, e.threads)
	for j := range e.iq {
		sem <- true
		go func(job entities.ScheduledJob) {
			defer func() {
				<-sem
			}()
			if err := e.process(ctx, job); err != nil {
				e.log.Error().Err(err)
			} else {
				e.log.Info().Int64("jobId", job.ID).Msg("executor success")
			}
		}(j)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
	return nil
}

func (e executor) process(ctx context.Context, job entities.ScheduledJob) error {
	statusCode, err := func() (int, error) {
		req, err := e.prepareReq(ctx, job)
		if err != nil {
			return 0, fmt.Errorf("fail to prepare req error: %w", err)
		}
		e.log.Info().Msgf("prepared req for job %d", job.ID)
		resp, err := common.RetryDo(e.client, req, job.Retries)
		if err != nil {
			return 0, fmt.Errorf("request fail with error: %w", err)
		}
		if resp == nil {
			return 0, fmt.Errorf("resp is nil")
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp.StatusCode, nil
	}()

	e.log.Info().Int64("jobId", job.ID).Err(err).Msg("process job")
	_ = statusCode

	job.UpdatedAt = time.Now().UTC()
	var msg string
	if err != nil {
		job.Status = entities.Success
		msg = err.Error()
	} else {
		job.Status = entities.Fail
	}
	execution := entities.Execution{
		ScheduledJobID: job.ID,
		StatusCode:     statusCode,
		Msg:            msg,
		CreatedAt:      job.UpdatedAt,
	}
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := storage.UpdateJobStatus(ctx, tx, job); err != nil {
		return err
	}
	if _, err := storage.InsertExecution(ctx, tx, execution); err != nil {
		return err
	}
	return tx.Commit()
}

func (e executor) prepareReq(ctx context.Context, job entities.ScheduledJob) (*http.Request, error) {
	var body io.Reader
	if len(job.Payload) > 0 {
		body = bytes.NewReader([]byte(job.Payload))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.Url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", job.ContentType)
	req.Header.Set("X-HERMESHOOKS-PAYLOAD-SIG", job.Signature)
	// TODO compute our sig using our PrivateKey
	signature := ""
	req.Header.Set("X-HERMESHOOKS-SIG", signature)
	return req, nil
}
