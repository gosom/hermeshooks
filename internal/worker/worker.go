package worker

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/hermeshooks/internal/common"
	"github.com/rs/zerolog"
)

type WorkerConfig struct {
	Log       zerolog.Logger
	Node      string
	NetClient common.HTTPClient
}

type worker struct {
	name      string
	node      string
	log       zerolog.Logger
	netClient common.HTTPClient
}

func NewWorker(cfg WorkerConfig) (*worker, error) {
	if len(cfg.Node) == 0 {
		return nil, errors.New("node is missing")
	}
	if cfg.NetClient == nil {
		cfg.NetClient = &http.Client{
			Timeout: time.Second * 5,
		}
	}
	ans := worker{
		name:      uuid.New().String(),
		log:       cfg.Log,
		node:      cfg.Node,
		netClient: cfg.NetClient,
	}
	return &ans, nil
}

func (w *worker) Start(ctx context.Context) error {
	// Here we need to register first and then we should ping
	// the node every 5 seconds
	// when we exist we should try to deregister
	return nil
}

func (w *worker) register(ctx context.Context) error {
	return nil
}
