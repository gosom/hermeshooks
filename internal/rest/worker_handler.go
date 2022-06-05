package rest

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type WorkerHandler struct {
	log       zerolog.Logger
	workerSrv WorkerService
}

type RegisterPayload struct {
	Name string `json:"name"`
}

type RegisterResponse struct {
	Partition int `json:"partition"`
}

func (h *WorkerHandler) Register(w http.ResponseWriter, r bunrouter.Request) error {
	var p RegisterPayload
	if err := Bind(r, &p); err != nil {
		return err
	}
	worker, err := h.workerSrv.Register(r.Context(), p.Name)
	if err != nil {
		return err
	}
	resp := RegisterResponse{
		Partition: worker.Partition,
	}
	return JSON(w, http.StatusOK, resp)
}
