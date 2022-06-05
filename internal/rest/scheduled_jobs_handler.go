package rest

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"

	"github.com/gosom/hermeshooks/internal/entities"
)

// TODO validate
type ScheduledJobsPayload struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	Payload     string    `json:"payload"`
	ContentType string    `json:"contentType"`
	Signature   string    `json:"signature"`
	RunAt       time.Time `json:"runAt"`
	Retries     int       `json:"retries"`
}

func Validate() error {
	return nil
}

func ToScheduledJob(p ScheduledJobsPayload) entities.ScheduledJob {
	ans := entities.ScheduledJob{
		UID:         uuid.New(),
		Name:        p.Name,
		Description: p.Description,
		Url:         p.Url,
		Payload:     p.Payload,
		ContentType: p.ContentType,
		Signature:   p.Signature,
		RunAt:       p.RunAt,
		Retries:     p.Retries,
		Status:      entities.Scheduled,
		CreatedAt:   time.Now().UTC(),
	}
	return ans
}

type ScheduledJobResponse struct {
	UUID string `json:"uuid"`
}

type ScheduledJobsHandler struct {
	log zerolog.Logger
	srv ScheduledJobService
}

func (h *ScheduledJobsHandler) Create(w http.ResponseWriter, r bunrouter.Request) error {
	var p ScheduledJobsPayload
	if err := Bind(r, &p); err != nil {
		return err
	}
	job := ToScheduledJob(p)
	job, err := h.srv.Schedule(r.Context(), job)
	if err != nil {
		return err
	}
	resp := ScheduledJobResponse{job.UID.String()}
	return JSON(w, http.StatusCreated, resp)
}
