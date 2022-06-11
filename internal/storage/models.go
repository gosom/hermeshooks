package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/uptrace/bun"
)

type ScheduledJob struct {
	bun.BaseModel

	ID          int64 `bun:"id,pk,autoincrement"`
	UID         uuid.UUID
	Name        string
	Description string
	Url         string
	Payload     string
	ContentType string
	Signature   string
	RunAt       time.Time
	Retries     int
	Status      int
	Partition   int
	CreatedAt   time.Time
	UpdatedAt   bun.NullTime
}

func FromScheduledJobEntity(j entities.ScheduledJob) ScheduledJob {
	ans := ScheduledJob{
		ID:          j.ID,
		UID:         j.UID,
		Name:        j.Name,
		Description: j.Description,
		Url:         j.Url,
		Payload:     j.Payload,
		ContentType: j.ContentType,
		Signature:   j.Signature,
		RunAt:       j.RunAt,
		Retries:     j.Retries,
		Status:      int(j.Status),
		Partition:   j.Partition,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   bun.NullTime{Time: j.UpdatedAt},
	}
	return ans
}

func ToScheduledJobEntity(j ScheduledJob) entities.ScheduledJob {
	ans := entities.ScheduledJob{
		ID:          j.ID,
		UID:         j.UID,
		Name:        j.Name,
		Description: j.Description,
		Url:         j.Url,
		Payload:     j.Payload,
		ContentType: j.ContentType,
		Signature:   j.Signature,
		RunAt:       j.RunAt,
		Retries:     j.Retries,
		Status:      entities.ScheduledJobStatus(j.Status),
		Partition:   j.Partition,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt.Time,
	}
	return ans
}

type Execution struct {
	bun.BaseModel

	ID             int64 `bun:"id,pk,autoincrement"`
	ScheduledJobID int64 `bun:"scheduled_job_id"`
	StatusCode     int
	Msg            string
	CreatedAt      time.Time
}

func FromEntitiesExecution(e entities.Execution) Execution {
	ans := Execution{
		ID:             e.ID,
		ScheduledJobID: e.ScheduledJobID,
		StatusCode:     e.StatusCode,
		Msg:            e.Msg,
		CreatedAt:      e.CreatedAt,
	}
	return ans
}

func ToEntitiesExecution(e Execution) entities.Execution {
	ans := entities.Execution{
		ID:             e.ID,
		ScheduledJobID: e.ScheduledJobID,
		StatusCode:     e.StatusCode,
		Msg:            e.Msg,
		CreatedAt:      e.CreatedAt,
	}
	return ans
}

type User struct {
	bun.BaseModel
	entities.User
}

func FromEntitiesUser(u entities.User) User {
	ans := User{
		User: u,
	}
	return ans
}

func ToEntitiesUser(u User) entities.User {
	return u.User
}
