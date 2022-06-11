package rest

import (
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type HealthHandler struct {
	workerSrv    WorkerService
	log          zerolog.Logger
	lock         sync.RWMutex
	lastCache    time.Time
	lastResponse HealthResponse
}

type HealthResponse struct {
	ServerUpSince  time.Time `json:"serverUpSince"`
	WorkersHealthy bool      `json:"workersHealthy"`
	DbHealthy      bool      `json:"dbHealthy"`
}

func (h *HealthHandler) Get(w http.ResponseWriter, r bunrouter.Request) error {
	h.lock.RLock()
	elapsed := time.Now().UTC().Sub(h.lastCache)
	if elapsed <= 1*time.Minute {
		ans := h.lastResponse
		h.lock.RUnlock()
		return JSON(w, http.StatusOK, ans)
	}
	h.lock.RUnlock()
	wcount := h.workerSrv.ActiveWorkers(r.Context())
	var wok bool
	if wcount > 0 {
		wok = true
	}
	ans := HealthResponse{
		ServerUpSince:  h.workerSrv.UpSince(r.Context()),
		WorkersHealthy: wok,
		DbHealthy:      h.workerSrv.DbOk(r.Context()),
	}
	h.lock.Lock()
	h.lastResponse = ans
	h.lastCache = time.Now().UTC()
	h.lock.Unlock()
	return JSON(w, http.StatusOK, ans)
}
