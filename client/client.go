package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	BaseUrl    string
	HttpClient HTTPClient
	Logf       func(format string, a ...any)
}

type HermesHooksAPI struct {
	baseUrl   string
	netClient HTTPClient
	logf      func(format string, a ...any)
}

func New(cfg Config) (*HermesHooksAPI, error) {
	if len(cfg.BaseUrl) == 0 {
		return nil, errors.New("BaseUrl is mandatory")
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	if cfg.HttpClient == nil {
		cfg.HttpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	ans := HermesHooksAPI{
		baseUrl:   cfg.BaseUrl,
		netClient: cfg.HttpClient,
		logf:      cfg.Logf,
	}
	return &ans, nil
}

func (h *HermesHooksAPI) Register(ctx context.Context, name uuid.UUID) (Worker, error) {
	payload := map[string]string{
		"name": name.String(),
	}
	json_data, err := json.Marshal(payload)
	if err != nil {
		return Worker{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		h.buildUrl("/scheduledJobs"),
		bytes.NewBuffer(json_data),
	)
	if err != nil {
		return Worker{}, err
	}
	resp, err := h.netClient.Do(req)
	if err != nil {
		return Worker{}, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	var w Worker
	if resp.StatusCode != http.StatusCreated {
		var e HttpError
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return w, err
		}
		return w, e
	}
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return w, err
	}
	w.Name = name
	return w, nil
}

func (h *HermesHooksAPI) buildUrl(path string) string {
	return h.baseUrl + path
}
