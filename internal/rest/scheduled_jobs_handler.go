package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/entities"
)

var supportedContentTypes map[string]bool = map[string]bool{
	"":                 true,
	"application/json": true,
	"text/plain":       true,
}

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

func (s ScheduledJobsPayload) Validate() error {
	if len(s.Name) == 0 {
		return ValidationError{"name is mandatory"}
	}
	if len(s.Name) > 32 {
		return ValidationError{"name cannot be more that 32 characters"}
	}
	if len(s.Description) > 100 {
		return ValidationError{"description cannot be more than 100 characters"}
	}
	if _, err := url.ParseRequestURI(s.Url); err != nil {
		return ValidationError{err.Error()}
	}
	n := len([]byte(s.Payload))
	kbSize := n >> 10
	if kbSize > 2048 {
		return ValidationError{"payload must be at most 2048Kb"}
	}
	if !supportedContentTypes[s.ContentType] {
		supported := make([]string, 0, len(supportedContentTypes))
		for k, _ := range supportedContentTypes {
			supported = append(supported, k)
		}
		return ValidationError{
			"unsupported content-type. Use one of: " + strings.Join(supported, ","),
		}
	}
	if len(s.Signature) > 64 {
		return ValidationError{"signature can be at most 64 characters"}
	}
	now := time.Now().UTC().Add(3 * time.Minute)
	if s.RunAt.UTC().Before(now) {
		msg := fmt.Sprintf("RunAt must be at least 3 minutes from now")
		return ValidationError{msg}
	}
	future := time.Now().UTC().Add(24 * time.Hour * 30)
	if s.RunAt.UTC().After(future) {
		msg := fmt.Sprintf("RunAt must be at most 30 days from now")
		return ValidationError{msg}
	}
	if s.Retries > 3 {
		return ValidationError{
			"retries can be at most 3",
		}
	}
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
	if err := p.Validate(); err != nil {
		return err
	}
	job := ToScheduledJob(p)
	currentUser, err := common.GetCurrentUser(r)
	if err != nil {
		return err
	}
	job.UserID = currentUser.ID
	job, err = h.srv.Schedule(r.Context(), job)
	if err != nil {
		return err
	}
	resp := ScheduledJobResponse{job.UID.String()}
	return JSON(w, http.StatusCreated, resp)
}

type ScheduledJobGetResponse struct {
	UID         uuid.UUID           `json:"uid"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Url         string              `json:"url"`
	RunAt       time.Time           `json:"runAt"`
	Status      string              `json:"status"`
	Executions  []ExecutionResponse `json:"executions"`
}

type ExecutionResponse struct {
	StatusCode int       `json:"statusCode"`
	Msg        string    `json:"msg"`
	ExecutedAt time.Time `json:"executedAt"`
}

func (h *ScheduledJobsHandler) Get(w http.ResponseWriter, r bunrouter.Request) error {
	uid := r.Param("uuid")
	if len(uid) == 0 {
		return ValidationError{"uuid is missing"}
	}
	id, err := uuid.Parse(uid)
	if err != nil {
		return ValidationError{"not a valid uuid"}
	}
	currentUser, err := common.GetCurrentUser(r)
	if err != nil {
		return err
	}
	job, executions, err := h.srv.Get(r.Context(), currentUser, id.String())
	if err != nil {
		return ErrNotFound
	}
	ans := ScheduledJobGetResponse{
		UID:         job.UID,
		Name:        job.Name,
		Description: job.Description,
		Url:         job.Url,
		RunAt:       job.RunAt,
		Status:      job.Status.String(),
		Executions:  []ExecutionResponse{},
	}
	for i := range executions {
		ans.Executions = append(ans.Executions,
			ExecutionResponse{
				StatusCode: executions[i].StatusCode,
				Msg:        executions[i].Msg,
				ExecutedAt: executions[i].CreatedAt,
			},
		)
	}

	return JSON(w, http.StatusOK, ans)
}
