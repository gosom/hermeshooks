package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/rs/zerolog"
)

type WorkerConfig struct {
	Log         zerolog.Logger
	Node        string
	NetClient   common.HTTPClient
	DB          *storage.DB
	Concurrency int
	ApiKey      string
}

type worker struct {
	name        string
	node        string
	log         zerolog.Logger
	netClient   common.HTTPClient
	db          *storage.DB
	concurrency int
	apiKey      string
}

func NewWorker(cfg WorkerConfig) (*worker, error) {
	if cfg.DB == nil {
		return nil, errors.New("db is missing")
	}
	if len(cfg.Node) == 0 {
		return nil, errors.New("node is missing")
	}
	if cfg.NetClient == nil {
		cfg.NetClient = &http.Client{
			Timeout: time.Second * 5,
		}
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}
	ans := worker{
		name:        uuid.New().String(),
		log:         cfg.Log,
		node:        cfg.Node,
		netClient:   cfg.NetClient,
		db:          cfg.DB,
		concurrency: cfg.Concurrency,
		apiKey:      cfg.ApiKey,
	}
	return &ans, nil
}

func (w *worker) Start(ctx context.Context) error {
	partition, err := w.register(ctx)
	if err != nil {
		return err
	}
	defer w.unregister(ctx)

	w.log.Info().Int("partition", partition).Msgf("worker registered")

	refreshc, errc1 := w.listen(ctx, partition)

	m := monitor{
		log: w.log,
		iq:  refreshc,
		p:   partition,
		db:  w.db,
	}
	jobsc, errc2 := m.start(ctx)

	ex := executor{
		log:     w.log,
		db:      w.db,
		iq:      jobsc,
		threads: w.concurrency,
		client:  w.netClient,
	}

	errc3 := func() <-chan error {
		errc := make(chan error, 1)
		go func() {
			defer func() {
				w.log.Info().Msg("exiting executor")
			}()
			defer close(errc)
			if err := ex.start(ctx); err != nil {
				errc <- err
				return
			}
		}()
		return errc
	}()

	errc4 := func() <-chan error {
		errc := make(chan error, 1)
		go func() {
			defer close(errc)
			if err := w.ping(ctx); err != nil {
				errc <- err
				return
			}
			return
		}()
		return errc
	}()

	select {
	case err := <-errc1:
		return err
	case err := <-errc2:
		return err
	case err := <-errc3:
		return err
	case err := <-errc4:
		return err
	}
	return nil
}

func (w *worker) register(ctx context.Context) (int, error) {
	u := w.node + "/api/v1/workers"
	v := map[string]string{
		"name": w.name,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&v); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, u, &buf,
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hermeshook worker")
	req.Header.Set("X-API-KEY", w.apiKey)
	registerResp := struct {
		Partition int `json:"partition"`
	}{}
	if err := w.doReq(req, &registerResp); err != nil {
		return 0, err
	}
	return registerResp.Partition, nil
}

func (w *worker) unregister(ctx context.Context) error {
	u := w.node + fmt.Sprintf("/api/v1/workers/%s", w.name)
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, u, nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hermeshook worker")
	req.Header.Set("X-API-KEY", w.apiKey)
	return w.doReq(req, nil)
}

func (w *worker) doReq(req *http.Request, v any) error {
	resp, err := common.RetryDo(w.netClient, req, 3)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("doReq - invalid status code %d", resp.StatusCode)
	}
	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

func (w *worker) ping(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.pingReq(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *worker) pingReq(ctx context.Context) error {
	u := w.node + fmt.Sprintf("/api/v1/workers/%s/health", w.name)
	fmt.Println(u)
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, u, nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hermeshook worker")
	req.Header.Set("X-API-KEY", w.apiKey)

	return w.doReq(req, nil)
}

func (w *worker) listen(ctx context.Context, partition int) (<-chan struct{}, <-chan error) {
	outc := make(chan struct{}, 1)
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		defer close(outc)
		if err := w.db.Listen(ctx, outc, partition); err != nil {
			errc <- err
			return
		}
	}()
	return outc, errc
}
