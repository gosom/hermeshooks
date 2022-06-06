package entities

import (
	"time"

	"github.com/google/uuid"
)

type ScheduledJobStatus int

const (
	Undefined ScheduledJobStatus = iota
	Scheduled
	Pending
	Success
	Fail
	Deleted
)

func (s ScheduledJobStatus) String() string {
	switch s {
	case Undefined:
		return "undefined"
	case Scheduled:
		return "scheduled"
	case Pending:
		return "pending"
	case Success:
		return "success"
	case Fail:
		return "fail"
	case Deleted:
		return "deleted"
	}
	return "unknown"
}

type ScheduledJob struct {
	ID          int64
	UID         uuid.UUID
	Name        string
	Description string
	Url         string
	Payload     string
	ContentType string
	Signature   string
	RunAt       time.Time
	Retries     int
	Status      ScheduledJobStatus
	Partition   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
