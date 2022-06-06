package entities

import (
	"context"
	"time"
)

type WorkerMeta struct {
	Name            string
	Partition       int
	RegisteredAt    time.Time
	LastHealthCheck time.Time
	Healthy         bool
	CancelFunc      context.CancelFunc
}
