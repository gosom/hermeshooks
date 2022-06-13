package rest

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type UserHandler struct {
	log zerolog.Logger
	srv AuthService
}

type SignupPayload struct {
	Username string
}

type SignupResponse struct {
	Apikey string `json:"apiKey"`
}

func (o SignupPayload) Validate() error {
	if len(o.Username) == 0 {
		return ValidationError{"username is mandatory"}
	}
	if len(o.Username) > 100 {
		return ValidationError{"username cannot be more than 100 characters"}
	}
	return nil
}

func (h *UserHandler) Create(w http.ResponseWriter, r bunrouter.Request) error {
	var p SignupPayload
	if err := Bind(r, &p); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	apiKey, err := h.srv.Signup(r.Context(), p.Username)
	if err != nil {
		return err
	}
	ans := SignupResponse{
		Apikey: apiKey,
	}
	return JSON(w, http.StatusCreated, ans)
}
