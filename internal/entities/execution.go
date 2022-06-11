package entities

import "time"

type Execution struct {
	ID             int64
	ScheduledJobID int64
	StatusCode     int
	Msg            string
	CreatedAt      time.Time
}
