package rest

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type MetaResponse struct {
	PublicKey string `json:"publicKey"`
}

type MetaHandler struct {
	log    zerolog.Logger
	pubKey string
}

func (h *MetaHandler) Get(w http.ResponseWriter, r bunrouter.Request) error {
	ans := MetaResponse{
		PublicKey: h.pubKey,
	}
	return JSON(w, http.StatusOK, ans)
}
